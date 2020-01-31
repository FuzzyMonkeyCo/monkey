package starlarktruth

import (
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

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

const maxdepth = 10

func isAtMost(t *T, b *starlark.Builtin, args ...starlark.Value) (starlark.Value, error) {
	other := args[0]
	if err := t.checkNone(b.Name(), other); err != nil {
		return nil, err
	}
	ok, err := starlark.CompareDepth(syntax.GT, t.actual, other, maxdepth)
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, t.failComparingValues("is at most", other, "")
	}
	return starlark.None, nil
}
