package ast

import (
	"../token"
)

type Tree struct {
	Val *Node
	Sub []*Tree
}

type Node struct {
	Tok token.Token
}

func (tree *Tree) Append(node *Node) *Tree {
	tree.Sub = append(tree.Sub, &Tree{
		Val: node,
	})
	return tree.Sub[len(tree.Sub)-1]
}

func (tree *Tree) String() string {
	s := ""
	if tree.Val != nil {
		s += tree.Val.String()
	}
	s += "{"
	for i := 0; i < len(tree.Sub); i++ {
		s += tree.Sub[i].String() + ","
	}
	s += "}"
	return s
}

func (node *Node) String() string {
	return node.Tok.String()
}
