package runtime

import (
	"fmt"
	"log"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkclone"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"go.starlark.net/starlark"
)

type check struct {
	hook          *starlark.Function
	tags          tags.Tags
	state, state0 starlark.Value
}

func (chk *check) reset() (err error) {
	chk.state, err = starlarkclone.Clone(chk.state0)
	return
}

func (rt *Runtime) bCheck(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var hook *starlark.Function
	var name starlark.String
	var taglist tags.StarlarkStringList
	var state0 starlark.Value
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"hook", &hook,
		"name", &name,
		"tags?", &taglist,
		"state?", &state0,
	); err != nil {
		return nil, err
	}

	chkname := name.GoString()
	if err := tags.LegalName(chkname); err != nil {
		return nil, fmt.Errorf("bad name for Check: %v", err)
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
	if err := starlarkvalue.ProtoCompatible(state0); err != nil { // TODO: ensure through preempting SetKey,...
		return nil, err
	}
	state0.Freeze()
	chk := &check{
		hook:   hook,
		tags:   taglist.Uniques,
		state0: state0,
	}
	if err := chk.reset(); err != nil {
		return nil, err
	}

	if _, ok := rt.checks[chkname]; ok {
		return nil, fmt.Errorf("a Check named %s already exists", name.String())
	}
	rt.checks[chkname] = chk
	log.Printf("[NFO] registered %s: %+v", b.Name(), chk)

	return starlark.None, nil
}
