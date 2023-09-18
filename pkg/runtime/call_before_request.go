package runtime

import (
	"context"
	"log"
	"time"

	"go.starlark.net/starlark"
	"golang.org/x/sync/errgroup"

	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
)

func (chk *check) tryBeforeRequest(
	ctx context.Context,
	name string,
	req *ctxRequest_,
	print func(string),
	maxSteps uint64,
	maxDuration time.Duration,
) (err error) {
	ctxG, cancel := context.WithTimeout(ctx, maxDuration)
	defer cancel()

	g, ctxG := errgroup.WithContext(ctxG)
	g.Go(func() (err error) {
		th := &starlark.Thread{
			Name:  name,
			Load:  loadDisabled,
			Print: func(_ *starlark.Thread, msg string) { print(msg) },
		}

		th.SetMaxExecutionSteps(maxSteps) // Upper bound on computation
		var hookRet starlark.Value
		start := time.Now()
		defer func() {
			log.Printf("[DBG] check %q ran in %s (%d steps)", name, time.Since(start), th.ExecutionSteps())
		}()
		if hookRet, err = starlark.Call(th, chk.beforeRequest, starlark.Tuple{req}, nil); err != nil {
			log.Println("[ERR]", err)
			// Check failed or an error happened
			return
		}
		if err = starlarktruth.Close(th); err != nil {
			log.Println("[ERR]", err)
			// Incomplete `assert that()` call
			return
		}
		if hookRet != starlark.None {
			err = newUserError("check(name = %q) should return None, got: %s", name, hookRet.String())
			log.Println("[ERR]", err)
			return
		}
		// Check passed

		return
	})
	err = g.Wait()
	return
}
