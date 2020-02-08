package starlarktruth

import (
	"errors"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
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

func isEqualTo(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	switch other := args[0].(type) {
	case starlark.String:
		if actual, ok := t.actual.(starlark.String); ok {
			a := actual.GoString()
			o := other.GoString()
			// Use unified diff strategy when comparing multiline strings.
			if strings.Contains(a, "\n") && strings.Contains(o, "\n") {
				diff := difflib.ContextDiff{
					A:        difflib.SplitLines(o),
					B:        difflib.SplitLines(a),
					FromFile: "Expected",
					ToFile:   "Actual",
					Context:  3,
					Eol:      "\n",
				}
				pretty, err := difflib.GetContextDiffString(diff)
				if err != nil {
					return nil, err
				}
				pretty = strings.Replace(pretty, "\t", " ", -1)
				msg := "is equal to expected, found diff:\n" + pretty
				return nil, t.failWithProposition(msg, "")
			}
		}
	case starlark.Sliceable: // e.g. tuple, list
		return containsExactlyElementsInOrderIn(t, b, other)
	case starlark.Iterable: // e.g. dict, set
		return containsExactlyElementsIn(other)
	default:
		ok, err := starlark.CompareDepth(syntax.EQL, t.actual, other, maxdepth)
		if err != nil {
			return nil, err
		}
		if !ok {
			suffix := ""
			if t.actual.String() == other.String() {
				suffix = " However, their str() representations are equal."
			}
			return nil, t.failComparingValues("is equal to", other, suffix)
		}
		return starlark.None, nil
	}
}

func named(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	str, ok := args[0].(starlark.String)
	if !ok || str.Len() == 0 {
		return nil, errors.New("named() expects a (non empty) string")
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

func (t *T) comparable(bName, verb string, op syntax.Token, other starlark.Value) (starlark.Value, error) {
	if err := t.checkNone(bName, other); err != nil {
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

func isAtLeast(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("isAtLeast", "is at least", syntax.LT, args[0])
}

func isAtMost(t *T, args ...starlark.Value) (starlark.Value, error) {
	return t.comparable("isAtMost", "is at most", syntax.GT, args[0])
}

func isGreaterThan(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is greater than", syntax.LE, args[0])
}

func isLessThan(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	return comparable(t, b, "is less than", syntax.GE, args[0])
}
