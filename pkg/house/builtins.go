package house

import (
	"log"
	"os"

	"go.starlark.net/starlark"
)

type builtin func(
	*starlark.Thread,
	*starlark.Builtin,
	starlark.Tuple,
	[]starlark.Tuple,
) (starlark.Value, error)

func (rt *Runtime) builtins() map[string]builtin {
	return map[string]builtin{
		"Env":                     rt.bEnv,
		"TriggerActionAfterProbe": rt.bTriggerActionAfterProbe,
	}
}

func (rt *Runtime) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()
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

type triggerActionAfterProbe struct {
	name      starlark.String
	probe     starlark.Tuple
	pred, act *starlark.Function
}

func (rt *Runtime) bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var trigger triggerActionAfterProbe
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"name?", &trigger.name,
		"probe", &trigger.probe,
		"predicate", &trigger.pred,
		"action", &trigger.act,
	); err != nil {
		return nil, err
	}
	// FIXME: enforce arities
	if name := trigger.name.GoString(); name == "" {
		trigger.name = starlark.String(trigger.act.Name())
		// FIXME: complain if trigger.Name == "lambda"
	}
	log.Printf("[NFO] registering %v %v: %s", b.Name(), trigger.probe, trigger.name)
	rt.triggers = append(rt.triggers, trigger)
	return starlark.None, nil
}
