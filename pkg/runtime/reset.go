package runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter/shell"
	"go.starlark.net/starlark"
)

// Cleanup ensures that resetters are terminated
func (rt *Runtime) Cleanup(ctx context.Context) (err error) {
	if rt.cleanedup {
		return
	}

	log.Println("[NFO] terminating resetter")
	var rsttr resetter.Interface
	for _, mdl := range rt.models {
		rsttr = mdl.GetResetter()
		break
	}
	if errR := rsttr.Terminate(ctx, os.Stdout, os.Stderr); errR != nil && err == nil {
		err = errR
		// Keep going
	}

	rt.cleanedup = true
	return
}

func (rt *Runtime) reset(ctx context.Context) error {
	const showp = "Resetting system under test..."
	rt.progress.Printf(showp + "\n")

	rp := func(msg *fm.Clt_ResetProgress) *fm.Clt {
		return &fm.Clt{Msg: &fm.Clt_ResetProgress_{ResetProgress: msg}}
	}

	if errT := rt.client.Send(ctx, rp(&fm.Clt_ResetProgress{
		Status: fm.Clt_ResetProgress_started,
	})); errT != nil {
		log.Println("[ERR]", errT)
		return errT
	}

	start := time.Now()
	errL := rt.runReset(ctx)
	elapsed := time.Since(start).Nanoseconds()
	if errL != nil {
		log.Println("[ERR] ExecReset:", errL)
		var reason []string
		if resetErr, ok := errL.(*resetter.Error); ok {
			reason = resetErr.Reason()
		} else {
			reason = strings.Split(errL.Error(), "\n")
		}

		if errT := rt.client.Send(ctx, rp(&fm.Clt_ResetProgress{
			Status:    fm.Clt_ResetProgress_failed,
			ElapsedNs: elapsed,
			Reason:    reason,
		})); errT != nil {
			log.Println("[ERR]", errT)
			return errT
		}

		rt.progress.Errorf(showp + " failed!\n")
		return nil // Don't end fuzz loop due to SUT error
	}

	if errT := rt.client.Send(ctx, rp(&fm.Clt_ResetProgress{
		Status:    fm.Clt_ResetProgress_ended,
		ElapsedNs: elapsed,
	})); errT != nil {
		log.Println("[ERR]", errT)
		return errT
	}

	rt.progress.Printf(showp + " done.\n")
	return nil
}

func (rt *Runtime) runReset(ctx context.Context) (err error) {
	var rsttr resetter.Interface
	for _, mdl := range rt.models {
		rsttr = mdl.GetResetter()
		break
	}

	{
		var state starlark.Value
		if state, err = slValueCopy(rt.modelState0); err != nil {
			log.Println("[ERR]", err)
			return
		}
		rt.modelState = state.(*modelState)
		log.Println("[NFO] re-initialized model state")
	}

	stdout := newProgressWriter(rt.progress.Printf)
	stderr := newProgressWriter(rt.progress.Errorf)
	err = rsttr.ExecReset(ctx, stdout, stderr, false)
	return
}

func newFromKwargs(modelerName string, r starlark.StringDict) (resetter.Interface, error) {
	const (
		tExecReset = "ExecReset"
		tExecStart = "ExecStart"
		tExecStop  = "ExecStop"
	)
	var (
		ok bool
		v  starlark.Value
		vv starlark.String
		t  string
		// TODO: other Resetter.s
		rsttr = &shell.Resetter{}
	)
	t = tExecStart
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Start = vv.GoString()
	}
	t = tExecReset
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Rst = vv.GoString()
	}
	t = tExecStop
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Stop = vv.GoString()
	}
	if len(r) != 0 {
		return nil, fmt.Errorf("unexpected arguments to %s(): %s", modelerName, strings.Join(r.Keys(), ", "))
	}
	return rsttr, nil
}
