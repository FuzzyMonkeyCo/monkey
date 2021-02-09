package runtime

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

func (rt *Runtime) call(ctx context.Context, msg *fm.Srv_Call) error {
	showf := func(format string, s ...interface{}) {
		// TODO: prepend with 2-space indentation (somehow doesn't work)
		rt.progress.Printf(format, s...)
	}

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	log.Printf("[NFO] raw input: %.999v", msg.GetInput())
	cllr := mdl.NewCaller(ctx, msg, showf)

	input := cllr.RequestProto()
	log.Printf("[NFO] call input: %.999v", input)
	if errT := rt.client.Send(ctx, &fm.Clt{Msg: &fm.Clt_CallRequestRaw_{
		CallRequestRaw: input,
	}}); errT != nil {
		log.Println("[ERR]", errT)
		return errT
	}
	if len(input.GetReason()) != 0 {
		// An error happened building the request: cannot continue.
		return nil
	}
	callRequest := input.GetInput().Expose()

	cllr.Do(ctx)

	output := cllr.ResponseProto()
	log.Printf("[NFO] call output: %.999v", output)
	if errT := rt.client.Send(ctx, &fm.Clt{Msg: &fm.Clt_CallResponseRaw_{
		CallResponseRaw: output,
	}}); errT != nil {
		log.Println("[ERR]", errT)
		return errT
	}
	if errT := rt.recvFuzzingProgress(ctx); errT != nil {
		return errT
	}

	// Just the amount of checks needed to be able to call cllr.Response()
	passed, errT := rt.callerChecks(ctx, cllr)
	if errT != nil {
		return errT
	}
	if !passed {
		// Return as early as the first check fails
		return nil
	}

	{
		callResponse := output.GetOutput().Expose(callRequest)
		printfunc := func(_ *starlark.Thread, msg string) {
			rt.progress.Printf("%s", msg)
		}
		printfunc, rt.thread.Print = rt.thread.Print, printfunc
		if passed, errT = rt.userChecks(ctx, callResponse); errT != nil {
			return errT
		}
		rt.thread.Print = printfunc
		log.Printf("[DBG] closeness >>> %+v", rt.thread.Local("closeness"))
		if !passed {
			// Return as early as the first check fails
			return nil
		}
	}

	// Through all checks
	if errT := rt.client.Send(ctx, cvp(&fm.Clt_CallVerifProgress{
		Status: fm.Clt_CallVerifProgress_done,
	})); errT != nil {
		log.Println("[ERR]", errT)
		return errT
	}

	log.Println("[DBG] checks passed")
	rt.progress.ChecksPassed()
	return nil
}

func cvp(msg *fm.Clt_CallVerifProgress) *fm.Clt {
	return &fm.Clt{Msg: &fm.Clt_CallVerifProgress_{CallVerifProgress: msg}}
}

// NOTE: callerChecks are applied sequentially in order of definition.
// Model state can be mutated by each check.
func (rt *Runtime) callerChecks(ctx context.Context, cllr modeler.Caller) (bool, error) {
	for {
		var lambda modeler.CheckerFunc
		v := &fm.Clt_CallVerifProgress{}
		if v.Name, lambda = cllr.NextCallerCheck(); lambda == nil {
			// No more caller checks to run
			return true, nil
		}
		log.Println("[NFO] checking", v.Name)

		v.Status = fm.Clt_CallVerifProgress_start
		if errT := rt.client.Send(ctx, cvp(v)); errT != nil {
			log.Println("[ERR]", errT)
			return false, errT
		}

		rt.runCallerCheck(v, lambda)

		if errT := rt.client.Send(ctx, cvp(v)); errT != nil {
			log.Println("[ERR]", errT)
			return false, errT
		}
		if errT := rt.recvFuzzingProgress(ctx); errT != nil {
			return false, errT
		}

		if v.Status == fm.Clt_CallVerifProgress_failure {
			return false, nil
		}
	}
}

func (rt *Runtime) runCallerCheck(
	v *fm.Clt_CallVerifProgress,
	lambda modeler.CheckerFunc,
) {
	success, skipped, failure := lambda()
	hasSuccess := success != ""
	hasSkipped := skipped != ""
	hasFailure := len(failure) != 0
	trues := 0
	if hasSuccess {
		trues++
	}
	if hasSkipped {
		trues++
	}
	if hasFailure {
		trues++
	}

	switch {
	case trues == 1 && hasSuccess:
		v.Status = fm.Clt_CallVerifProgress_success
		v.Reason = []string{success}
		rt.progress.CheckPassed(v.Name, v.Reason[0])
		return
	case trues == 1 && hasSkipped:
		v.Status = fm.Clt_CallVerifProgress_skipped
		v.Reason = []string{skipped}
		rt.progress.CheckSkipped(v.Name, v.Reason[0])
		return
	case trues == 1 && hasFailure:
		v.Status = fm.Clt_CallVerifProgress_failure
		v.Reason = failure
		log.Printf("[NFO] check failed: %v", failure)
		rt.progress.CheckFailed(v.Name, v.Reason)
		return
	default: // unreachable
		v.Status = fm.Clt_CallVerifProgress_failure
		v.Reason = []string{"unexpected check result"}
		log.Printf("[ERR] %s: success:%q skipped:%q failure:%+v",
			v.Reason[0], success, skipped, failure)
		rt.progress.CheckFailed(v.Name, v.Reason)
		return
	}
}

func (rt *Runtime) userChecks(ctx context.Context, callResponse *types.Struct) (bool, error) {
	log.Printf("[NFO] checking %d user properties", len(rt.triggers))

	cloneResponse := func() starlark.Value {
		return starlarkvalue.FromProtoValue(&types.Value{
			Kind: &types.Value_StructValue{StructValue: callResponse}})
	}

	for _, trggr := range rt.triggers {
		v := &fm.Clt_CallVerifProgress{}
		v.Name = trggr.name.GoString()
		v.UserProperty = true
		log.Println("[NFO] checking user property:", v.Name)

		v.Status = fm.Clt_CallVerifProgress_start
		if errT := rt.client.Send(ctx, cvp(v)); errT != nil {
			log.Println("[ERR]", errT)
			return false, errT
		}

		errL := rt.runUserCheck(v, trggr, cloneResponse)
		switch {
		case errL == nil && v.Status == fm.Clt_CallVerifProgress_success:
			rt.progress.CheckPassed(v.Name, trggr.act.String())
		case errL == nil && v.Status == fm.Clt_CallVerifProgress_skipped:
			rt.progress.CheckSkipped(v.Name, "")
		case errL == nil: // unreachable
			errL = fmt.Errorf("unexpected v.Status %+v %q", v.Status, v.Status)
			log.Println("[ERR]", errL)
		}
		if errL != nil {
			v.Status = fm.Clt_CallVerifProgress_failure
			var reason string
			if e, ok := errL.(*starlark.EvalError); ok {
				reason = e.Backtrace()
			} else {
				reason = errL.Error()
			}
			v.Reason = strings.Split(reason, "\n")
			rt.progress.CheckFailed(v.Name, v.Reason)
		}

		if errT := rt.client.Send(ctx, cvp(v)); errT != nil {
			log.Println("[ERR]", errT)
			return false, errT
		}
		if errT := rt.recvFuzzingProgress(ctx); errT != nil {
			return false, errT
		}

		if v.Status == fm.Clt_CallVerifProgress_failure {
			return false, nil
		}
	}
	return true, nil
}

func (rt *Runtime) runUserCheck(
	v *fm.Clt_CallVerifProgress,
	trggr triggerActionAfterProbe,
	cloneResponse func() starlark.Value,
) (err error) {
	// On success or skipping set status + return no error,
	// in all other cases just return error.

	var modelState1 starlark.Value
	if modelState1, err = slValueCopy(rt.modelState); err != nil {
		log.Println("[ERR]", err)
		return
	}
	args1 := starlark.Tuple{modelState1, cloneResponse()}

	var shouldBeBool starlark.Value
	if shouldBeBool, err = starlark.Call(rt.thread, trggr.pred, args1, nil); err != nil {
		log.Println("[NFO]", err)
		return
	}

	didTrigger, ok := shouldBeBool.(starlark.Bool)
	if !ok {
		err = fmt.Errorf("expected predicate to return a Bool, got: %s", shouldBeBool.Type())
		log.Println("[NFO]", err)
		return
	}
	if !didTrigger {
		// Predicate did not trigger
		v.Status = fm.Clt_CallVerifProgress_skipped
		return
	}

	var modelState2 starlark.Value
	if modelState2, err = slValueCopy(rt.modelState); err != nil {
		log.Println("[ERR]", err)
		return
	}
	args2 := starlark.Tuple{modelState2, cloneResponse()}

	var newModelState starlark.Value
	if newModelState, err = starlark.Call(rt.thread, trggr.act, args2, nil); err != nil {
		// Check failed or an error happened
		log.Println("[NFO]", err)
		return
	}
	switch newModelState.(type) {
	case starlark.NoneType:
		// Check passed without returning new State
		v.Status = fm.Clt_CallVerifProgress_success
		return
	case *modelState:
		// Check passed and returned new State
		v.Status = fm.Clt_CallVerifProgress_success
		var newModelStateCopy starlark.Value
		if newModelStateCopy, err = slValueCopy(newModelState); err != nil {
			log.Println("[ERR]", err)
			return
		}
		rt.modelState = newModelStateCopy.(*modelState)
		return
	default:
		err = fmt.Errorf(
			"expected action %q (of %s) to return a ModelState, got: %s",
			trggr.act.Name(), v.Name, newModelState.Type(),
		)
		log.Println("[NFO]", err)
		return
	}
}
