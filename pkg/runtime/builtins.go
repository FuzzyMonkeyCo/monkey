package runtime

import (
	"fmt"

	"go.starlark.net/starlark"
)

type builtin func(
	th *starlark.Thread,
	b *starlark.Builtin,
	args starlark.Tuple,
	kwargs []starlark.Tuple,
) (ret starlark.Value, err error)

func (rt *Runtime) builtins() map[string]builtin {
	return map[string]builtin{
		"Check": rt.bCheck,
		"Env":   rt.bEnv,
	}
}

type userError string

var _ error = userError("")

func newUserError(f string, a ...interface{}) userError { return userError(fmt.Sprintf(f, a...)) }
func (e userError) Error() string                       { return string(e) }
