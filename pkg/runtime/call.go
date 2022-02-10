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

func (rt *Runtime) call(ctx context.Context, msg *fm.Srv_Call, tagsFilter *tags.Filter, maxSteps uint64, maxDuration time.Duration) error {
	showf := func(format string, s ...interface{}) {
		rt.progress.Printf(format, s...)
	}

	log.Printf("[NFO] raw input: %.999v", msg.GetInput())
	mdl := rt.models[msg.GetModelName()]
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
		print := func(msg string) { rt.progress.Printf("%s", msg) }
		ctxer1 := ctxer2(output.GetOutput())
		var passed2 bool
		if passed2, errT = rt.userChecks(ctx, print, tagsFilter, ctxer1, maxSteps, maxDuration); errT != nil {
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

func (rt *Runtime) userChecks(
	ctx context.Context,
	print func(string),
	tagsFilter *tags.Filter,
	ctxer1 ctxctor1,
	maxSteps uint64,
	maxDuration time.Duration,
) (bool, error) {
	log.Printf("[NFO] checking %d user properties", len(rt.checks))

	// Run all checks concurrently, send their results to vs.
	// Concurrently, consume and send these one-by-one.
	// Return early if ctx is canceled.

	ctxG, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()

	g, ctxG := errgroup.WithContext(ctxG)
	vs := make(chan *fm.Clt_CallVerifProgress, len(rt.checks))
	threads := rt.makeThreads(ctxG)

	passed := true
	g.Go(func() (errT error) {
		for name := range rt.checks {
			name := name
			select {
			case <-ctxG.Done():
				errT, passed = ctxG.Err(), false
				if errT != nil {
					threads[name].Cancel(errT.Error())
				}
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

	_ = rt.forEachCheck(func(name string, chk *check) error {
		g.Go(func() error {
			th := threads[name]
			v := rt.runUserCheckWrapper(name, th, chk, print, tagsFilter, ctxer1, maxSteps)
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
		return nil
	})

	if errT := g.Wait(); errT != nil {
		return false, errT
	}
	return passed, nil
}

func (rt *Runtime) runUserCheckWrapper(
	name string,
	th *starlark.Thread,
	chk *check,
	print func(string),
	tagsFilter *tags.Filter,
	ctxer1 ctxctor1,
	maxSteps uint64,
) *fm.Clt_CallVerifProgress {
	v := &fm.Clt_CallVerifProgress{Name: name, Origin: fm.Clt_CallVerifProgress_after_response}
	log.Println("[NFO] checking user property:", v.Name)

	start := time.Now()
	errL := rt.runUserCheck(v, th, chk, print, tagsFilter, ctxer1, maxSteps)
	v.ElapsedNs = time.Since(start).Nanoseconds()
	if errL != nil {
		v.Reason = []string{fmt.Sprintf("%T", errL)}
		reason := starTrickError(errL).Error()
		v.Reason = append(v.Reason, strings.Split(reason, "\n")...)
	}
	return v
}

func (rt *Runtime) makeThreads(ctx context.Context) map[string]*starlark.Thread {
	threads := make(map[string]*starlark.Thread, len(rt.checks))
	for name := range rt.checks {
		th := &starlark.Thread{
			Name: name,
			Load: loadDisabled,
		}
		th.SetLocal("ctx", ctx)
		threads[name] = th
	}
	return threads
}

func (rt *Runtime) runUserCheck(
	v *fm.Clt_CallVerifProgress,
	th *starlark.Thread,
	chk *check,
	print func(string),
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

	didPrint := false
	th.Print = func(_ *starlark.Thread, msg string) {
		didPrint = true
		print(msg)
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
		// Incomplete `assert that()` call
		v.Status = fm.Clt_CallVerifProgress_failure
		return
	}
	if hookRet != starlark.None {
		err = newUserError("hooks should return None, got: %s", hookRet.String())
		log.Println("[ERR]", err)
		v.Status = fm.Clt_CallVerifProgress_failure
		return
	}

	if wasMutated := snapshot != chk.state.String(); wasMutated {
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
	} else if !didPrint && !starlarktruth.Asserted(th) {
		// Predicate did not trigger
		v.Status = fm.Clt_CallVerifProgress_skipped
		return
	}

	// Check passed
	v.Status = fm.Clt_CallVerifProgress_success
	return
}
