package runtime

import (
	"fmt"
	"log"
	"strings"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkclone"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
)

type check struct {
	beforeRequest *starlark.Function
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
		return newUserError("state for check %q must be dict", chkname)
	}
	return err
}

func ensureStateDict(chkname string, v starlark.Value) error {
	if _, ok := v.(*starlark.Dict); !ok {
		return newUserError("state for check %q must be dict, got (%s) %s", chkname, v.Type(), v.String())
	}
	if err := starlarkvalue.ProtoCompatible(v); err != nil {
		return newUserError(err.Error())
	}
	return nil
}

func (rt *Runtime) bCheck(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var lot struct {
		beforeRequest, afterResponse *starlark.Function
		name                         starlark.String
		taglist                      tags.UniqueStrings
		state0                       *starlark.Dict
	}
	if err := starlark.UnpackArgs(b.Name(), args, kwargs,
		"name", &lot.name,
		// NOTE: all args following an optional? are implicitly optional.
		"before_request?", &lot.beforeRequest,
		"after_response?", &lot.afterResponse,
		"tags?", &lot.taglist,
		"state?", &lot.state0,
	); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	log.Printf("[DBG] unpacked %+v", lot)

	// verify each

	chkname := lot.name.GoString()
	if err := tags.LegalName(chkname); err != nil { //TODO: newUserError
		log.Println("[ERR]", err)
		return nil, err
	}

	if lot.beforeRequest != nil {
		if lot.beforeRequest.HasVarargs() || lot.beforeRequest.HasKwargs() || lot.beforeRequest.NumParams() != 1 {
			err := fmt.Errorf("before_request for check %s must have only one param: ctx", lot.name.String()) //TODO: newUserError
			log.Println("[ERR]", err)
			return nil, err
		}
		if pname, _ := lot.beforeRequest.Param(0); pname != "ctx" {
			err := fmt.Errorf("before_request for check %s must have only one param: ctx", lot.name.String()) //TODO: newUserError
			log.Println("[ERR]", err)
			return nil, err
		}
	}

	if lot.afterResponse != nil {
		if lot.afterResponse.HasVarargs() || lot.afterResponse.HasKwargs() || lot.afterResponse.NumParams() != 1 {
			err := fmt.Errorf("after_response for check %s must have only one param: ctx", lot.name.String()) //TODO: newUserError
			log.Println("[ERR]", err)
			return nil, err
		}
		if pname, _ := lot.afterResponse.Param(0); pname != "ctx" {
			err := fmt.Errorf("after_response for check %s must have only one param: ctx", lot.name.String()) //TODO: newUserError
			log.Println("[ERR]", err)
			return nil, err
		}
	}

	state0Unset := lot.state0 == nil
	if state0Unset {
		lot.state0 = &starlark.Dict{}
	}
	if err := ensureStateDict(chkname, lot.state0); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}
	lot.state0.Freeze()

	// verify all

	if lot.beforeRequest != nil && lot.afterResponse != nil {
		err := fmt.Errorf("check %s must have only one of before_request, after_response", lot.name.String()) //TODO: newUserError
		log.Println("[ERR]", err)
		return nil, err
	}
	if lot.beforeRequest == nil && lot.afterResponse == nil {
		err := fmt.Errorf("check %s must have one of before_request, after_response", lot.name.String()) //TODO: newUserError
		log.Println("[ERR]", err)
		return nil, err
	}

	if lot.beforeRequest != nil && !state0Unset {
		err := fmt.Errorf("unused state for check %s", lot.name.String()) //TODO: newUserError
		log.Println("[ERR]", err)
		return nil, err
	}

	// assemble

	chk := &check{
		beforeRequest: lot.beforeRequest,
		afterResponse: lot.afterResponse,
		tags:          lot.taglist.GoStringsMap(),
		state0:        lot.state0,
	}
	if err := chk.reset(chkname); err != nil {
		log.Println("[ERR]", err)
		return nil, err
	}

	if _, ok := rt.checks[chkname]; ok {
		err := fmt.Errorf("a check named %s already exists", lot.name.String()) //TODO: newUserError
		log.Println("[ERR]", err)
		return nil, err
	}
	rt.checks[chkname] = chk
	rt.checksNames = append(rt.checksNames, chkname)
	log.Printf("[NFO] registered %s: %q", b.Name(), chkname)

	return starlark.None, nil
}
