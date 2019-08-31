package lib

import (
	"fmt"

	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func init() {
	// non-standard dialect flags
	///resolve.AllowNestedDef = true      // def statements within function bodies
	resolve.AllowLambda = true         // lambda x, y: (x,y)
	resolve.AllowFloat = true          // floating point
	resolve.AllowSet = true            // sets
	resolve.AllowGlobalReassign = true // reassignment to top-level names
	//> Starlark programs cannot be Turing complete
	//> unless the -recursion flag is specified.
	resolve.AllowRecursion = false
}

// DoExecREPL executes a Starlark Read-Eval-Print-Loop
func DoExecREPL() error {
	thread := &starlark.Thread{Load: repl.MakeLoad()}
	// globals, err := loadCfg([]byte{}, false)
	// if err != nil {
	// 	return err
	// }

	fmt.Println("Welcome to Starlark (go.starlark.net)")
	thread.Name = "REPL"
	// repl.REPL(thread, globals)
	repl.REPL(thread, starlark.StringDict{})
	return nil
}
