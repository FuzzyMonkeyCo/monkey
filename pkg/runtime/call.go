package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/caller"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// CheckerFunc TODO
type CheckerFunc func(*runtime.runtime) (string, []string)

func (rt *runtime) call(ctx context.Context, cllr caller.Caller) (err error) {
	// mnk.eid = act.EID

	tcap = newHTTPTCap(func(format string, s ...interface{}) {
		// TODO: prepend with 2-space indentation (somehow doesn't work)
		rt.progress.Showf(format, s)
	})
	var nxt *RepCallDone
	if nxt, err = tcap.makeRequest(act.GetRequest()); err != nil {
		return
	}

	if err = rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Msg{
			Msg: &fm.Clt_Msg_CallResponseRaw_{
				CallResponseRaw: cllr.ToProto(),
			}}}); err != nil {
		log.Println("[ERR]", err)
		return err
	}

	err = mnk.castPostConditions(nxt)
	// mnk.eid = 0
	return
}

func (mnk *pkg.monkey) castPostConditions(act *msgCallResponseRaw) (err error) {
	if act.Failure {
		log.Println("[DBG] call failed, skipping checks")
		return
	}

	var reqrep CallCapturer = tcap
	// FIXME: turn this into a sync.errgroup with additional tasks being
	// triggers with match-all predicates andalso pure actions
	for {
		name, lambda := reqrep.CheckFirst()
		if name == "" {
			break
		}

		check := &RepValidateProgress{Details: []string{name}}
		log.Println("[NFO] checking", check.Details[0])
		success, failure := lambda(mnk)
		switch {
		case success != "":
			check.Success = true
			mnk.progress.checkPassed(success)
		case len(failure) != 0:
			check.Details = append(check.Details, failure...)
			log.Println(append([]string{"[NFO]"}, failure...))
			mnk.progress.checkFailed(failure)
		default:
			mnk.progress.checkSkipped(check.Details[0])
		}

		if err = mnk.ws.cast(check); err != nil {
			log.Println("[ERR]", err)
			return
		}
		if check.Failure {
			return
		}
	}

	// Check #N: user-provided postconditions
	{
		log.Printf("[NFO] checking %d user properties", len(mnk.triggers))
		var response starlark.Value
		if response, err = slValueFromInterface(reqrep.Response()); err != nil {
			log.Println("[ERR]", err)
			return
		}
		mnk.thread.Print = func(_ *starlark.Thread, msg string) { mnk.progress.wrn(msg) }
		for i, trigger := range mnk.triggers {
			checkN := &RepValidateProgress{Details: []string{fmt.Sprintf("user property #%d: %q", i, trigger.Name.GoString())}}
			log.Println("[NFO] checking", checkN.Details[0])

			var modelState1, response1 starlark.Value
			if modelState1, err = slValueCopy(mnk.modelState); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if response1, err = slValueCopy(response); err != nil {
				log.Println("[ERR]", err)
				return
			}
			args1 := starlark.Tuple{modelState1, response1}

			var shouldBeBool starlark.Value
			if shouldBeBool, err = starlark.Call(mnk.thread, trigger.Predicate, args1, nil); err == nil {
				if triggered, ok := shouldBeBool.(starlark.Bool); ok {
					if triggered {

						var modelState2, response2 starlark.Value
						if modelState2, err = slValueCopy(mnk.modelState); err != nil {
							log.Println("[ERR]", err)
							return
						}
						if response2, err = slValueCopy(response); err != nil {
							log.Println("[ERR]", err)
							return
						}
						args2 := starlark.Tuple{modelState2, response2}

						var newModelState starlark.Value
						if newModelState, err = starlark.Call(mnk.thread, trigger.Action, args2, nil); err == nil {
							switch newModelState := newModelState.(type) {
							case starlark.NoneType:
								checkN.Success = true
								mnk.progress.checkPassed(checkN.Details[0])
							case *modelState:
								mnk.modelState = newModelState
								checkN.Success = true
								mnk.progress.checkPassed(checkN.Details[0])
							default:
								checkN.Failure = true
								err = fmt.Errorf("expected action %q (of %s) to return a ModelState, got: %T %v",
									trigger.Action.Name(), checkN.Details[0], newModelState, newModelState)
								e := err.Error()
								checkN.Details = append(checkN.Details, e)
								log.Println("[NFO]", err)
								mnk.progress.checkFailed([]string{e})
							}
						} else {
							checkN.Failure = true
							//TODO: split on \n.s or you know create a type better than []string
							if evalErr, ok := err.(*starlark.EvalError); ok {
								checkN.Details = append(checkN.Details, evalErr.Backtrace())
							} else {
								checkN.Details = append(checkN.Details, err.Error())
							}
							log.Println("[NFO]", err)
							mnk.progress.checkFailed(checkN.Details)
						}
					} else {
						mnk.progress.checkSkipped(checkN.Details[0])
					}
				} else {
					checkN.Failure = true
					err = fmt.Errorf("expected predicate to return a Bool, got: %v", shouldBeBool)
					e := err.Error()
					checkN.Details = append(checkN.Details, e)
					log.Println("[NFO]", err)
					mnk.progress.checkFailed([]string{e})
				}
			} else {
				checkN.Failure = true
				//TODO: split on \n.s or you know create a type better than []string
				if evalErr, ok := err.(*starlark.EvalError); ok {
					checkN.Details = append(checkN.Details, evalErr.Backtrace())
				} else {
					checkN.Details = append(checkN.Details, err.Error())
				}
				log.Println("[NFO]", err)
				mnk.progress.checkFailed(checkN.Details[1:])
			}
			if err = mnk.ws.cast(checkN); err != nil {
				log.Println("[ERR]", err)
				return
			}
			if checkN.Failure {
				return
			}
		}
	}

	// Check #Z: all checks passed
	checkZ := &RepCallResult{} //FIXME:Response: enumFromGo(jsonData)}
	if err = mnk.ws.cast(checkZ); err != nil {
		log.Println("[ERR]", err)
		return
	}
	log.Println("[DBG] checks passed")
	mnk.progress.checksPassed()
	return
}
