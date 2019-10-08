package lib

import (
	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

// InitExec specifies Monkey's dialect flags
func InitExec() {
	resolve.AllowNestedDef = false     // def statements within function bodies
	resolve.AllowLambda = true         // lambda x, y: (x,y)
	resolve.AllowFloat = true          // floating point
	resolve.AllowSet = true            // sets
	resolve.AllowGlobalReassign = true // reassignment to top-level names
	//> Starlark programs cannot be Turing complete
	//> unless the -recursion flag is specified.
	resolve.AllowRecursion = false

	RegisterModeler("OpenAPIv3", modelerOpenAPIv3)
}

// DoExecREPL executes a Starlark Read-Eval-Print-Loop
func DoExecREPL() error {
	thread := &starlark.Thread{Load: repl.MakeLoad()}
	// TODO: load user globals so user can try things

	fmt.Println("Welcome to Starlark (go.starlark.net)")
	thread.Name = "REPL"
	repl.REPL(thread, starlark.StringDict{})
	return nil
}
