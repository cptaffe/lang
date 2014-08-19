// This is the less generic tree, it currently only supports
// integers, keys (unevaluated operators), and variables (unevaluated constants)

// TODO: Add better errors, current ones sorta suck.

package optim

import (
	"errors"
	"fmt"
	"github.com/cptaffe/lang/parser"
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/ast"
	"github.com/cptaffe/lang/variable"
	"time"
	"bufio"
	"os"
)

var Variabs = new(variable.Variab) // global list of variables

type evals struct {
	Root      *ast.Tree        // root of tree
	Tree      *ast.Tree        // current branch
}

func Eval(tree *ast.Tree) *ast.Tree {
	e := &evals{
		Root:      tree,
		Tree:      tree,
	}
	e.evaluate(e.Root, Variabs)
	return e.Root
}

var lookup = map[token.ItemType]eval{
	token.ItemAdd: evalAdd,
	token.ItemSub: evalSub,
	token.ItemMul: evalMul,
	token.ItemDiv: evalDiv,
	token.ItemEq:  evalEq,
	token.ItemLt: evalLt,
	token.ItemTime: evalTime,
	token.ItemScan: evalScan,
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// print error message
	fmt.Printf("\033[1m%s: \033[31merror:\033[0m\033[1m %s\033[0m\n", "optim", msg)
}

// evaluate does all the maths it can
func (e *evals) evaluate(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// kill nils
	if t == nil {
		return t
	}
	// evaluate valueless trees that contain children
	if t.Val == nil {
		if t.Sub == nil {
			errorf("no values in tree")
			return nil
		}
		return e.evaluateSubs(t, v)
	} else if t.Val.Typ == ast.ItemKey {
		tr := e.keys(t, v)
		return tr
	} else if t.Val.Typ == ast.ItemVar {
		tr := e.variables(t, v)
		if tr != nil{
			trs := e.evaluate(tr, v)
			if trs != nil {
				return trs
			} else {
				return tr
			}
		} else {
			return tr // nil
		}
	}
	return t
}

// Keys
func (e *evals) keys(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// kill nils
	if t == nil {
		return t
	}
	// special tokens
	if t.Val.Typ == ast.ItemKey {
		switch {
		case t.Val.Key == token.ItemAssign || t.Val.Key == token.ItemFunction:
			tree := e.variables(t, v)
			if tree != nil {
				return tree
			}
		case t.Val.Key == token.ItemCmp:
			tree := e.compare(t, v)
			return tree
		case t.Val.Key == token.ItemLazy:
			v.Lazy = true
			tree := e.evaluate(t.Sub[0], v)
			v.Lazy = false
			if tree != nil {
				return tree
			}
			errorf("cannot evaluate: %s", t)
			return nil
		case t.Val.Key == token.ItemEval:
			ta := e.evaluateSubs(t, v)
			tr, err := evalEval(ta)
			if err != nil || tr == nil {
				errorf("cannot evaluate: %s", ta)
				return nil
			}
			return tr
		case t.Val.Key == token.ItemList:
			return e.evaluateSubs(t, v) // lists evaluate to themselves
		case t.Val.Key == token.ItemLambda:
			return evalLambda(e.evaluateSubs(t, v), e, v)
		case t.Val.Key == token.ItemPrint:
			t := e.evaluateSubs(t, v)
			//t , _ = evalPrint(t)
		default:
			// Compute math
			trs := e.evaluateSubs(t, v)
			if trs != nil {
				val, ok := lookup[t.Val.Key]
				if ok && OnlyNums(t) {
					result, err := val(trs)
					if err == nil {
						return result
					}
					errorf("%s: %s", err, trs)
				}
			} else {
				return nil
			}
		}
	}
	return t
}

func OnlyNums(t *ast.Tree) bool {
	for _, j := range(t.Sub) {
		if j.Val.Typ != ast.ItemNum {
			return false
		}
	}
	return true
}

// Evaluate Subs, so simple.
func (e *evals) evaluateSubs(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// Evaluate subs
	if t.Sub != nil {
		for i := 0; i < len(t.Sub); i++ {
			tree := e.evaluate(t.Sub[i], v)
			if tree != nil {
				t.Sub[i] = tree
			}
		}
	}
	return t
}

// Variables is called when (1) there is an assignment key
// (2) there is a variable
func (e *evals) variables(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// get variable from record
	if t.Val.Typ == ast.ItemVar {
		va := v.GetName(t.Val.Var)
		if va != nil && va.Tree != nil {
			// evaluate here, RETURN VALUE, NOT POINTER
			tr := new(ast.Tree)
			tr = ast.CopyTree(va.Tree, tr)
			return tr
		} else {
			errorf("undefined variable %s", t)
			return nil
		}
		// evaluate assignment keys
	} else if t.Val.Key == token.ItemAssign {
		// itemAssign is the assignment operator
		if len(t.Sub) == 2 && t.Sub[0].Val.Typ == ast.ItemVar {

			// lazy evaluation, ie. do not eval
			tree := t.Sub[1]
			name := t.Sub[0].Val.Var

			// check if just reassigning existing variable
			va := v.GetName(name)
			if v.Lazy {
				tr := e.evaluate(tree, v)
				if tr != nil{
					tree = tr
				}
			}
			if va != nil {
				va.Tree = tree
				// new variable creation
			} else {
				val := &variable.Var{
						Var:  name,
						Tree: tree,
					}
				if v.Scope != nil {
					v.Scope = append(v.Scope, val)
				} else {
					v.Var = append(v.Var, val)
				}
			}
			// return tree
			return &ast.Tree{
				Val: &ast.Node{
					Typ: ast.ItemVar,
					Var: name,
					VarTree: tree,
				},
			}
		}
	}
	return nil
}

// lambda takes a lambda tree: variable keyword with args
func (e *evals) lambda(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// check if lambda is defined
	va := e.variables(&ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemVar,
			Var: t.Val.Var,
		},
	}, v)
	if va != nil {
		scope := new(variable.Variab)
		scope.Var = v.Var // don't copy any scope
		//fmt.Printf("%s\n",t)
		args:= va.Sub[0].Sub;
		if len(t.Sub) == len(args) {
			// add evaluated variables to scope
			for i := 0; i < len(args); i++ {
				// create new scope
				scope.Scope = append(scope.Scope, &variable.Var{
					Var:  args[i].Val.Var,
					Tree: t.Sub[i],
				})
			}
			// the program reaches here, but does not return when it delves deep enough into recursion.
			// ex: (assign factorial (lambda (list n) (cmp n 1 (mul n (factorial (sub n 1))))))
			// calling (factorial 3) will cause this issue.
			//fmt.Printf(".")
			tr := e.evaluate(va.Sub[1], scope)
			if tr != nil {
				return tr
			}
		} else {
			errorf("%s: wrong number of args: %d for %d", t.Val.Var, len(t.Sub), len(args))
			return nil
		}
	} // undefined lambda
	return nil
}

// compare conditionally evaluates the second parameter pending the first
func (e *evals) compare(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// kill nils
	if t == nil {
		return t
	}
	// test value of first param
	if len(t.Sub) != 3 {
		return nil
	}
	tree := e.evaluate(t.Sub[0], v)
	if tree.Val.Num == 1 {
		tree := e.evaluate(t.Sub[1], v)
		return tree
	} else {
		tree := e.evaluate(t.Sub[2], v)
		return tree
	}
	return nil
}

// evals
type eval func(t *ast.Tree) (*ast.Tree, error)

func evalLambda(t *ast.Tree, e *evals, v *variable.Variab) *ast.Tree {
	tree := e.lambda(t, v)
	return tree
}

func evalEq(t *ast.Tree) (*ast.Tree, error) {
	if len(t.Sub) != 2 {
		return nil, errors.New("eq takes 2 atoms")
	}

	var n float64 = 0
	if t.Sub[0].Val.Num == t.Sub[1].Val.Num {
		n = 1
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}, nil
}

func evalLt(t *ast.Tree) (*ast.Tree, error) {
	if len(t.Sub) != 2 {
		return nil, errors.New("eq takes 2 atoms")
	}

	var n float64 = 0
	if t.Sub[0].Val.Num < t.Sub[1].Val.Num {
		n = 1
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}, nil
}

// returns time in nanoseconds
func evalTime(t *ast.Tree) (*ast.Tree, error) {
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: float64((time.Now()).Nanosecond()),
		},
	}, nil
}

// prints a tree
func evalPrint(t *ast.Tree) (*ast.Tree, error) {
	for i := 0; i < len(t.Sub); i++ {
		fmt.Printf("%s", t.Sub[i])
	}
	fmt.Printf("\n")
	return t, nil
}

// scans a line

func readFile() string {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	fmt.Printf("%s", line)
	if err != nil {
		// handle error :)
	}
	return string(line)
}

func evalScan(t *ast.Tree) (*ast.Tree, error) {
	for i := 0; i < len(t.Sub); i++ {
		fmt.Printf("%s", t.Sub[i])
	}
	str := readFile()
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemString,
			Str: str,
		},
	}, nil
}

//lex/parses into a tree
func evalEval(t *ast.Tree) (*ast.Tree, error) {
	tree := parser.Parse(t.Sub[0].Val.Str, "eval")
	tr := Eval(tree)
	return tr, nil
}

func evalAdd(t *ast.Tree) (*ast.Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n += t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}, nil
}


func evalSub(t *ast.Tree) (*ast.Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n -= t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}, nil
}

func evalMul(t *ast.Tree) (*ast.Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n *= t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}, nil
}

func evalDiv(t *ast.Tree) (*ast.Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n /= t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}, nil
}
