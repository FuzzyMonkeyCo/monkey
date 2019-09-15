package lib

import (
	"fmt"
	"reflect"

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

func slValueFromInterface(x interface{}) (starlark.Value, error) {
	if x == nil {
		return starlark.None, nil
	}
	switch v := reflect.ValueOf(x); v.Kind() {
	case reflect.Bool:
		return starlark.Bool(v.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		return starlark.MakeInt64(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return starlark.MakeUint64(v.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return starlark.Float(v.Float()), nil
	case reflect.String:
		return starlark.String(v.String()), nil
	case reflect.Slice:
		values := make([]starlark.Value, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			s, err := slValueFromInterface(v.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			values = append(values, s)
		}
		return starlark.NewList(values), nil
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("expected string keys: %T", x)
		}
		values := starlark.NewDict(v.Len())
		for _, k := range v.MapKeys() {
			value := v.MapIndex(k)
			key := k.String()
			if !printableASCII(key) {
				return nil, fmt.Errorf("illegal string key: %q", key)
			}
			s, err := slValueFromInterface(value.Interface())
			if err != nil {
				return nil, err
			}
			if err = values.SetKey(starlark.String(key), s); err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		err := fmt.Errorf("not a JSON value: %T %+v", x, x)
		return nil, err
	}
}
