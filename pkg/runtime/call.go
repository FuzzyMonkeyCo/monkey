package runtime

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"go.starlark.net/starlark"
	"golang.org/x/sync/errgroup"
)

func (rt *Runtime) call(ctx context.Context, msg *fm.Srv_Call, tagsFilter *tags.Filter, maxSteps uint64) error {
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
	ctxer2 := ctxCurry(input.GetInput())

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
		ctxer1 := ctxer2(output.GetOutput())
		var passed2 bool
		if passed2, errT = rt.userChecks(ctx, tagsFilter, ctxer1, maxSteps); errT != nil {
			return errT
		}
		if !passed2 {
			passed = false
			// Keep going
		}
	}

	// Through all checks
	if errT := rt.client.Send(ctx, cvp(&fm.Clt_CallVerifProgress{
		Origin: fm.Clt_CallVerifProgress_built_in,
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
		v := &fm.Clt_CallVerifProgress{Origin: fm.Clt_CallVerifProgress_built_in}
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

func (rt *Runtime) userChecks(ctx context.Context, tagsFilter *tags.Filter, ctxer1 ctxctor1, maxSteps uint64) (bool, error) {
	log.Printf("[NFO] checking %d user properties", len(rt.checks))

	g, _ := errgroup.WithContext(ctx)
	vs := make(chan *fm.Clt_CallVerifProgress, len(rt.checks))
	// Run all checks concurrently, send their results to vs.
	// Concurrently, consume and send these one-by-one.
	// Return early if ctx is canceled.

	passed := true
	g.Go(func() (errT error) {
		for i := 0; i < len(rt.checks); i++ {
			select {
			case <-ctx.Done():
				errT, passed = ctx.Err(), false
				// TODO: (*starlark.Thread).Cancel(ctx.Err().Error()) for each userCheck
				return
			case v := <-vs:
				if errT = rt.client.Send(ctx, cvp(v)); errT != nil {
					log.Println("[ERR]", errT)
					passed = false
					return
				}
				if errT = rt.recvFuzzingProgress(ctx); errT != nil {
					passed = false
					return
				}

				if v.Status == fm.Clt_CallVerifProgress_failure {
					passed = false
				}
			}
		}
		close(vs)
		return
	})

	for _, name := range rt.checksNames {
		name, chk := name, rt.checks[name]

		g.Go(func() error {
			v := rt.runUserCheckWrapper(name, chk, tagsFilter, ctxer1, maxSteps)
			switch v.Status {
			case fm.Clt_CallVerifProgress_success:
				rt.progress.CheckPassed(v.Name, chk.afterResponse.String())
			case fm.Clt_CallVerifProgress_skipped:
				rt.progress.CheckSkipped(v.Name, "")
			case fm.Clt_CallVerifProgress_failure:
				rt.progress.CheckFailed(v.Name, v.Reason)
			}

			vs <- v
			return nil
		})
	}

	if errT := g.Wait(); errT != nil {
		return false, errT
	}
	return passed, nil
}

func (rt *Runtime) runUserCheckWrapper(
	name string,
	chk *check,
	tagsFilter *tags.Filter,
	ctxer1 ctxctor1,
	maxSteps uint64,
) *fm.Clt_CallVerifProgress {
	v := &fm.Clt_CallVerifProgress{Name: name, Origin: fm.Clt_CallVerifProgress_after_response}
	log.Println("[NFO] checking user property:", v.Name)

	start := time.Now()
	errL := rt.runUserCheck(v, chk, tagsFilter, ctxer1, maxSteps)
	v.ElapsedNs = time.Since(start).Nanoseconds()
	if errL != nil {
		v.Reason = []string{fmt.Sprintf("%T", errL)}
		var reason string
		if e, ok := errL.(*starlark.EvalError); ok {
			reason = e.Backtrace()
		} else {
			reason = errL.Error()
		}
		v.Reason = append(v.Reason, strings.Split(reason, "\n")...)
	}
	return v
}

func (rt *Runtime) runUserCheck(
	v *fm.Clt_CallVerifProgress,
	chk *check,
	tagsFilter *tags.Filter,
	ctxer1 ctxctor1,
	maxSteps uint64,
) (err error) {
	// On success or skipping set status + return no error,
	// in all other cases just return error.

	if tagsFilter.Excludes(chk.tags) {
		v.Status = fm.Clt_CallVerifProgress_skipped
		return
	}

	th := &starlark.Thread{
		Name:  v.Name,
		Load:  loadDisabled,
		Print: func(_ *starlark.Thread, msg string) { rt.progress.Printf("%s", msg) },
	}
	th.SetMaxExecutionSteps(maxSteps) // Upper bound on computation

	snapshot := chk.state.String() // Assumes deterministic repr

	args := starlark.Tuple{ctxer1(chk.state)}

	defer func() { v.ExecutionSteps = th.ExecutionSteps() }()

	var hookRet starlark.Value
	if hookRet, err = starlark.Call(th, chk.afterResponse, args, nil); err != nil {
		err = errStateDict(v.Name, err)
		log.Println("[ERR]", err)
		// Check failed or an error happened
		v.Status = fm.Clt_CallVerifProgress_failure
		return
	}
	if err = starlarktruth.Close(th); err != nil {
		log.Println("[ERR]", err)
		// Incomplete assert.that() call
		v.Status = fm.Clt_CallVerifProgress_failure
		return
	}
	if hookRet != starlark.None {
		err = newUserError("hooks should return None, got: %s", hookRet.String())
		log.Println("[ERR]", err)
		v.Status = fm.Clt_CallVerifProgress_failure
		return
	}

	wasMutated := snapshot != chk.state.String()

	if wasMutated {
		if err = ensureStateDict(v.Name, chk.state); err != nil {
			log.Println("[ERR]", err)
			v.Status = fm.Clt_CallVerifProgress_failure
			return
		}
		// Ensure ctx.state is still proto-representable
		if err = starlarkvalue.ProtoCompatible(chk.state); err != nil {
			err = newUserError(err.Error())
			log.Println("[ERR]", err)
			v.Status = fm.Clt_CallVerifProgress_failure
			return
		}
	} else if !starlarktruth.Asserted(th) {
		// Predicate did not trigger
		v.Status = fm.Clt_CallVerifProgress_skipped
		return
	}

	// Check passed
	v.Status = fm.Clt_CallVerifProgress_success
	return
}
