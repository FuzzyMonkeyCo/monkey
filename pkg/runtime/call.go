package runtime

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
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
	callRequest := inputAsValue(input.GetInput())

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
		callResponse := outputAsValue(output.GetOutput())
		var passed2 bool
		if passed2, errT = rt.userChecks(ctx, callRequest, callResponse); errT != nil {
			return errT
		}
		if !passed2 {
			passed = false
			// Keep going
		}
	}

	// Through all checks
	if errT := rt.client.Send(ctx, cvp(&fm.Clt_CallVerifProgress{
		Status: fm.Clt_CallVerifProgress_done,
	})); errT != nil {
		log.Println("[ERR]", errT)
		return errT
	}

	if passed {
		log.Println("[DBG] checks passed")
		rt.progress.ChecksPassed()
	}
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
	start := time.Now()
	success, skipped, failure := lambda()
	v.ElapsedNs = time.Since(start).Nanoseconds()
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

func (rt *Runtime) userChecks(ctx context.Context, callRequest, callResponse starlark.Value) (bool, error) {
	log.Printf("[NFO] checking %d user properties", len(rt.checks))

	for name, chk := range rt.checks {
		name, chk := name, chk

		passed, errT := func() (bool, error) {
			v := &fm.Clt_CallVerifProgress{}
			v.Name = name
			v.UserProperty = true
			log.Println("[NFO] checking user property:", v.Name)

			start := time.Now()
			errL := rt.runUserCheck(v, chk, callRequest, callResponse)
			v.ElapsedNs = time.Since(start).Nanoseconds()
			switch {
			case errL == nil && v.Status == fm.Clt_CallVerifProgress_success:
				rt.progress.CheckPassed(v.Name, chk.hook.String())
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

			return v.Status == fm.Clt_CallVerifProgress_failure, nil
		}()
		if !passed || errT != nil {
			return passed, errT
		}
	}
	return true, nil
}

func (rt *Runtime) runUserCheck(
	v *fm.Clt_CallVerifProgress,
	chk *check,
	request, response starlark.Value,
) (err error) {
	// On success or skipping set status + return no error,
	// in all other cases just return error.

	th := &starlark.Thread{
		Name:  v.Name,
		Load:  loadDisabled,
		Print: func(_ *starlark.Thread, msg string) { rt.progress.Printf("%s", msg) },
	}

	args := starlark.Tuple{newCtx(chk.state, request, response)}
	// args.Freeze()
	log.Println("[NFO] >>>", v.Name, args.String())

	var hookRet starlark.Value
	if hookRet, err = starlark.Call(th, chk.hook, args, nil); err != nil {
		// Check failed or an error happened
		log.Println("[NFO]", err)
		return
	}
	if hookRet != starlark.None {
		err = fmt.Errorf("hooks should return None, got: %s", hookRet.String())
		log.Println("[ERR]", err)
		return
	}
	log.Printf("[DBG] closeness >>> %+v", th.Local("closeness"))

	// FIXME: didTrigger = chk.state.mutated() || assert.that called
	// if !didTrigger {
	// 	// Predicate did not trigger
	// 	v.Status = fm.Clt_CallVerifProgress_skipped
	// 	return
	// }

	// Check passed
	v.Status = fm.Clt_CallVerifProgress_success
	return
}
