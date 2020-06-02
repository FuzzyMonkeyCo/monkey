package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// Lint TODO
func (rt *Runtime) Lint(ctx context.Context, showSpec bool) error {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.Lint(ctx, showSpec)
}
