package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// Lint goes through specs and unsures they're valid
func (rt *Runtime) Lint(ctx context.Context, showSpec bool) error {
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	return mdl.Lint(ctx, showSpec)
}
