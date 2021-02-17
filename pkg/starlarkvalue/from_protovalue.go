package starlarkvalue

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

// ProtoCompatible returns a non-nil error when the Starlark value
// has no trivial Protocol Buffers Well-Known-Types representation.
func ProtoCompatible(value starlark.Value) (err error) {
	switch v := value.(type) {
	case starlark.NoneType:
		return
	case starlark.Bool:
		return
	case starlark.Int, starlark.Float:
		return
	case starlark.String:
		return
	case *starlark.List:
		for i := 0; i < v.Len(); i++ {
			if err = ProtoCompatible(v.Index(i)); err != nil {
				return
			}
		}
		return
	case starlark.Tuple:
		for i := 0; i < v.Len(); i++ {
			if err = ProtoCompatible(v.Index(i)); err != nil {
				return
			}
		}
		return
	case *starlark.Dict:
		for _, kv := range v.Items() {
			if _, ok := kv.Index(0).(starlark.String); !ok {
				err = fmt.Errorf("want string key, got: (%s) %s", value.Type(), value.String())
				return
			}
			if err = ProtoCompatible(kv.Index(1)); err != nil {
				return
			}
		}
		return
	default:
		err = fmt.Errorf("incompatible value %T: %s", value, value.String())
		return
	}
}

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
