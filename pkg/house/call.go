package runtime

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

func (rt *Runtime) recvFuzzProgress() error {
	log.Println("[DBG] receiving fm.Srv_FuzzProgress_...")
	srv, err := rt.client.Recv()
	if err != nil {
		log.Println("[ERR]", err)
		return err
	}

	switch srv.GetMsg().(type) {
	case *fm.Srv_FuzzProgress_:
		log.Println("[NFO] handling srvprogress")
		stts := srv.GetFuzzProgress()
		rt.progress.TotalTestsCount(stts.GetTotalTestsCount())
		rt.progress.TotalCallsCount(stts.GetTotalCallsCount())
		rt.progress.TotalChecksCount(stts.GetTotalChecksCount())
		rt.progress.TestCallsCount(stts.GetTestCallsCount())
		rt.progress.CallChecksCount(stts.GetCallChecksCount())
		if stts.GetSuccess() {
			rt.progress.CampaignSuccess(true)
		} else if stts.GetFailure() {
			rt.progress.CampaignSuccess(false)
		}
		log.Println("[NFO] done handling srvprogress")
		return nil
	default:
		err := fmt.Errorf("unexpected srv msg %T: %+v", srv.GetMsg(), srv)
		log.Println("[ERR]", err)
		return err
	}
}

func (rt *Runtime) call(ctx context.Context, msg *fm.Srv_Call) (err error) {
	showf := func(format string, s ...interface{}) {
		// TODO: prepend with 2-space indentation (somehow doesn't work)
		rt.progress.Showf(format, s)
	}

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}
	var cllr modeler.Caller
	if cllr, err = mdl.NewCaller(ctx, msg, showf); err != nil {
		return
	}
	log.Println("[NFO] ▼", msg.GetInput())

	var errCall error
	if errCall = cllr.Do(ctx); errCall != nil && errCall != modeler.ErrCallFailed {
		log.Println("[NFO] ▲", errCall)
		return errCall
	}

	output := cllr.ToProto()
	log.Println("[NFO] ▲", output)

	if err = rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_CallResponseRaw_{
			CallResponseRaw: output,
		}}); err != nil {
		log.Println("[ERR]", err)
		return
	}
	if err = rt.recvFuzzProgress(); err != nil {
		return
	}

	// FIXME? merge ErrCallFailed with output, as Do's return
	if errCall == modeler.ErrCallFailed {
		log.Println("[DBG] call failed, skipping checks")
		return modeler.ErrCallFailed
	}

	// Just the amount of checks needed to be able to call cllr.Response()
	if err = rt.firstChecks(cllr); err != nil {
		return
	}

	callResponse := cllr.Response()
	// Actionable response data parsed...
	if err = rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_CallVerifProgress_{
			CallVerifProgress: &fm.Clt_CallVerifProgress{
				Status:   fm.Clt_CallVerifProgress_data,
				Response: callResponse,
			}}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

	// TODO: user-defined eBPF triggers
	if err = rt.userChecks(callResponse); err != nil {
		return
	}

	// Through all checks: we're done
	if err = rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_CallVerifProgress_{
			CallVerifProgress: &fm.Clt_CallVerifProgress{},
		}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

	log.Println("[DBG] checks passed")
	rt.progress.ChecksPassed()
	return
}

// FIXME: turn this into a sync.errgroup with additional tasks being
// triggers with match-all predicates andalso pure actions
func (rt *Runtime) firstChecks(cllr modeler.Caller) (err error) {
	for {
		var lambda modeler.CheckerFunc
		v := &fm.Clt_CallVerifProgress{}
		v.Name, lambda = cllr.CheckFirst()
		if lambda == nil {
			return
		}
		log.Println("[NFO] checking", v.Name)

		v.Status = fm.Clt_CallVerifProgress_start
		if err = rt.client.Send(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}); err != nil {
			log.Println("[ERR]", err)
			return
		}

		success, failure := lambda()
		switch {
		case success != "":
			v.Status = fm.Clt_CallVerifProgress_success
			v.Reason = []string{success}
			rt.progress.CheckPassed(success)
		case len(failure) != 0:
			v.Status = fm.Clt_CallVerifProgress_failure
			v.Reason = failure
			log.Println(append([]string{"[NFO]"}, failure...))
			rt.progress.CheckFailed(failure)
		default:
			v.Status = fm.Clt_CallVerifProgress_skipped
			rt.progress.CheckSkipped(v.Name)
		}

		if err = rt.client.Send(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = rt.recvFuzzProgress(); err != nil {
			return
		}

		if v.Status == fm.Clt_CallVerifProgress_failure {
			err = modeler.ErrCheckFailed
			log.Println("[ERR]", err)
			return
		}
	}
}

func (rt *Runtime) userChecks(callResponse *types.Struct) (err error) {
	log.Printf("[NFO] checking %d user properties", len(rt.triggers))
	var response starlark.Value
	//FIXME: replace response copies by calls to this
	if response, err = slValueFromProto(&types.Value{
		Kind: &types.Value_StructValue{StructValue: callResponse}}); err != nil {
		log.Println("[ERR]", err)
		return
	}
	rt.thread.Print = func(_ *starlark.Thread, msg string) { rt.progress.Showf("%s", msg) }

	for i, trggr := range rt.triggers {
		v := &fm.Clt_CallVerifProgress{}
		v.Name = fmt.Sprintf("user property #%d: %q", i, trggr.name.GoString())
		log.Println("[NFO] checking", v.Name)

		v.Status = fm.Clt_CallVerifProgress_start
		if err = rt.client.Send(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}); err != nil {
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
							rt.progress.CheckPassed(v.Name)
						case *modelState:
							v.Status = fm.Clt_CallVerifProgress_success
							rt.modelState = newModelState
							rt.progress.CheckPassed(v.Name)
						default:
							v.Status = fm.Clt_CallVerifProgress_failure
							err = fmt.Errorf(
								"expected action %q (of %s) to return a ModelState, got: %T %v",
								trggr.act.Name(), v.Name, newModelState, newModelState,
							)
							v.Reason = strings.Split(err.Error(), "\n")
							log.Println("[NFO]", err)
							rt.progress.CheckFailed(v.Reason)
						}
					} else {
						v.Status = fm.Clt_CallVerifProgress_failure
						maybeEvalError(v, err)
						log.Println("[NFO]", err)
						rt.progress.CheckFailed(v.Reason)
					}
				} else {
					v.Status = fm.Clt_CallVerifProgress_skipped
					rt.progress.CheckSkipped(v.Name)
				}
			} else {
				v.Status = fm.Clt_CallVerifProgress_failure
				err = fmt.Errorf("expected predicate to return a Bool, got: %v", shouldBeBool)
				v.Reason = strings.Split(err.Error(), "\n")
				log.Println("[NFO]", err)
				rt.progress.CheckFailed(v.Reason)
			}
		} else {
			v.Status = fm.Clt_CallVerifProgress_failure
			maybeEvalError(v, err)
			log.Println("[NFO]", err)
			rt.progress.CheckFailed(v.Reason)
		}

		if err = rt.client.Send(&fm.Clt{
			Msg: &fm.Clt_CallVerifProgress_{
				CallVerifProgress: v,
			}}); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if err = rt.recvFuzzProgress(); err != nil {
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
	if e, ok := err.(*starlark.EvalError); ok {
		// TODO: think about a dedicated type
		v.Reason = strings.Split(e.Backtrace(), "\n")
		return
	}
	v.Reason = strings.Split(err.Error(), "\n")
}
