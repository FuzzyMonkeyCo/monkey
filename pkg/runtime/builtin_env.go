package runtime

import (
	"log"
	"os"

	"go.starlark.net/starlark"
)

func (rt *Runtime) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()

	if cachedStr, ok := rt.envRead[envStr]; ok {
		log.Printf("[NFO] read (cached) env %q: %q", envStr, cachedStr)
		return starlark.String(cachedStr), nil
	}

	if read, ok := os.LookupEnv(envStr); ok {
		rt.envRead[envStr] = read
		log.Printf("[NFO] read env %q: %q", envStr, read)
		return starlark.String(read), nil
	}

	defStr := def.GoString()
	rt.envRead[envStr] = defStr
	log.Printf("[NFO] read (unset) env %q: %q", envStr, defStr)
	return def, nil
}
