package ast

import(
	"github.com/cptaffe/lang/token"
	"strconv"
	"fmt"
	"errors"
)

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

// Walk down tree
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
		return fmt.Sprintf("%s", node.Var)
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
