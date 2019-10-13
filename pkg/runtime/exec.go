package runtime

import (
	"context"
	"fmt"

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
	resetter := rt.models[0].GetResetter()
	return resetter.ExecStart(context.Background(), nil)
}

// JustExecReset only executes SUT 'reset'
func (rt *runtime) JustExecReset() error {
	resetter := rt.models[0].GetResetter()
	return resetter.ExecReset(context.Background(), nil)
}

// JustExecStop only executes SUT 'stop'
func (rt *runtime) JustExecStop() error {
	resetter := rt.models[0].GetResetter()
	return resetter.ExecStop(context.Background(), nil)
}
