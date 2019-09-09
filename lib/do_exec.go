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

func slValueFromInterface(v interface{}) (starlark.Value, error) {
	switch vv := v.(type) {
	case nil:
		return starlark.None, nil
	case bool:
		return starlark.Bool(vv), nil
	case int64:
		return starlark.MakeInt64(vv), nil
	case float64:
		return starlark.Float(vv), nil
	case string:
		return starlark.String(vv), nil
	case []interface{}:
		values := make([]starlark.Value, 0, len(vv))
		for _, value := range vv {
			var vvv starlark.Value
			var err error
			if vvv, err = slValueFromInterface(value); err != nil {
				return nil, err
			}
			values = append(values, vvv)
		}
		return starlark.NewList(values), nil
	case map[string]interface{}:
		values := starlark.NewDict(len(vv))
		for key, value := range vv {
			var vvv starlark.Value
			var err error
			if vvv, err = slValueFromInterface(value); err != nil {
				return nil, err
			}
			if err = values.SetKey(starlark.String(key), vvv); err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		err := fmt.Errorf("not a JSON value: %v", v)
		return nil, err
	}
}
