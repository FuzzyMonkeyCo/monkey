package starlarkvalue

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

// FromProtoValue converts a Google Well-Known-Type Value to a Starlark value.
// Panics on unexpected proto value.
func FromProtoValue(x *types.Value) starlark.Value {
	switch x.GetKind().(type) {

	case *types.Value_NullValue:
		return starlark.None

	case *types.Value_BoolValue:
		return starlark.Bool(x.GetBoolValue())

	case *types.Value_NumberValue:
		return starlark.Float(x.GetNumberValue())

	case *types.Value_StringValue:
		return starlark.String(x.GetStringValue())

	case *types.Value_ListValue:
		xs := x.GetListValue().GetValues()
		values := make([]starlark.Value, 0, len(xs))
		for _, x := range xs {
			value := FromProtoValue(x)
			values = append(values, value)
		}
		return starlark.NewList(values)

	case *types.Value_StructValue:
		kvs := x.GetStructValue().GetFields()
		values := starlark.NewDict(len(kvs))
		for k, v := range kvs {
			value := FromProtoValue(v)
			key := starlark.String(k)
			if err := values.SetKey(key, value); err != nil {
				panic(err) // unreachable: hashable key, not iterating, not frozen.
			}
		}
		return values

	default:
		panic(fmt.Errorf("unhandled: %T %+v", x.GetKind(), x)) // unreachable: only proto values.
	}
}
