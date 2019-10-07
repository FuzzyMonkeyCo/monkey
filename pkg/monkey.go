package pkg

import (
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"go.starlark.net/starlark"
)

var binTitle string

type monkey struct {
	usage    []string
	vald     *Validator
	eid      eid
	progress *progress

	thread     *starlark.Thread
	globals    starlark.StringDict
	modelState *modelState
	// EnvRead holds all the envs looked up on initial run
	envRead  map[string]string
	triggers []triggerActionAfterProbe

	modelers []modeler.Modeler
}
