package runtime

import (
	"errors"
	"fmt"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkclone"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"go.starlark.net/starlark"
)

type check struct {
	hook          *starlark.Function
	tags          map[string]struct{}
	state, state0 starlark.Value
	//FIXME set/get ctx values: th.Local(k) / th.SetLocal(k,v)
}

func (chk *check) reset() (err error) {
	chk.state, err = starlarkclone.Clone(chk.state0)
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
	if err := starlarkvalue.ProtoCompatible(state0); err != nil {
		return nil, err
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

	chkname := name.GoString()
	if _, ok := rt.checks[chkname]; ok {
		return nil, fmt.Errorf("a Check named %s already exists", name.String())
	}
	rt.checks[chkname] = chk
	log.Printf("[NFO] registered %v: %+v", b.Name(), chk)

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
