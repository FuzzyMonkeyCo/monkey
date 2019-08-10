package lib

import (
	"fmt"

	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func init() {
	// non-standard dialect flags
	resolve.AllowFloat = true  // allow floating-point numbers
	resolve.AllowSet = true    // allow set data type
	resolve.AllowLambda = true // allow lambda expressions
}

// DoExecREPL executes a Starlark Read-Eval-Print-Loop
func DoExecREPL() {
	thread := &starlark.Thread{Load: repl.MakeLoad()}
	globals := make(starlark.StringDict)

	fmt.Println("Welcome to Starlark (go.starlark.net)")
	thread.Name = "REPL"
	repl.REPL(thread, globals)

	// for _, name := range globals.Keys() {
	// 	if !strings.HasPrefix(name, "_") {
	// 		fmt.Fprintf(os.Stderr, "%s = %s\n", name, globals[name])
	// 	}
	// }
}
