// This is the less generic tree, it currently only supports
// integers, keys (unevaluated operators), and variables (unevaluated constants)

// TODO: Add better errors, current ones sorta suck.

package optim

import (
	"errors"
	"fmt"
	"github.com/cptaffe/lang/parser"
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/lexer"
	"github.com/cptaffe/lang/ast"
	"time"
	"bufio"
	"os"
	//"io"
)

type evals struct {
	Root      *ast.Tree        // root of tree
	Tree      *ast.Tree        // current branch
	ParseRoot *parser.Tree // root of parse tree
	ParseTree *parser.Tree // root of parse tree
}

func Eval(tree *parser.Tree) *ast.Tree {
	t := new(ast.Tree)
	e := &evals{
		Root:      t,
		Tree:      t,
		ParseRoot: tree,
		ParseTree: tree,
	}
	ast.CreateFromParse(e.ParseRoot, e.Root)
	e.evaluate(e.Root, ast.Variabs)
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
	token.ItemPrint: evalPrint,
	token.ItemScan: evalScan,
}

// evaluate does all the maths it can
func (e *evals) evaluate(t *ast.Tree, v *ast.Variab) *ast.Tree {
	//fmt.Printf("evaluate: %s\n", t)
	// evaluate valueless trees that contain children
	if t.Val == nil {
		if t.Sub == nil {
			return nil
		}
		return e.evaluateSubs(t, v)
		// evaluate keys
	} else if t.Val.Typ == ast.ItemKey {
		tree := e.keys(t, v)
		return tree
	} else if t.Val.Typ == ast.ItemVar {
		tree := e.evaluate(e.variables(t, v), v)
		return tree
	}
	return t
}

// Keys
func (e *evals) keys(t *ast.Tree, v *ast.Variab) *ast.Tree {
	//fmt.Printf("keys: %s\n", t)
	// special tokens
	if t.Val.Typ == ast.ItemKey {
		switch {
		case t.Val.Key == token.ItemAssign || t.Val.Key == token.ItemFunction:
			tree := e.variables(t, v)
			return tree
		case t.Val.Key == token.ItemCmp:
			tree := e.compare(t, v)
			return tree
		case t.Val.Key == token.ItemLazy:
			v.Lazy = true
			tree := e.evaluate(t.Sub[0], v)
			v.Lazy = false
			return tree
		case t.Val.Key == token.ItemEval:
			ta := e.evaluateSubs(t, v)
			tr, err := evalEval(ta)
			if err != nil || tr == nil {
				return nil
			}
			return tr
		case t.Val.Key == token.ItemList:
			return e.evaluateSubs(t, v) // lists evaluate to themselves
		case t.Val.Key == token.ItemLambda:
			tree := evalLambda(e.evaluateSubs(t, v), e, v)
			return tree
		default:
			t := e.evaluateSubs(t, v)
			// Compute math
			if val, ok := lookup[t.Val.Key]; ok {
				result, err := val(t)
				if err != nil {
					return nil
				}
				return result
			}
		}
	}
	return nil
}

// Evaluate Subs, so simple.
func (e *evals) evaluateSubs(t *ast.Tree, v *ast.Variab) *ast.Tree {
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
func (e *evals) variables(t *ast.Tree, v *ast.Variab) *ast.Tree {
	//fmt.Printf("variables: %s\n", t)
	// get variable from record
	if t.Val.Typ == ast.ItemVar {
		// t.Val.Var is in the list of found variables
		va := v.GetName(t.Val.Var)
		if va != nil {
			// evaluate here, RETURN VALUE, NOT POINTER
			tr := new(ast.Tree)
			tr = ast.CopyTree(va.Tree, tr)
			//fmt.Printf("varaibles found: %s\n", tr)
			return tr
		} else {
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
				tree = e.evaluate(tree, v)
			}
			if va != nil {
				va.Tree = tree
				// new variable creation
			} else {
				variable := &ast.Var{
						Var:  name,
						Tree: tree,
					}
				if v.Scope != nil {
					v.Scope = append(v.Scope, variable)
				} else {
					v.Var = append(v.Var, variable)
				}
			}
			// return tree
			return &ast.Tree{
				Val: &ast.Node{
					Typ: ast.ItemVar,
					Var: name,
				},
			}
		}
	}
	return nil
}

// lambda takes a lambda tree: variable keyword with args
func (e *evals) lambda(t *ast.Tree, v *ast.Variab) *ast.Tree {
	//fmt.Printf("lambda: %s\n", t)
	// check if lambda is defined
	va := e.variables(&ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemVar,
			Var: t.Val.Var,
		},
	}, v)
	if va != nil {
		scope := new(ast.Variab)
		scope.Var = v.Var // don't copy any scope
		//fmt.Printf("%s\n",t)
		args:= va.Sub[0].Sub;
		if len(t.Sub) == len(args) {
			// add evaluated variables to scope
			for i := 0; i < len(args); i++ {
				// create new scope
				scope.Scope = append(scope.Scope, &ast.Var{
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
			fmt.Printf("Wrong number of arguments: (%s) for (%s).\n", t.Sub, args)
		}
	} else {
		fmt.Printf("undefined lambda!\n")
	} // undefined lambda
	return nil
}

// compare conditionally evaluates the second parameter pending the first
func (e *evals) compare(t *ast.Tree, v *ast.Variab) *ast.Tree {
	//fmt.Printf("compare: %s\n", t)
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

func evalLambda(t *ast.Tree, e *evals, v *ast.Variab) *ast.Tree {
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
	ch := lexer.Lex(t.Sub[0].Val.Str)
	done := make(chan *parser.Tree)
	parser.Parse(ch, done)
	tree := <-done
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
