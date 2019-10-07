package fm

import (
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

	modelers []Modeler
}

type cltDoer interface {
	isClt_Msg_Msg()
	do(mnk *monkey) (err error)
}
