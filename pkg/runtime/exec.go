package runtime

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
)

// JustExecStart only executes SUT 'start'
func (rt *Runtime) JustExecStart(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		stdout := newProgressWriter(fmter(os.Stdout))
		stderr := newProgressWriter(fmter(os.Stderr))
		return rsttr.ExecStart(ctx, stdout, stderr, true, rt.envRead)
	})
}

func fmter(w io.Writer) modeler.ShowFunc {
	return func(format string, args ...interface{}) { _, _ = fmt.Fprintf(w, format+"\n", args...) }
}

// JustExecReset only executes SUT 'reset' which may be 'stop' followed by 'start'
func (rt *Runtime) JustExecReset(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		// return rsttr.ExecReset(ctx, os.Stdout, os.Stderr, true, rt.envRead)
		//fixme: just find/make a package that makes an io.Writer from a func + a ptr
		stdout := newProgressWriter(fmter(os.Stdout))
		stderr := newProgressWriter(fmter(os.Stderr))
		return rsttr.ExecReset(ctx, stdout, stderr, true, rt.envRead)
	})
}

// JustExecStop only executes SUT 'stop'
func (rt *Runtime) JustExecStop(ctx context.Context) error {
	return rt.forEachResetter(func(name string, rsttr resetter.Interface) error {
		stdout := newProgressWriter(fmter(os.Stdout))
		stderr := newProgressWriter(fmter(os.Stderr))
		return rsttr.ExecStop(ctx, stdout, stderr, true, rt.envRead)
	})
}
