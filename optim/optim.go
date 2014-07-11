// This is the less generic tree, it currently only supports
// integers, keys (unevaluated operators), and variables (unevaluated constants)

package optim

import (
	"../parser"
	"../token"
	"errors"
	"fmt"
	"log"
	"strconv"
)

type ItemType int

const (
	ItemInt ItemType = iota
	ItemVar
	ItemKey
)

type Tree struct {
	Val *Node
	Sub []*Tree
}

type Node struct {
	Typ    ItemType
	Num    float64        // if it is an int
	Var    string         // if it is a variable
	Key    token.ItemType // deferred keyword
	Solved bool           // var is either assigned a value or unknown
}

func (tree *Tree) Append(node *Node) *Tree {
	tree.Sub = append(tree.Sub, &Tree{
		Val: node,
	})
	return tree.Sub[len(tree.Sub)-1]
}

func (tree *Tree) Walk(level int) (*Tree, error) {
	if level != 0 {
		if len(tree.Sub) > 0 {
			return tree.Sub[len(tree.Sub)-1].Walk(level - 1)
		} else {
			return nil, errors.New("level nonexistant")
		}
	} else {
		return tree, nil
	}
}

func (tree *Tree) String() string {
	var s string
	if tree.Val != nil {
		s += tree.Val.String()
	}
	if len(tree.Sub) > 0 {
		s += "{"
		for i := 0; i < len(tree.Sub); i++ {
			if i != len(tree.Sub)-1 {
				s += tree.Sub[i].String() + ", "
			} else {
				s += tree.Sub[i].String()
			}
		}
		s += "}"
	}
	return s
}

func (node *Node) String() string {
	switch node.Typ {
	case ItemInt:
		return "\"" + strconv.FormatFloat(node.Num, 'g', -1, 64) + "\""
	case ItemVar:
		va := variabs.getName(node.Var)
		if va != nil {
			return "(" + node.Var + ":" + variabs.getName(node.Var).String() + ")"
		} else {
			return "(" + node.Var + ")"
		}
	case ItemKey:
		if node.Key == token.ItemLambda {
			return "call(" + node.Var + ")"
		}
		return token.StringLookup(node.Key)
	default:
		return "unk"
	}
}

type evals struct {
	Root      *Tree        // root of tree
	Tree      *Tree        // current branch
	ParseRoot *parser.Tree // root of parse tree
	ParseTree *parser.Tree // root of parse tree
}

func Eval(tree *parser.Tree) *Tree {
	t := new(Tree)
	e := &evals{
		Root:      t,
		Tree:      t,
		ParseRoot: tree,
		ParseTree: tree,
	}
	e.createTree(e.ParseRoot, e.Root)
	e.evaluate(e.Root, variabs)
	return e.Root
}

var lookup = map[token.ItemType]eval{
	token.ItemAdd: evalAdd,
	token.ItemSub: evalSub,
	token.ItemMul: evalMul,
	token.ItemDiv: evalDiv,
	token.ItemEq:  evalEq,
}

// Creates a tree from the parse tree
func (e *evals) createTree(t *parser.Tree, tr *Tree) {
	// check fo' nills
	if t.Val == nil {
		if len(t.Sub) < 1 {
			return
		}
	} else {
		//fmt.Printf("%d\n", t.Val.Tok.Typ)
	}

	// We can do stuff
	if t.Val == nil && len(t.Sub) > 0 {
		for i := 0; i < len(t.Sub); i++ {
			e.createTree(t.Sub[i], tr)
		}
	} else if token.Keyword(t.Val.Tok.Typ) {
		if t.Val.Tok.Typ == token.ItemLambda {
			tr = tr.Append(&Node{
				Typ: ItemKey,
				Key: t.Val.Tok.Typ,
				Var: t.Val.Tok.Val,
			})
		} else {
			tr = tr.Append(&Node{
				Typ: ItemKey,
				Key: t.Val.Tok.Typ,
			})
		}
		for i := 0; i < len(t.Sub); i++ {
			e.createTree(t.Sub[i], tr)
		}
	} else if token.Constant(t.Val.Tok.Typ) {
		num, err := strconv.ParseFloat(t.Val.Tok.Val, 64)
		if err != nil {
			log.Fatal(err)
		}
		tr.Append(&Node{
			Typ: ItemInt,
			Num: num,
		})
	} else if t.Val.Tok.Typ == token.ItemVariable {
		tr = tr.Append(&Node{
			Typ: ItemVar,
			Var: t.Val.Tok.Val,
		})
		for i := 0; i < len(t.Sub); i++ {
			e.createTree(t.Sub[i], tr)
		}
	}
}

type Var struct {
	Var  string // variable name
	Tree *Tree  // every variable stored as a tree
}

type Variab struct {
	Var   []*Var // array of Vars to create a list of variables
	Scope []*Var // scope has precedence if not null
}

var variabs = new(Variab) // global list of variables

func (v Variab) getName(s string) *Var {
	// check scope first
	if v.Scope != nil {
		for i := 0; i < len(v.Scope); i++ {
			if v.Scope[i].Var == s {
				return v.Scope[i]
			}
		}
	}
	// now check var
	for i := 0; i < len(v.Var); i++ {
		if v.Var[i].Var == s {
			return v.Var[i]
		}
	}
	return nil
}

func (v *Var) String() string {
	return v.Tree.String()
}

// evaluate does all the maths it can
func (e *evals) evaluate(t *Tree, v *Variab) *Tree {
	//fmt.Printf("evaluate: %s\n", t)
	// evaluate valueless trees that contain children
	if t.Val == nil {
		for i := 0; i < len(t.Sub); i++ {
			tree := e.evaluate(t.Sub[i], v)
			if tree != nil {
				t.Sub[i] = tree
			}
		}
		return t
		// evaluate keys
	} else if t.Val.Typ == ItemKey {
		tree := e.keys(t, v)
		return tree
	} else if t.Val.Typ == ItemVar {
		tree := e.variables(t, v)
		return tree
	}
	return t
}

// Keys
func (e *evals) keys(t *Tree, v *Variab) *Tree {
	//fmt.Printf("keys: %s\n", t)
	// special tokens
	if t.Val.Typ == ItemKey {
		switch {
		case t.Val.Key == token.ItemAssign || t.Val.Key == token.ItemFunction:
			tree := e.variables(t, v)
			return tree
		case t.Val.Key == token.ItemCmp:
			tree := e.compare(t, v)
			return tree
		case t.Val.Key == token.ItemLambda:
			tree := evalLambda(t, e, v)
			return tree
		default:
			// Evaluate subs
			for i := 0; i < len(t.Sub); i++ {
				tree := e.evaluate(t.Sub[i], v)
				if tree != nil {
					t.Sub[i] = tree
				}
			}
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

// Variables is called when (1) there is an assignment key
// (2) there is a variable
func (e *evals) variables(t *Tree, v *Variab) *Tree {
	//fmt.Printf("variables: %s\n", t)
	// get variable from record
	if t.Val.Typ == ItemVar {
		// t.Val.Var is in the list of found variables
		va := v.getName(t.Val.Var)
		if va != nil {
			// evaluate here
			tree := e.evaluate(va.Tree, v)
			return tree
		}
		// evaluate assignment keys
	} else if t.Val.Key == token.ItemAssign {
		// itemAssign is the assignment operator
		if len(t.Sub) == 2 && t.Sub[0].Val.Typ == ItemVar {

			// lazy evaluation, ie. do not eval
			tree := t.Sub[1]
			if tree != nil {
				t.Sub[1] = tree
			}

			// check if just reassigning existing variable
			va := v.getName(t.Sub[0].Val.Var)
			if va != nil {
				va.Tree = t.Sub[1]
				// new variable creation
			} else {
				v.Var = append(v.Var, &Var{
					Var:  t.Sub[0].Val.Var,
					Tree: t.Sub[1],
				})
			}
			// return tree
			return &Tree{
				Val: &Node{
					Typ: ItemVar,
					Var: t.Sub[0].Val.Var,
				},
			}
		}
	}
	return nil
}

// lambda takes a lambda tree: variable keyword with args
func (e *evals) lambda(t *Tree, v *Variab) *Tree {
	//fmt.Printf("lambda: %s\n", t)
	// check if lambda is defined
	if va := v.getName(t.Val.Var); va != nil {
		args := va.Tree.Sub[0].Sub
		scope := new(Variab)
		scope.Var = v.Var // don't copy any scope
		if len(t.Sub) == len(args) {
			// add variables to scope
			for i := 0; i < len(args); i++ {
				// evaluate arguments
				tree := e.evaluate(t.Sub[i], v)
				if tree != nil {
					t.Sub[i] = tree
				}

				// create new scope
				scope.Scope = append(scope.Scope, &Var{
					Var:  args[i].Val.Var,
					Tree: t.Sub[i],
				})
			}
			// the program reaches here, but does not return when it delves deep enough into recursion.
			// ex: (assign factorial (lambda (list n) (cmp n 1 (mul n (factorial (sub n 1))))))
			// calling (factorial 3) will cause this issue.
			//fmt.Printf("We are here.\n")
			tr := e.evaluate(va.Tree.Sub[1], scope)
			//fmt.Printf("We are not here.\n")
			if tr != nil {
				return tr
			}
		} else {
			fmt.Printf("Not enough arguments: (%s) for (%s).\n", t.Sub, args)
		}
	} else {
		fmt.Printf("undefined lambda!\n")
	} // undefined lambda
	return nil
}

// compare conditionally evaluates the second parameter pending the first
func (e *evals) compare(t *Tree, v *Variab) *Tree {
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
type eval func(t *Tree) (*Tree, error)

func evalLambda(t *Tree, e *evals, v *Variab) *Tree {
	tree := e.lambda(t, v)
	return tree
}

func evalEq(t *Tree) (*Tree, error) {
	if len(t.Sub) != 2 {
		return nil, errors.New("eq takes 2 atoms")
	}
	if t.Sub[0].Val.Num == t.Sub[1].Val.Num {
		return &Tree{
			Val: &Node{
				Typ: ItemInt,
				Num: 1,
			},
		}, nil
	} else {
		return &Tree{
			Val: &Node{
				Typ: ItemInt,
				Num: 0,
			},
		}, nil
	}
	//fmt.Printf("add %s = %d\n", num, n)
}

func evalAdd(t *Tree) (*Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n += t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}, nil
	//fmt.Printf("add %s = %d\n", num, n)
}

func evalSub(t *Tree) (*Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n -= t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}, nil
	//fmt.Printf("subtract %s = %d\n", num, n)
}

func evalMul(t *Tree) (*Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n *= t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}, nil
	//fmt.Printf("multiply %s = %d\n", num, n)
}

func evalDiv(t *Tree) (*Tree, error) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n /= t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}, nil
	//fmt.Printf("divide %s = %d\n", num, n)
}

func hasActionChildren(tree *parser.Tree) bool {
	if tree.Sub != nil && len(tree.Sub) > 0 {
		for i := 0; i < len(tree.Sub); i++ {
			if token.Keyword(tree.Sub[i].Val.Tok.Typ) {
				return true
			}
		}
	}
	return false
}

func hasOnlyValuedChildren(tree *Tree, v *Variab) bool {
	if tree.Sub != nil && len(tree.Sub) > 0 {
		for i := 0; i < len(tree.Sub); i++ {
			if tree.Sub[i].Val.Typ == ItemVar {
				va := v.getName(tree.Sub[i].Val.Var)
				if va == nil {
					return false // unsolved variable
				}
			} else if tree.Sub[i].Val.Typ == ItemInt {
				// good to go
			} else {
				return false
			}
		}
		return true
	}
	return false
}

func hasSomeKeyChildren(tree *Tree) bool {
	if tree.Sub != nil && len(tree.Sub) > 0 {
		for i := 0; i < len(tree.Sub); i++ {
			if tree.Sub[i].Val.Typ == ItemKey {
				return true
			}
		}
	}
	return false
}

func hasSomeVarChildren(tree *Tree) bool {
	if tree.Sub != nil && len(tree.Sub) > 0 {
		for i := 0; i < len(tree.Sub); i++ {
			if tree.Sub[i].Val.Typ == ItemVar {
				return true
			}
		}
	}
	return false
}

func isAction(tree *parser.Tree) bool {
	if token.Keyword(tree.Val.Tok.Typ) {
		return true
	}
	return false
}

func intify(tree *parser.Tree) ([]int, error) {
	var num []int
	for i := 0; i < len(tree.Sub); i++ {
		switch tree.Sub[i].Val.Tok.Typ {
		case token.ItemNumber:
			n, err := strconv.Atoi(tree.Sub[i].Val.Tok.Val)
			if err != nil {
				return nil, err
			}
			num = append(num, n)
		}
	}
	return num, nil
}
