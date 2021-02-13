package runtime

import (
	"errors"
	"fmt"
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
		"Check":                   rt.bCheck,
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

type check struct {
	// name starlark.String
	hook          *starlark.Function
	tags          map[string]struct{}
	state, state0 starlark.Value
	//FIXME measure steps https://pkg.go.dev/go.starlark.net/starlark#Thread.ExecutionSteps
	//FIXME set/get ctx values: th.Local(k) / th.SetLocal(k,v)
}

func (chk *check) reset() (err error) {
	chk.state, err = slValueCopy(chk.state0)
	return
}

func (rt *Runtime) bCheck(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var hook *starlark.Function
	var name starlark.String
	var tags starlarkStringList
	var state0 starlark.Value //FIXME: only Value.s of ptypes
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"hook", &hook,
		"name", &name,
		"tags?", &tags,
		"state?", &state0,
	); err != nil {
		return nil, err
	}

	if name.Len() == 0 {
		return nil, errors.New("name for Check must be non-empty")
	}
	if hook.HasVarargs() || hook.HasKwargs() || hook.NumParams() != 1 {
		return nil, fmt.Errorf("hook for Check %s must have only one param: ctx", name.String())
	}
	if pname, _ := hook.Param(0); pname != "ctx" {
		return nil, fmt.Errorf("hook for Check %s must have only one param: ctx", name.String())
	}

	if state0 == nil {
		state0 = starlark.None
	}
	state0.Freeze()
	chk := &check{
		hook:   hook,
		tags:   tags.uniques,
		state0: state0,
	}

	if err := chk.reset(); err != nil {
		return nil, err
	}

	cname := name.GoString()
	if _, ok := rt.checks[cname]; ok {
		return nil, fmt.Errorf("a Check named %s already exists", name.String())
	}

	rt.checks[cname] = chk
	log.Printf("[NFO] registered %v: %+v", b.Name(), rt.checks[cname])
	return starlark.None, nil
}

var _ starlark.Unpacker = (*starlarkStringList)(nil)

type starlarkStringList struct {
	uniques map[string]struct{}
}

func (sl *starlarkStringList) Unpack(v starlark.Value) error {
	list, ok := v.(*starlark.List)
	if !ok {
		return fmt.Errorf("got %s, want list", v.Type())
	}

	sl.uniques = make(map[string]struct{}, list.Len())
	it := list.Iterate()
	defer it.Done()
	var x starlark.Value
	for it.Next(&x) {
		str, ok := starlark.AsString(x)
		if !ok {
			return fmt.Errorf("got %s, want string", x.Type())
		}
		if str == "" {
			return errors.New("empty strings are illegal")
		}
		if _, ok := sl.uniques[str]; ok {
			return fmt.Errorf("string %s appears at least twice in list", x.String())
		}
		sl.uniques[str] = struct{}{}
	}
	return nil
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
