package runtime

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

func (rt *runtime) builtins() map[string]builtin {
	return map[string]builtin{
		"Env":                     rt.bEnv,
		"TriggerActionAfterProbe": rt.bTriggerActionAfterProbe,
	}
}

func (rt *runtime) bEnv(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var env, def starlark.String
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &env, &def); err != nil {
		return nil, err
	}
	envStr := env.GoString()
	// FIXME: actually maybe read env from Exec shell? These shells should inherit user env anyway?
	read, ok := os.LookupEnv(envStr)
	if !ok {
		return def, nil
	}
	rt.envRead[envStr] = read
	return starlark.String(read), nil
}

type triggerActionAfterProbe struct {
	name      starlark.String
	probe     starlark.Tuple
	pred, act *starlark.Function
}

func (rt *runtime) bTriggerActionAfterProbe(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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
	log.Println("[NFO] registering", b.Name(), trigger)
	if name := trigger.name.GoString(); name == "" {
		trigger.name = starlark.String(trigger.act.Name())
		// FIXME: complain if trigger.Name == "lambda"
	}
	rt.triggers = append(rt.triggers, trigger)
	return starlark.None, nil
}
