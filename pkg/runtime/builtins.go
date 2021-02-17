package runtime

import (
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
