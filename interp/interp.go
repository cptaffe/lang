package interp

import (
	"../ast"
	"../token"
	"fmt"
	"log"
	"strconv"
)

type exec func(num []int) int

var lookup = map[token.ItemType]exec{
	token.ItemAdd: execAdd,
	token.ItemSub: execSub,
	token.ItemMul: execMul,
}

func Exec(tree *ast.Tree) []int {
	var nums []int
	if (tree.Val == nil && hasActionChildren(tree)) || token.Keyword(tree.Val.Tok.Typ) {
		for i := 0; i < len(tree.Sub); i++ {
			if isAction(tree.Sub[i]) {
				nums = append(nums, Exec(tree.Sub[i])...)
			} else {
				num, err := strconv.Atoi(tree.Sub[i].Val.Tok.Val)
				if err != nil {
					log.Fatal(err)
				}
				nums = append(nums, num)
			}
		}
		if tree.Val == nil {
			return nums
		} else if val, ok := lookup[tree.Val.Tok.Typ]; ok {
			return []int{val(nums)}
		} else {
			fmt.Printf("unknown type: %s\n", tree.Val.Tok.Val)
			return nil
		}
	} else {
		return nil
	}
}

func hasActionChildren(tree *ast.Tree) bool {
	if tree.Sub != nil {
		for i := 0; i < len(tree.Sub); i++ {
			if token.Keyword(tree.Sub[i].Val.Tok.Typ) {
				return true
			}
		}
	}
	return false
}

func isAction(tree *ast.Tree) bool {
	if token.Keyword(tree.Val.Tok.Typ) {
		return true
	}
	return false
}

func intify(tree *ast.Tree) ([]int, error) {
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

// execs

func execAdd(num []int) int {
	n := num[0]
	for i := 1; i < len(num); i++ {
		n += num[i]
	}
	//fmt.Printf("add %s = %d\n", num, n)
	return n
}

func execSub(num []int) int {
	n := num[0]
	for i := 1; i < len(num); i++ {
		n -= num[i]
	}
	//fmt.Printf("subtract %s = %d\n", num, n)
	return n
}

func execMul(num []int) int {
	n := num[0]
	for i := 1; i < len(num); i++ {
		n *= num[i]
	}
	//fmt.Printf("multiply %s = %d\n", num, n)
	return n
}
