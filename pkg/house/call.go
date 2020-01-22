package house

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

func (rt *Runtime) call(ctx context.Context, msg *fm.Srv_Call) (err error) {
	showf := func(format string, s ...interface{}) {
		// TODO: prepend with 2-space indentation (somehow doesn't work)
		rt.progress.Printf(format, s)
	}

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}
	var cllr modeler.Caller
	if cllr, err = mdl.NewCaller(ctx, msg, showf); err != nil {
		return
	}

	log.Println("[NFO] call input:", msg.GetInput())
	cllr.Do(ctx)
	output := cllr.ToProto()
	log.Println("[NFO] call output:", output)

	select {
	case <-time.After(tx30sTimeout):
		err = err30sTimeout
	case err = <-rt.client.Snd(&fm.Clt{
		Msg: &fm.Clt_CallResponseRaw_{
			CallResponseRaw: output,
		}}):
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	if err = rt.recvFuzzProgress(ctx); err != nil {
		return
	}

	// Just the amount of checks needed to be able to call cllr.Response()
	if err = rt.callerChecks(ctx, cllr); err != nil {
		return
	}

	callResponse := cllr.Response()
	// Actionable response data parsed...
	select {
	case <-time.After(tx30sTimeout):
		err = err30sTimeout
	case err = <-rt.client.Snd(&fm.Clt{
		Msg: &fm.Clt_CallVerifProgress_{
			CallVerifProgress: &fm.Clt_CallVerifProgress{
				Status:   fm.Clt_CallVerifProgress_data,
				Response: callResponse,
			}}}):
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	// TODO: user-defined eBPF triggers
	if err = rt.userChecks(ctx, callResponse); err != nil {
		return
	}

	// Through all checks: we're done
	select {
	case <-time.After(tx30sTimeout):
		err = err30sTimeout
	case err = <-rt.client.Snd(&fm.Clt{
		Msg: &fm.Clt_CallVerifProgress_{
			CallVerifProgress: &fm.Clt_CallVerifProgress{},
		}}):
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[DBG] checks passed")
	rt.progress.ChecksPassed()
	return
}

// FIXME: turn this into a sync.errgroup with additional tasks being
// triggers with match-all predicates andalso pure actions
func (rt *Runtime) callerChecks(ctx context.Context, cllr modeler.Caller) (err error) {
	for {
		var lambda modeler.CheckerFunc
		v := &fm.Clt_CallVerifProgress{}
		v.Name, lambda = cllr.NextCallerCheck()
		if lambda == nil {
			// No more caller checks to run
			return
		}
		log.Println("[NFO] checking", v.Name)

		v.Status = fm.Clt_CallVerifProgress_start
		select {
		case <-time.After(tx30sTimeout):
			err = err30sTimeout
		case err = <-rt.client.Snd(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}):
		}
		if err != nil {
			log.Println("[ERR]", err)
			return
		}

		success, skipped, failure := lambda()
		switch {
		case (success != "" && skipped != "") || (success != "" && len(failure) != 0) || (skipped != "" && len(failure) != 0) || (success == "" && skipped == "" && len(failure) == 0):
			v.Status = fm.Clt_CallVerifProgress_failure
			v.Reason = []string{"check result unclear"}
			log.Println("[ERR]", v.Reason[0])
			log.Printf("[ERR] success: %q", success)
			log.Printf("[ERR] skipped: %q", skipped)
			log.Printf("[ERR] failure: %v", failure)
			rt.progress.CheckFailed(v.Name, v.Reason)
		case success != "":
			v.Status = fm.Clt_CallVerifProgress_success
			v.Reason = []string{success}
			rt.progress.CheckPassed(v.Name, v.Reason[0])
		case len(failure) != 0:
			v.Status = fm.Clt_CallVerifProgress_failure
			v.Reason = failure
			log.Printf("[NFO] check failed: %v", failure)
			rt.progress.CheckFailed(v.Name, v.Reason)
		default:
			v.Status = fm.Clt_CallVerifProgress_skipped
			v.Reason = []string{skipped}
			rt.progress.CheckSkipped(v.Name, v.Reason[0])
		}

		select {
		case <-time.After(tx30sTimeout):
			err = err30sTimeout
		case err = <-rt.client.Snd(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}):
		}
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = rt.recvFuzzProgress(ctx); err != nil {
			return
		}

		if v.Status == fm.Clt_CallVerifProgress_failure {
			err = modeler.ErrCheckFailed
			log.Println("[ERR]", err)
			return
		}
	}
}

func (rt *Runtime) userChecks(ctx context.Context, callResponse *types.Struct) (err error) {
	log.Printf("[NFO] checking %d user properties", len(rt.triggers))
	var response starlark.Value
	//FIXME: replace response copies by calls to this
	if response, err = slValueFromProto(&types.Value{
		Kind: &types.Value_StructValue{StructValue: callResponse}}); err != nil {
		log.Println("[ERR]", err)
		return
	}
	rt.thread.Print = func(_ *starlark.Thread, msg string) {
		rt.progress.Printf("%s", msg)
	}

	for _, trggr := range rt.triggers {
		v := &fm.Clt_CallVerifProgress{}
		v.Name = trggr.name.GoString()
		v.UserProperty = true
		log.Println("[NFO] checking user property:", v.Name)

		v.Status = fm.Clt_CallVerifProgress_start
		select {
		case <-time.After(tx30sTimeout):
			err = err30sTimeout
		case err = <-rt.client.Snd(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}):
		}
		if err != nil {
			log.Println("[ERR]", err)
			return
		}

		var modelState1, response1 starlark.Value
		if modelState1, err = slValueCopy(rt.modelState); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if response1, err = slValueCopy(response); err != nil {
			log.Println("[ERR]", err)
			return
		}
		args1 := starlark.Tuple{modelState1, response1}

		var shouldBeBool starlark.Value
		//FIXME: forbid modelState mutation from pred
		if shouldBeBool, err = starlark.Call(rt.thread, trggr.pred, args1, nil); err == nil {
			if triggered, ok := shouldBeBool.(starlark.Bool); ok {
				if triggered {
					var modelState2, response2 starlark.Value
					if modelState2, err = slValueCopy(rt.modelState); err != nil {
						log.Println("[ERR]", err)
						return
					}
					if response2, err = slValueCopy(response); err != nil {
						log.Println("[ERR]", err)
						return
					}
					args2 := starlark.Tuple{modelState2, response2}

					var newModelState starlark.Value
					if newModelState, err = starlark.Call(rt.thread, trggr.act, args2, nil); err == nil {
						switch newModelState := newModelState.(type) {
						case starlark.NoneType:
							v.Status = fm.Clt_CallVerifProgress_success
							rt.progress.CheckPassed(v.Name, "")
						case *modelState:
							v.Status = fm.Clt_CallVerifProgress_success
							rt.modelState = newModelState
							rt.progress.CheckPassed(v.Name, "")
						default:
							v.Status = fm.Clt_CallVerifProgress_failure
							err = fmt.Errorf(
								"expected action %q (of %s) to return a ModelState, got: %T %v",
								trggr.act.Name(), v.Name, newModelState, newModelState,
							)
							v.Reason = strings.Split(err.Error(), "\n")
							log.Println("[NFO]", err)
							rt.progress.CheckFailed(v.Name, v.Reason)
						}
					} else {
						v.Status = fm.Clt_CallVerifProgress_failure
						maybeEvalError(v, err)
						log.Println("[NFO]", err)
						rt.progress.CheckFailed(v.Name, v.Reason)
					}
				} else {
					v.Status = fm.Clt_CallVerifProgress_skipped
					rt.progress.CheckSkipped(v.Name, "predicate doesn't hold")
				}
			} else {
				v.Status = fm.Clt_CallVerifProgress_failure
				err = fmt.Errorf("expected predicate to return a Bool, got: %v", shouldBeBool)
				v.Reason = strings.Split(err.Error(), "\n")
				log.Println("[NFO]", err)
				rt.progress.CheckFailed(v.Name, v.Reason)
			}
		} else {
			v.Status = fm.Clt_CallVerifProgress_failure
			maybeEvalError(v, err)
			log.Println("[NFO]", err)
			rt.progress.CheckFailed(v.Name, v.Reason)
		}

		select {
		case <-time.After(tx30sTimeout):
			err = err30sTimeout
		case err = <-rt.client.Snd(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}):
		}
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = rt.recvFuzzProgress(ctx); err != nil {
			return
		}

		if v.Status == fm.Clt_CallVerifProgress_failure {
			err = modeler.ErrCheckFailed
			log.Println("[ERR]", err)
			return
		}
	}
	return
}

func maybeEvalError(v *fm.Clt_CallVerifProgress, err error) {
	var reason string
	if e, ok := err.(*starlark.EvalError); ok {
		reason = e.Backtrace()
	} else {
		reason = err.Error()
	}
	v.Reason = strings.Split(reason, "\n")
}
