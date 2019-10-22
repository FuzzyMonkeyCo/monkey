package runtime

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// Lint TODO
func (rt *runtime) Lint(showSpec bool) error {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.Lint(showSpec)
}
