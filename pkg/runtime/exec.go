package runtime

import (
	"context"

	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
)

// JustExecStart only executes SUT 'start'
func (rt *Runtime) JustExecStart(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		return rsttr.ExecStart(ctx, &osShower{}, true, rt.envRead)
	})
}

// JustExecReset only executes SUT 'reset' which may be 'stop' followed by 'start'
func (rt *Runtime) JustExecReset(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		return rsttr.ExecReset(ctx, &osShower{}, true, rt.envRead)
	})
}

// JustExecStop only executes SUT 'stop'
func (rt *Runtime) JustExecStop(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		return rsttr.ExecStop(ctx, &osShower{}, true, rt.envRead)
	})
}
