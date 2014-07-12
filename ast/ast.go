package ast

import(
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/parser"
	"strconv"
	"log"
	"fmt"
)

type Var struct {
	Var  string // variable name
	Tree *Tree  // every variable stored as a tree
}

type Variab struct {
	Lazy bool // lazy assignment or not so much
	Var   []*Var // array of Vars to create a list of variables
	Scope []*Var // scope has precedence if not null
}

var Variabs = new(Variab) // global list of variables

func (v Variab) GetName(s string) *Var {
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

type ItemType int

// Types used in the Abstract Syntax Tree
// guidelines: keep types as generic as possible.
const (
	ItemNum ItemType = iota
	ItemString
	ItemVar
	ItemKey
)

// variable n-dimensional tree
type Tree struct {
	Val *Node
	Sub []*Tree
}

// 
type Node struct {
	Typ    ItemType
	Num    float64        // number type, float64 should handle this well.
	Str string // string type
	Var    string         // variable name
	Key    token.ItemType // keywords have an itemtype for identification
	Solved bool           // shows whether var is indexed or not
}

// Append adds a node to the Sub tree of the tree.
func (tree *Tree) Append(node *Node) *Tree {
	tree.Sub = append(tree.Sub, &Tree{
		Val: node,
	})
	return tree.Sub[len(tree.Sub)-1]
}

// Creates a tree from the parse tree
func CreateFromParse(t *parser.Tree, tr *Tree) {
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
			CreateFromParse(t.Sub[i], tr)
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
			CreateFromParse(t.Sub[i], tr)
		}
	} else if token.Constant(t.Val.Tok.Typ) {
		var node *Node
		switch {
		case t.Val.Tok.Typ == token.ItemNumber:
			num, err := strconv.ParseFloat(t.Val.Tok.Val, 64)
			if err != nil {
				log.Fatal(err)
			}
			node = &Node{
				Typ: ItemNum,
				Num: num,
			}
		case t.Val.Tok.Typ == token.ItemString:
			node = &Node{
				Typ: ItemString,
				Str: t.Val.Tok.Val[1:len(t.Val.Tok.Val)-1],
			}
		}
		tr.Append(node)
	} else if t.Val.Tok.Typ == token.ItemVariable {
		tr = tr.Append(&Node{
			Typ: ItemVar,
			Var: t.Val.Tok.Val,
		})
		for i := 0; i < len(t.Sub); i++ {
			CreateFromParse(t.Sub[i], tr)
		}
	}
}

// duplicate a tree to avoid mutating data
func CopyTree(t *Tree, tr *Tree) *Tree {
	if t.Val != nil {
		tr = &Tree{
			Val: &Node{
				Typ: t.Val.Typ, // int
				Num: t.Val.Num, // float64
				Str: t.Val.Str, // string
				Var: t.Val.Var, // string
				Key: t.Val.Key, // int
				Solved: t.Val.Solved,
			},
		}
	}
	if len(t.Sub) > 0 {
		for i := 0; i < len(t.Sub); i++ {
			tr.Sub = append(tr.Sub, new(Tree))
			tr.Sub[i] = CopyTree(t.Sub[i], tr.Sub[i])
		}
	}
	return tr
}

// String interfaces

func (tree *Tree) String() string {
	var s string
	if tree.Val != nil {
		s += tree.Val.String()
	}
	if len(tree.Sub) > 0 {
		for i := 0; i < len(tree.Sub); i++ {
			if i != len(tree.Sub)-1 {
				s += tree.Sub[i].String() + ", "
			} else {
				// shortens long lambda printing
				if tree.Val != nil && tree.Val.Key == token.ItemFunction {
					st := tree.Sub[i].String()
					if len(s) > 10 {
						s += fmt.Sprintf("%s...", st[:10])
					} else {
						s += fmt.Sprintf("%s", st)
					}
				} else {
					s += tree.Sub[i].String()
				}
			}
		}
		s = fmt.Sprintf("{%s}", s)
	}
	return s
}

func (node *Node) String() string {
	switch node.Typ {
	case ItemNum:
		return fmt.Sprintf("%s", strconv.FormatFloat(node.Num, 'g', -1, 64))
	case ItemVar:
		va := Variabs.GetName(node.Var)
		if va != nil {
			s := Variabs.GetName(node.Var).String()
			return fmt.Sprintf("(%s:%s)", node.Var, s)
		} else {
			return fmt.Sprintf("(%s)", node.Var)
		}
	case ItemKey:
		if node.Key == token.ItemLambda {
			return fmt.Sprintf("call(%s)", node.Var)
		}
		return token.StringLookup(node.Key)
	case ItemString:
		return node.Str
	default:
		return "unk"
	}
}
