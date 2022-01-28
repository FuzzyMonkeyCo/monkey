package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
)

// Lint goes through specs and unsures they are valid
func (rt *Runtime) Lint(ctx context.Context, showSpec bool) error {
	return rt.forEachModel(func(name string, mdl modeler.Interface) error {
		return mdl.Lint(ctx, showSpec)
	})
}
