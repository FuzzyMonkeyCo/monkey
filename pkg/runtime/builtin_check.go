package runtime

import (
	"fmt"
	"log"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkclone"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"go.starlark.net/starlark"
)

type check struct {
	afterResponse *starlark.Function
	tags          tags.Tags
	state, state0 *starlark.Dict
}

func (chk *check) reset(chkname string) (err error) {
	var state starlark.Value
	if state, err = starlarkclone.Clone(chk.state0); err != nil {
		return
	}
	if err = ensureStateDict(chkname, state); err != nil {
		return
	}
	chk.state = state.(*starlark.Dict)
	return
}

func errStateDict(chkname string, err error) error {
	if strings.Contains(err.Error(), `can't assign to .state field of `) {
		return fmt.Errorf("state for Check %q must be dict", chkname)
	}
	return err
}

func ensureStateDict(chkname string, v starlark.Value) (err error) {
	if _, ok := v.(*starlark.Dict); !ok {
		err = fmt.Errorf("state for Check %q must be dict, got (%s) %s", chkname, v.Type(), v.String())
	}
	return
}

func (rt *Runtime) bCheck(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var afterResponse *starlark.Function
	var name starlark.String
	var taglist tags.StarlarkStringList
	var state0 *starlark.Dict
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"after_response", &afterResponse,
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
	if afterResponse.HasVarargs() || afterResponse.HasKwargs() || afterResponse.NumParams() != 1 {
		return nil, fmt.Errorf("after_response for Check %s must have only one param: ctx", name.String())
	}
	if pname, _ := afterResponse.Param(0); pname != "ctx" {
		return nil, fmt.Errorf("after_response for Check %s must have only one param: ctx", name.String())
	}

	if state0 == nil {
		state0 = &starlark.Dict{}
	}
	if err := ensureStateDict(chkname, state0); err != nil {
		return nil, err
	}
	if err := starlarkvalue.ProtoCompatible(state0); err != nil { // TODO: ensure through preempting SetKey,...
		return nil, err
	}
	state0.Freeze()
	chk := &check{
		afterResponse: afterResponse,
		tags:          taglist.Uniques,
		state0:        state0,
	}
	if err := chk.reset(chkname); err != nil {
		return nil, err
	}

	if _, ok := rt.checks[chkname]; ok {
		return nil, fmt.Errorf("a Check named %s already exists", name.String())
	}
	rt.checks[chkname] = chk
	log.Printf("[NFO] registered %s: %+v", b.Name(), chk)

	return starlark.None, nil
}
