package starlarkclone

import (
	"fmt"

	"go.starlark.net/starlark"
)

// A Cloner defines custom Value copying.
// See Clone.
type Cloner interface {
	Clone() (starlark.Value, error)
}

// Clone returns a copy of the given value.
func Clone(value starlark.Value) (dst starlark.Value, err error) {
	switch src := value.(type) {
	case Cloner:
		return src.Clone()

	case starlark.NoneType, starlark.Bool, starlark.Float:
		dst = src
		return

	case starlark.Int:
		dst = starlark.MakeBigInt(src.BigInt())
		return

	case starlark.String:
		dst = starlark.String(src.GoString())
		return

	case *starlark.List:
		n := src.Len()
		vs := make([]starlark.Value, 0, n)
		for i := 0; i < n; i++ {
			var v starlark.Value
			if v, err = Clone(src.Index(i)); err != nil {
				return
			}
			vs = append(vs, v)
		}
		dst = starlark.NewList(vs)
		return

	case starlark.Tuple:
		n := src.Len()
		vs := make([]starlark.Value, 0, n)
		for i := 0; i < n; i++ {
			var v starlark.Value
			if v, err = Clone(src.Index(i)); err != nil {
				return
			}
			vs = append(vs, v)
		}
		dst = starlark.Tuple(vs)
		return

	case *starlark.Dict:
		vs := starlark.NewDict(src.Len())
		for _, kv := range src.Items() {
			var k, v starlark.Value
			if k, err = Clone(kv.Index(0)); err != nil {
				return
			}
			if v, err = Clone(kv.Index(1)); err != nil {
				return
			}
			if err = vs.SetKey(k, v); err != nil {
				return
			}
		}
		dst = vs
		return

	default:
		err = fmt.Errorf("un-Clone-able value of type %s: %s", value.Type(), value.String())
		return
	}
}
