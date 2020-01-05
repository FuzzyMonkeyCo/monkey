package house

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// Lint TODO
func (rt *Runtime) Lint(showSpec bool) error {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.Lint(showSpec)
}
