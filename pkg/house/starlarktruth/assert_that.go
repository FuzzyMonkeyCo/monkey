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

func isFalse(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	if b, ok := t.actual.(starlark.Bool); ok && b == starlark.False {
		return starlark.None, nil
	}
	suffix := ""
	if !t.actual.Truth() {
		suffix = " However, it is falsy. Did you mean to call isFalsy() instead?"
	}
	return nil, t.failWithProposition("is False", suffix)
}

func isFalsy(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	if t.actual.Truth() {
		return nil, t.failWithProposition("is falsy", "")
	}
	return starlark.None, nil
}

func isTrue(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	if b, ok := t.actual.(starlark.Bool); ok && b == starlark.True {
		return starlark.None, nil
	}
	suffix := ""
	if t.actual.Truth() {
		suffix = " However, it is truthy. Did you mean to call isTruthy() instead?"
	}
	return nil, t.failWithProposition("is True", suffix)
}

func isTruthy(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	if !t.actual.Truth() {
		return nil, t.failWithProposition("is truthy", "")
	}
	return starlark.None, nil
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
