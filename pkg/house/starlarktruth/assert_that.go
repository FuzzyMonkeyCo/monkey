package starlarktruth

import (
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

func IsAtLeast(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is at least", syntax.LT, args[0])
}

func IsAtMost(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is at most", syntax.GT, args[0])
}

func IsGreaterThan(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is greater than", syntax.LE, args[0])
}

func IsLessThan(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is less than", syntax.GE, args[0])
}
