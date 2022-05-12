package runtime

import (
	"context"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
)

// JustExecStart only executes SUT 'start'
func (rt *Runtime) JustExecStart(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		rsttr.Env(rt.envRead)
		return rsttr.ExecStart(ctx, os.Stdout, os.Stderr, true)
	})
}

// JustExecReset only executes SUT 'reset' which may be 'stop' followed by 'start'
func (rt *Runtime) JustExecReset(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		rsttr.Env(rt.envRead)
		return rsttr.ExecReset(ctx, os.Stdout, os.Stderr, true)
	})
}

// JustExecStop only executes SUT 'stop'
func (rt *Runtime) JustExecStop(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		rsttr.Env(rt.envRead)
		return rsttr.ExecStop(ctx, os.Stdout, os.Stderr, true)
	})
}
