package variable

import(
	"github.com/cptaffe/lang/ast"
)

type Var struct {
	Var  string // variable name
	Tree *ast.Tree  // every variable stored as a tree
}

type Variab struct {
	Lazy bool // lazy assignment or not so much
	Var   []*Var // array of Vars to create a list of variables
	Scope []*Var // scope has precedence if not null
}

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