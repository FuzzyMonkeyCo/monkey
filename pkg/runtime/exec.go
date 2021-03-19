package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"go.starlark.net/repl"
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
		"None":      {},
		"True":      {},
		"False":     {},
		"any":       {},
		"all":       {},
		"bool":      {},
		"bytes":     {},
		"chr":       {},
		"dict":      {},
		"dir":       {},
		"enumerate": {},
		"float":     {},
		"getattr":   {},
		"hasattr":   {},
		"hash":      {},
		"int":       {},
		"len":       {},
		"list":      {},
		"max":       {},
		"min":       {},
		"ord":       {},
		"print":     {},
		"range":     {},
		"repr":      {},
		"reversed":  {},
		"set":       {},
		"sorted":    {},
		"str":       {},
		"tuple":     {},
		"type":      {},
		"zip":       {},
	}
	deny := map[string]struct{}{
		"fail": {},
	}
	starlarktruth.NewModule(starlark.Universe) // Adds assert.that()
	for f := range starlark.Universe {
		_, allowed := allow[f]
		_, denied := deny[f]
		switch {
		case allowed:
		case denied:
			delete(starlark.Universe, f)
		case f == starlarktruth.Default: // For check tests
		default:
			panic(fmt.Sprintf("unexpected builting %q", f))
		}
	}
}

func loadDisabled(_ *starlark.Thread, module string) (starlark.StringDict, error) {
	return nil, errors.New("load() disabled")
}

// JustExecREPL executes a Starlark Read-Eval-Print Loop
func (rt *Runtime) JustExecREPL() error {
	fmt.Println("# Welcome to Starlark https://go.starlark.net")
	rt.thread.Name = "REPL"
	rt.thread.Load = loadDisabled
	repl.REPL(rt.thread, rt.globals)
	return starlarktruth.Close(rt.thread)
}

// JustExecStart only executes SUT 'start'
func (rt *Runtime) JustExecStart() error {
	// FIXME: require a model name
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	resetter := mdl.GetResetter()
	resetter.Env(rt.envRead)
	return resetter.ExecStart(context.Background(), os.Stdout, os.Stderr, true)
}

// JustExecReset only executes SUT 'reset' which may be 'stop' followed by 'start'
func (rt *Runtime) JustExecReset() error {
	// FIXME: require a model name
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	resetter := mdl.GetResetter()
	resetter.Env(rt.envRead)
	return resetter.ExecReset(context.Background(), os.Stdout, os.Stderr, true)
}

// JustExecStop only executes SUT 'stop'
func (rt *Runtime) JustExecStop() error {
	// FIXME: require a model name
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	resetter := mdl.GetResetter()
	resetter.Env(rt.envRead)
	return resetter.ExecStop(context.Background(), os.Stdout, os.Stderr, true)
}
