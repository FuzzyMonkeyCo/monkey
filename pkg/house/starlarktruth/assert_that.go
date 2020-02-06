package starlarktruth

import (
	"errors"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// Maximum nesting browsed when comparing values
const maxdepth = 10

// const (
// 	tyNoneType = "NoneType"
// 	tyBool="bool"
// 	tyInt="int"
// 	tyFloat="float"
// 	tyString="string"
// 	tyList ="list" // ptr
// 	tyTuple="tuple"
// 	tyDict="dict" // ptr
// 	tySet="set" // ptr
// 	// *Function
// 	// *Builtin
// )

func named(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	str, ok := args[0].(starlark.String)
	if !ok {
		return nil, errors.New(".named() expects a string")
	}
	t.name = str.GoString()
	return t, nil
}

func comparable(t *T, b *starlark.Builtin, verb string, op syntax.Token, other starlark.Value) (starlark.Value, error) {
	if err := t.checkNone(b.Name(), other); err != nil {
		return nil, err
	}
	ok, err := starlark.CompareDepth(op, t.actual, other, maxdepth)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, t.failComparingValues(verb, other, "")
	}
	return starlark.None, nil
}

func isAtLeast(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is at least", syntax.LT, args[0])
}

func isAtMost(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is at most", syntax.GT, args[0])
}

func isGreaterThan(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is greater than", syntax.LE, args[0])
}

func isLessThan(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is less than", syntax.GE, args[0])
}
