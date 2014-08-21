package optim

import (
	"fmt"
	//"github.com/cptaffe/lang/parser"
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/ast"
	"github.com/cptaffe/lang/variable"
)

type Scope variable.Scope

// error printing
func errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// print error message
	fmt.Printf("\033[1m%s: \033[31merror:\033[0m\033[1m %s\033[0m\n", "optim", msg)
}

// generate child scope
func (s *Scope) childScope() *Scope {
	scope := new(Scope)
	scope.Parent = (*variable.Scope)(s)
	return scope
}

// exported api
func Eval(tree *ast.Tree) *ast.Tree {
	scope := new(Scope) // scope
	return scope.evalChildren(tree) // evaluate in scope
}

// concurrently evaluates children
func (scope *Scope) evalChildren(tree *ast.Tree) *ast.Tree {
	for i := 0; i < len(tree.Sub); i++ {
		t := scope.eval(tree.Sub[i])
		if t != nil {
			tree.Sub[i] = t
		} else {
			errorf("eval fail: %s", tree.Sub[i])
		}
	}
	return tree
}

// eval 
func (scope *Scope) eval(tree *ast.Tree) *ast.Tree {
	errorf("in eval: %s", tree)
	if tree.Val.Typ == ast.ItemKey {
		return scope.evalKey(tree)
	} else if tree.Val.Typ == ast.ItemVar {
		return scope.evalVar(tree)
	} else if tree.Val.Typ == ast.ItemNum{
		return tree
	} else {
		return nil
	}
}

// evaluates keys
func (scope *Scope) evalKey(tree *ast.Tree) *ast.Tree {
	if tree.Val.Key == token.ItemAssign {
		return scope.evalAssign(tree)
	} else if tree.Val.Key == token.ItemFunction {
		return scope.evalFunc(tree)
	} else if tree.Val.Key == token.ItemLambda {
		return scope.evalLambda(tree)
	} else if tree.Val.Key == token.ItemCmp {
		return scope.evalCmp(tree)
	} else {
		t := scope.evalChildren(tree)
		if t != nil {
			val, ok := evalLookup[t.Val.Key]
			if ok && onlyNums(t) {
				return val(t)
			}
		} else {
			return nil
		}
	}
	return nil
}

func onlyNums(tree *ast.Tree) bool {
	num := true
	for i := 0; i < len(tree.Sub); i++ {
		if tree.Sub[0].Val.Typ != ast.ItemNum {
			num = false
			break
		}
	}
	return num
}

func (scope *Scope) evalVar(tree *ast.Tree) *ast.Tree {
	t := ((*variable.Scope)(scope)).GetName(tree.Val.Var)
	if t != nil && t.Tree != nil {
		errorf("var: %s", t.Tree)
		return ast.CopyTree(t.Tree, new(ast.Tree))
	} else {
		return nil
	}
}

func (scope *Scope) evalAssign(tree *ast.Tree) *ast.Tree {
	if len(tree.Sub) == 2 && tree.Sub[0].Val.Typ == ast.ItemVar {
		name := tree.Sub[0].Val.Var
		assig := ((*variable.Scope)(scope)).GetName(name)
		if assig != nil {
			assig.Tree = tree.Sub[1]
		} else {
			val := &variable.Var{
				Var: name,
				Tree: tree.Sub[1],
			}
			scope.Scope = append(scope.Scope, val)
		}
		// return tree
			return &ast.Tree{
				Val: &ast.Node{
					Typ: ast.ItemVar,
					Var: name,
					VarTree: tree.Sub[1],
				},
			}
	} else {
		errorf("incorrect assign syntax %s", tree)
		return nil
	}
}

// these may not exist, not sure...
func (scope *Scope) evalFunc(tree *ast.Tree) *ast.Tree {
	errorf("evalFunc")
	return nil
}

func (scope *Scope) evalLambda(tree *ast.Tree) *ast.Tree {
	// eval subs
	def := scope.evalVar(&ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemVar,
			Var: tree.Val.Var,

		},
	})
	if def != nil {
		sc := scope.childScope()
		args := def.Sub[0].Sub
		if len(tree.Sub) == len(args) {
			for i := 0; i < len(args); i++ {
				// populate scope
				sc.Scope = append(sc.Scope, &variable.Var{
					Var: args[i].Val.Var,
					Tree: scope.eval(tree.Sub[i]),
				})
			}
			tr := sc.eval(def.Sub[1])
			if tr != nil {
				return tr
			} else {
				return def.Sub[1]
			}
		} else {
			errorf("lambda: arg number incorrect")
			return nil
		}
	} else {
		errorf("undefined func")
		return nil
	}
}

func (scope *Scope) evalCmp(tree *ast.Tree) *ast.Tree {
	if len(tree.Sub) == 3 {
		t := scope.eval(tree.Sub[0])
		if t != nil {
			if t.Val.Typ == ast.ItemNum && t.Val.Num == 1 {
				return scope.eval(tree.Sub[1])
			} else {
				return scope.eval(tree.Sub[2])
			}
		} else {
			return nil
		}
	} else {
		errorf("cmp: arg number incorrect")
		return nil
	}
}

// stuff functions

type eval func(tree *ast.Tree) (*ast.Tree)

var evalLookup = map[token.ItemType]eval{
	token.ItemAdd: evalAdd,
	token.ItemSub: evalSub,
	token.ItemMul: evalMul,
	token.ItemDiv: evalDiv,
	token.ItemEq:  evalEq,
	token.ItemLt: evalLt,
}

func evalEq(t *ast.Tree) (*ast.Tree) {
	if len(t.Sub) != 2 {
		errorf("eq takes two atoms")
		return nil
	}
	var n int32 = 0
	if t.Sub[0].Val.Num == t.Sub[1].Val.Num {
		n = 1
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}
}

func evalLt(t *ast.Tree) (*ast.Tree) {
	if len(t.Sub) != 2 {
		errorf("lt takes two atoms")
		return nil
	}
	var n int32 = 0
	if t.Sub[0].Val.Num < t.Sub[1].Val.Num {
		n = 1
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}
}

func evalAdd(t *ast.Tree) (*ast.Tree) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n += t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}
}


func evalSub(t *ast.Tree) (*ast.Tree) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n -= t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}
}

func evalMul(t *ast.Tree) (*ast.Tree) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n *= t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}
}

func evalDiv(t *ast.Tree) (*ast.Tree) {
	n := t.Sub[0].Val.Num
	for i := 1; i < len(t.Sub); i++ {
		n /= t.Sub[i].Val.Num
	}
	return &ast.Tree{
		Val: &ast.Node{
			Typ: ast.ItemNum,
			Num: n,
		},
	}
}


