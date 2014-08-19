// Originally made this as an llvm emitter,
// but I'm just going to emit C code.
// SUPER COMPATABILITY MODE!

package llvm

import(
	"fmt"
	"github.com/cptaffe/lang/token"
	"github.com/cptaffe/lang/ast"
	"github.com/cptaffe/lang/variable"
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

// emit
type emit func(t *ast.Tree) error

var lookup = map[token.ItemType]emit{
	token.ItemAdd: emitAdd,
	token.ItemSub: emitSub,
	token.ItemMul: emitMul,
	token.ItemDiv: emitDiv,
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	// print error message
	fmt.Printf("\033[1m%s: \033[31merror:\033[0m\033[1m %s\033[0m\n", "optim", msg)
}

func emitCode(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Evaluate
// evaluates the tree semi-recursively.
// Evaluate tries to quantize as much as it possibly can.
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
		// Here we are retrieving the value of a key,
		// in the interpreter, this ment evaluating a function,
		// here, it means creating a function call.
		tr := e.keys(t, v)
		return tr
	} else if t.Val.Typ == ast.ItemVar {
		// Here we are retrieving a variable definition.
		// These act as both constants and variables,
		// the emitter will optimize.
		tr := e.variables(t, v)
		if tr != nil {
			// If the variable is found, the variable's value
			// is evaluated.
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
// Emits the text equivalent to the key used.
// For example, (+ 1 1) will be "1 + 1" in c code,
// which can actually be optimized to "2".
func (e *evals) keys(t *ast.Tree, v *variable.Variab) *ast.Tree {
	// kill nils
	if t == nil {
		return t
	}
	// special tokens
	if t.Val.Typ == ast.ItemKey {
		switch {
			// For now I'm only focusing on implementing
			// the basic operators.
		/*case t.Val.Key == token.ItemAssign || t.Val.Key == token.ItemFunction:
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
			// emits proper C function call.
			return evalLambda(e.evaluateSubs(t, v), e, v)
		*/
		default:
			// Each language provided function or call has
			// a corresponding function that can emit appropriate
			// C code.
			trs := e.evaluateSubs(t, v)
			if trs != nil {
				val, ok := lookup[t.Val.Key]
				if ok && OnlyNums(t) {
					err := val(trs)
					if err == nil {
						// Totally rad shim...
						// TODO: Fix this shit.
						return &ast.Tree{
							Val: &ast.Node{
								Typ: ast.ItemNum,
								Num: 0,
							},
						}
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

// Does not actually add, instead emits C adding syntax.
// This operation is permitted within others, so no semicolons,
// newlines, or tabs were emitted.
// TODO: Add optimization.
func emitAdd(t *ast.Tree) error {
	for i := 0; i < len(t.Sub); i++ {
		// emit digit (only integers for now).
		emitCode("%d", t.Sub[i].Val.Num)
		if i != len(t.Sub) - 1 {
			// If between two digits, emit a '+' to add them
			emitCode(" + ")
		}
	}
	return nil
}

// Does not actually subtract, instead emits C adding syntax.
// This operation is permitted within others, so no semicolons,
// newlines, or tabs were emitted.
// TODO: Add optimization.
func emitSub(t *ast.Tree) error {
	for i := 0; i < len(t.Sub); i++ {
		// emit digit (only integers for now).
		emitCode("%d", t.Sub[i].Val.Num)
		if i != len(t.Sub) - 1 {
			// If between two digits, emit a '-' to subtract them
			emitCode(" - ")
		}
	}
	return nil
}

// Does not actually multiply, instead emits C adding syntax.
// This operation is permitted within others, so no semicolons,
// newlines, or tabs were emitted.
// TODO: Add optimization.
func emitMul(t *ast.Tree) error {
	for i := 0; i < len(t.Sub); i++ {
		// emit digit (only integers for now).
		emitCode("%d", t.Sub[i].Val.Num)
		if i != len(t.Sub) - 1 {
			// If between two digits, emit a '*' to multiply them
			emitCode(" * ")
		}
	}
	return nil
}

// Does not actually divide, instead emits C adding syntax.
// This operation is permitted within others, so no semicolons,
// newlines, or tabs were emitted.
// TODO: Add optimization.
func emitDiv(t *ast.Tree) error {
	for i := 0; i < len(t.Sub); i++ {
		// emit digit (only integers for now).
		emitCode("%d", t.Sub[i].Val.Num)
		if i != len(t.Sub) - 1 {
			// If between two digits, emit a '/' to divide them
			emitCode(" / ")
		}
	}
	return nil
}