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
	Num    int            // if it is an int
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
		return "\"" + strconv.Itoa(node.Num) + "\""
	case ItemVar:
		if node.Solved {
			return "(" + node.Var + ":" + variabs.getName(node.Var).String() + ")"
		} else {
			return "(" + node.Var + ")"
		}
	case ItemKey:
		if node.Key == token.ItemLambda {
			return "lambda"
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
	level     int
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
		num, err := strconv.Atoi(t.Val.Tok.Val)
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

type ValueType int

// Multiple types of variables
const (
	ValueFunc ValueType = iota // function
	ValueNum                   // integer
)

type Value struct {
	Typ  ValueType
	Tree *Tree // deferred evaluation on function tree
	Num  int   // integer value for numbers
}

type Var struct {
	Var string // variable name
	Num *Value // integer value
}

type Variab struct {
	Var []*Var // array of Vars to create a list of variables
}

var variabs = new(Variab) // global list of variables

func (v Variab) getName(s string) *Var {
	for i := 0; i < len(v.Var); i++ {
		if v.Var[i].Var == s {
			return v.Var[i]
		}
	}
	return nil
}

func (v *Var) String() string {
	if v.Num.Typ == ValueFunc {
		return v.Num.Tree.String()
		//return v.Num.Tree.String()
	} else if v.Num.Typ == ValueNum {
		return strconv.Itoa(v.Num.Num)
	} else {
		return "unk"
	}
}

// evaluate does all the maths it can
func (e *evals) evaluate(t *Tree, v *Variab) *Tree {
	// evaluate valueless trees that contain children
	if t.Val == nil {
		for i := 0; i < len(t.Sub); i++ {
			tree := e.evaluate(t.Sub[i], v)
			if tree != nil {
				t.Sub[i] = tree
			}
		}
		// evaluate keys
	} else if t.Val.Typ == ItemKey {
		// if the key is an assignment key (special case)
		// call variables to compute value
		if t.Val.Key == token.ItemAssign || t.Val.Key == token.ItemFunction {
			return e.variables(t, v)
			// evaluate lambda function calls
		} else if t.Val.Key == token.ItemLambda {
			tree := evalLambda(t, e, v)
			return tree
		}
		// there are keys/vars that need to be computed,
		// recurse to compute them
		if hasSomeKeyChildren(t) || hasSomeVarChildren(t) {
			for i := 0; i < len(t.Sub); i++ {
				// recurse on key
				if t.Sub[i].Val.Typ == ItemKey {
					tree := e.evaluate(t.Sub[i], v)
					if tree != nil {
						t.Sub[i] = tree
					}
					// recurse on var
				} else if t.Sub[i].Val.Typ == ItemVar {
					tree := e.evaluate(t.Sub[i], v)
					if tree != nil {
						t.Sub[i] = tree
					}
				}
			}
		}
		// At this point there should only be ints,
		// if so, compute
		if hasOnlyIntChildren(t) {
			if val, ok := lookup[t.Val.Key]; ok {
				return val(t)
			} else {
				return nil
			}
		}
		// evaluate Lambda if one is called
	} else if t.Val.Typ == ItemVar {
		return e.variables(t, v)
	}
	return nil
}

// Variables is called when (1) there is an assignment key
// (2) there is a variable
func (e *evals) variables(t *Tree, v *Variab) *Tree {
	// get variable from record
	if t.Val.Typ == ItemVar {
		// t.Val.Var is in the list of found variables
		va := v.getName(t.Val.Var)
		if va != nil && va.Num.Typ == ValueNum {
			// return the tree, which will be appended by its parent in its place
			return &Tree{
				Val: &Node{
					Typ:    ItemVar,
					Num:    va.Num.Num,
					Var:    va.Var,
					Solved: true,
				},
			}
		} else if va.Num.Typ == ValueFunc {
			fmt.Printf("%s is not a lambda variable.\n", t.Val.Var)
		} else {
			fmt.Printf("%s is not a variable.\n", t.Val.Var)
		}
		// evaluate assignment keys
	} else if t.Val.Key == token.ItemAssign {
		// itemAssign is the assignment operator, look for variables
		// assignment must have two operators, a variable & some assigned value (can be a key)
		if len(t.Sub) == 2 && t.Sub[0].Val.Typ == ItemVar {
			// evaluate the key
			if t.Sub[1].Val.Typ == ItemKey {
				tree := e.evaluate(t.Sub[1], v)
				if tree != nil {
					t.Sub[1] = tree
				}
				// evaluate variable
			} else if t.Sub[1].Val.Typ == ItemVar {
				// replace var with var's value node if the var is known
				tree := e.variables(t.Sub[1], v)
				if tree != nil {
					t.Sub[1] = tree
				}
				return e.variables(t, v)
			}
			// should be int at this point
			if t.Sub[1].Val.Typ == ItemInt {
				// check if just reassigning existing variable
				va := v.getName(t.Sub[0].Val.Var)
				if va != nil {
					va.Num = &Value{
						Typ: ValueNum,
						Num: t.Sub[1].Val.Num,
					}
					// new variable creation
				} else {
					v.Var = append(v.Var, &Var{
						Var: t.Sub[0].Val.Var,
						Num: &Value{
							Typ: ValueNum,
							Num: t.Sub[1].Val.Num,
						},
					})
				}
				// return tree
				return &Tree{
					Val: &Node{
						Typ:    ItemVar,
						Num:    t.Sub[1].Val.Num,
						Var:    t.Sub[0].Val.Var,
						Solved: true,
					},
				}
				return nil
				// function lambda variable
			} else if t.Sub[1].Val.Key == token.ItemFunction {
				// check if just reassigning existing variable
				va := v.getName(t.Sub[0].Val.Var)
				if va != nil {
					va.Num = &Value{
						Typ:  ValueFunc,
						Tree: t.Sub[1],
					}
					// new variable creation
				} else {
					v.Var = append(v.Var, &Var{
						Var: t.Sub[0].Val.Var,
						Num: &Value{
							Typ:  ValueFunc,
							Tree: t.Sub[1],
						},
					})
				}
				// return tree
				return &Tree{
					Val: &Node{
						Typ:    ItemVar,
						Var:    t.Sub[0].Val.Var,
						Solved: true,
					},
				}
			}
		} else {
			fmt.Printf("Incorrect usage of assignment operator.")
			return nil
		}
	}
	return nil
}

// lambda takes a lambda tree: variable keyword with args
func (e *evals) lambda(t *Tree, v *Variab) *Tree {
	if t.Val.Typ == ItemKey && t.Val.Key == token.ItemLambda {
		// check if lambda is defined
		if va := v.getName(t.Val.Var); va != nil {
			// check number of args
			args := va.Num.Tree.Sub[0].Sub
			variab := new(Variab)
			if len(t.Sub) == len(args) {
				// add variables to Variab
				for i := 0; i < len(args); i++ {
					variab.Var = append(variab.Var, &Var{
						Var: args[i].Val.Var,
						Num: &Value{
							Typ: ValueNum,
							Num: t.Sub[i].Val.Num,
						},
					})
				}
				tree := e.evaluate(va.Num.Tree.Sub[1], variab)
				if tree != nil {
					return tree
				}
			} else {
				fmt.Printf("Not enough arguments: (%s) for (%s).\n", t.Sub, args)
			}
		} else {
			fmt.Printf("Undefined function\n")
		}
	}
	return nil
}

// evals
type eval func(t *Tree) *Tree

func evalLambda(t *Tree, e *evals, v *Variab) *Tree {
	tree := e.lambda(t, v)
	return tree
}

func evalAdd(t *Tree) *Tree {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n += t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}
	//fmt.Printf("add %s = %d\n", num, n)
}

func evalSub(t *Tree) *Tree {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n -= t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}
	//fmt.Printf("subtract %s = %d\n", num, n)
}

func evalMul(t *Tree) *Tree {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n *= t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}
	//fmt.Printf("multiply %s = %d\n", num, n)
}

func evalDiv(t *Tree) *Tree {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n /= t.Sub[i].Val.Num
	}
	return &Tree{
		Val: &Node{
			Typ: ItemInt,
			Num: n,
		},
	}
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

func hasOnlyIntChildren(tree *Tree) bool {
	if tree.Sub != nil && len(tree.Sub) > 0 {
		for i := 0; i < len(tree.Sub); i++ {
			if tree.Sub[i].Val.Typ != ItemInt && !(tree.Sub[i].Val.Typ == ItemVar && tree.Sub[i].Val.Solved == true) {
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
