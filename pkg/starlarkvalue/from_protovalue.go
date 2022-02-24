package starlarkvalue

import (
	"fmt"
	"math"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"
)

// ProtoCompatible returns a non-nil error when the Starlark value
// has no trivial Protocol Buffers Well-Known-Types representation.
func ProtoCompatible(value starlark.Value) (err error) {
	switch v := value.(type) {
	case starlark.NoneType:
		return
	case starlark.Bool:
		return
	case starlark.Int:
		return
	case starlark.Float:
		if !isFinite(float64(v)) {
			return fmt.Errorf("non-finite float: %v", v)
		}
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
		err = fmt.Errorf("incompatible value (%s): %s", value.Type(), value.String())
		return
	}
}

// FromProtoValue converts a Google Well-Known-Type Value to a Starlark value.
// Panics on unexpected proto value.
func FromProtoValue(x *structpb.Value) starlark.Value {
	switch x.GetKind().(type) {

	case *structpb.Value_NullValue:
		return starlark.None

	case *structpb.Value_BoolValue:
		return starlark.Bool(x.GetBoolValue())

	case *structpb.Value_NumberValue:
		return starlark.Float(x.GetNumberValue())

	case *structpb.Value_StringValue:
		return starlark.String(x.GetStringValue())

	case *structpb.Value_ListValue:
		xs := x.GetListValue().GetValues()
		values := make([]starlark.Value, 0, len(xs))
		for _, x := range xs {
			value := FromProtoValue(x)
			values = append(values, value)
		}
		return starlark.NewList(values)

	case *structpb.Value_StructValue:
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

// isFinite reports whether f represents a finite rational value.
// It is equivalent to !math.IsNan(f) && !math.IsInf(f, 0).
func isFinite(f float64) bool {
	return math.Abs(f) <= math.MaxFloat64
}
