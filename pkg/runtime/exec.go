package runtime

import (
	"context"
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"go.starlark.net/repl"
	"go.starlark.net/resolve"
)

func init() {
	resolve.AllowNestedDef = false     // def statements within function bodies
	resolve.AllowLambda = true         // lambda x, y: (x,y)
	resolve.AllowFloat = true          // floating point
	resolve.AllowSet = true            // sets
	resolve.AllowGlobalReassign = true // reassignment to top-level names
	//> Starlark programs cannot be Turing complete
	//> unless the -recursion flag is specified.
	resolve.AllowRecursion = false
}

// JustExecREPL executes a Starlark Read-Eval-Print Loop
func (rt *runtime) JustExecREPL() error {
	rt.thread.Load = repl.MakeLoad()
	fmt.Println("Welcome to Starlark (go.starlark.net)")
	rt.thread.Name = "REPL"
	repl.REPL(rt.thread, rt.globals)
	return nil
}

// JustExecStart only executes SUT 'start'
func (rt *runtime) JustExecStart() error {
	// FIXME: require a model name
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	resetter := mdl.GetResetter()
	return resetter.ExecStart(context.Background(), nil)
}

// JustExecReset only executes SUT 'reset'
func (rt *runtime) JustExecReset() error {
	// FIXME: require a model name
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	resetter := mdl.GetResetter()
	return resetter.ExecReset(context.Background(), nil)
}

// JustExecStop only executes SUT 'stop'
func (rt *runtime) JustExecStop() error {
	// FIXME: require a model name
	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	resetter := mdl.GetResetter()
	return resetter.ExecStop(context.Background(), nil)
}
