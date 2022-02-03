package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func initExec() {
	resolve.AllowSet = true            // set([]) (no proto representation)
	resolve.AllowGlobalReassign = true // reassignment to top-level names
	//> Starlark programs cannot be Turing complete
	//> unless the -recursion flag is specified.
	resolve.AllowRecursion = false

	// TODO: set maxdepth https://github.com/google/starlark-go/issues/360

	allow := map[string]struct{}{
		"abs":       {},
		"all":       {},
		"any":       {},
		"bool":      {},
		"bytes":     {},
		"chr":       {},
		"dict":      {},
		"dir":       {},
		"enumerate": {},
		"False":     {},
		"float":     {},
		"getattr":   {},
		"hasattr":   {},
		"hash":      {},
		"int":       {},
		"len":       {},
		"list":      {},
		"max":       {},
		"min":       {},
		"None":      {},
		"ord":       {},
		"print":     {},
		"range":     {},
		"repr":      {},
		"reversed":  {},
		"set":       {},
		"sorted":    {},
		"str":       {},
		"True":      {},
		"tuple":     {},
		"type":      {},
		"zip":       {},
	}
	deny := map[string]struct{}{
		"fail": {},
	}
	starlarktruth.NewModule(starlark.Universe) // Adds `assert that()`
	for f := range starlark.Universe {
		_, allowed := allow[f]
		_, denied := deny[f]
		switch {
		case allowed:
		case denied:
			delete(starlark.Universe, f)
		case f == starlarktruth.Module: // For check tests
		default:
			panic(fmt.Sprintf("unexpected builtin %q", f))
		}
	}
}

func loadDisabled(_th *starlark.Thread, _module string) (starlark.StringDict, error) {
	return nil, errors.New("load() disabled")
}

// JustExecStart only executes SUT 'start'
func (rt *Runtime) JustExecStart(ctx context.Context) error {
	return rt.forEachSelectedResetter(ctx, func(name string, rsttr resetter.Interface) error {
		rsttr.Env(rt.envRead)
		return rsttr.ExecStart(ctx, os.Stdout, os.Stderr, true)
	})
}

// JustExecReset only executes SUT 'reset' which may be 'stop' followed by 'start'
func (rt *Runtime) JustExecReset(ctx context.Context) error {
	return rt.forEachSelectedResetter(ctx, func(name string, rsttr resetter.Interface) error {
		rsttr.Env(rt.envRead)
		return rsttr.ExecReset(ctx, os.Stdout, os.Stderr, true)
	})
}

// JustExecStop only executes SUT 'stop'
func (rt *Runtime) JustExecStop(ctx context.Context) error {
	return rt.forEachSelectedResetter(ctx, func(name string, rsttr resetter.Interface) error {
		rsttr.Env(rt.envRead)
		return rsttr.ExecStop(ctx, os.Stdout, os.Stderr, true)
	})
}
