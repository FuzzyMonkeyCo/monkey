package runtime

import (
	"context"
	"fmt"
	"os"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarktruth"
	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func initExec() {
	resolve.AllowNestedDef = false     // def statements within function bodies
	resolve.AllowLambda = true         // lambda x, y: (x,y)
	resolve.AllowFloat = true          // floating point
	resolve.AllowSet = false           // sets (no proto representation yet)
	resolve.AllowGlobalReassign = true // reassignment to top-level names
	//> Starlark programs cannot be Turing complete
	//> unless the -recursion flag is specified.
	resolve.AllowRecursion = false

	starlarktruth.NewModule(starlark.Universe)
}

// JustExecREPL executes a Starlark Read-Eval-Print Loop
func (rt *Runtime) JustExecREPL() error {
	rt.thread.Load = repl.MakeLoad()
	fmt.Println("Welcome to Starlark (go.starlark.net)")
	rt.thread.Name = "REPL"
	repl.REPL(rt.thread, rt.globals)
	return nil
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