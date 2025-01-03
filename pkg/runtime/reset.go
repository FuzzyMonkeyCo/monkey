package runtime

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/as"
	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
)

// Cleanup ensures that resetters are terminated
func (rt *Runtime) Cleanup(ctx context.Context) (err error) {
	as.ColorNFO.Println()
	as.ColorWRN.Printf("Ran for %s.\n", time.Since(rt.fuzzingStartedAt))
	if rt.cleanedup {
		return
	}
	as.ColorNFO.Println("Cleaning up...")

	log.Println("[NFO] terminating resetter")
	if errR := rt.forEachSelectedResetter(ctx, func(name string, rsttr resetter.Interface) error {
		return rsttr.Terminate(ctx, &osShower{}, rt.envRead)
	}); errR != nil {
		err = errR
		// Keep going
	}

	rt.cleanedup = true
	return
}

func (rt *Runtime) reset(ctx context.Context) (errL, errT error) {
	const showp = "Resetting system under test..."
	rt.progress.Printf(showp + "\n")

	rp := func(msg *fm.Clt_ResetProgress) *fm.Clt {
		return &fm.Clt{Msg: &fm.Clt_ResetProgress_{ResetProgress: msg}}
	}

	if errT = rt.client.Send(ctx, rp(&fm.Clt_ResetProgress{
		Status: fm.Clt_ResetProgress_started,
	})); errT != nil {
		log.Println("[ERR]", errT)
		return
	}

	start := time.Now()
	errL = rt.runReset(ctx)
	elapsed := time.Since(start).Nanoseconds()
	if errL != nil {
		log.Println("[ERR] exec'd:", errL)

		var reason []string
		if resetErr, ok := errL.(*resetter.Error); ok {
			reason = resetErr.Reason()
		} else {
			reason = strings.Split(errL.Error(), "\n")
		}

		if errT = rt.client.Send(ctx, rp(&fm.Clt_ResetProgress{
			Status:    fm.Clt_ResetProgress_failed,
			ElapsedNs: elapsed,
			Reason:    reason,
		})); errT != nil {
			log.Println("[ERR]", errT)
			return
		}

		if strings.Contains(errL.Error(), context.Canceled.Error()) {
			rt.progress.Errorf(showp + " failed! (timed out)\n")
		} else {
			rt.progress.Errorf(showp + " failed!\n")
		}
		return
	}

	if errT = rt.client.Send(ctx, rp(&fm.Clt_ResetProgress{
		Status:    fm.Clt_ResetProgress_ended,
		ElapsedNs: elapsed,
	})); errT != nil {
		log.Println("[ERR]", errT)
		return
	}

	rt.progress.Printf(showp + " done.\n")
	return
}

func (rt *Runtime) runReset(ctx context.Context) (err error) {
	if err = rt.forEachAfterResponseCheck(func(name string, chk *check) error {
		if err := chk.reset(name); err != nil {
			log.Println("[ERR]", err)
			return err
		}
		return nil
	}); err != nil {
		return
	}
	log.Println("[NFO] re-initialized model state")

	return rt.forEachSelectedResetter(ctx, func(name string, rsttr resetter.Interface) error {
		return rsttr.ExecReset(ctx, rt.progress, false, rt.envRead)
	})
}
