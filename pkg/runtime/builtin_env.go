package runtime

import (
	"fmt"
	"log"
	"os"

	"go.starlark.net/starlark"
)

func (rt *Runtime) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	// def = starlark.None
	var env starlark.String
	var def starlark.Value
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}

	var defStr string
	if def != nil {
		if d, ok := def.(starlark.String); ok {
			defStr = d.GoString()
		} else {
			return nil, fmt.Errorf("expected string, got %s: %s", def.Type(), def.String())
		}
	}
	envStr := env.GoString()

	if cachedStr, ok := rt.envRead[envStr]; ok {
		log.Printf("[NFO] read (cached) env %q", envStr)
		return starlark.String(cachedStr), nil
	}

	if read, ok := os.LookupEnv(envStr); ok {
		rt.envRead[envStr] = read
		log.Printf("[NFO] read env %q: %q", envStr, read)
		return starlark.String(read), nil
	}

	if def == nil {
		return nil, fmt.Errorf("unset environment variable: %q", envStr)
	}

	rt.envRead[envStr] = defStr
	log.Printf("[NFO] read (unset) env %q: %q", envStr, defStr)
	return def, nil
}
