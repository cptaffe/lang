package variable

import(
	"github.com/cptaffe/lang/ast"
)

type Var struct {
	Var  string // variable name
	Tree *ast.Tree  // every variable stored as a tree
}

type Scope struct {
	Parent *Scope // array of Vars to create a list of variables
	Scope []*Var // scope has precedence if not null
}

func (scope Scope) GetName(s string) *Var {
	// check current scope
	for i := 0; i < len(scope.Scope); i++ {
		if scope.Scope[i].Var == s {
			return scope.Scope[i]
		}
	}
	// check parent scope
	if scope.Parent != nil {
		return scope.Parent.GetName(s);
	}
	return nil
}

func (v *Var) String() string {
	return v.Tree.String()
}