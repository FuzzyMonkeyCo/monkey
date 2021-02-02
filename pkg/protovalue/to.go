package protovalue

import (
	"fmt"

	"github.com/gogo/protobuf/types"
)

// ToGo compiles a google.protobuf.Value value to Go.
// It panics on unexpected types.
// See https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#google.protobuf.Value
func ToGo(value *types.Value) interface{} {
	switch value.GetKind().(type) {
	case *types.Value_NullValue:
		return nil
	case *types.Value_BoolValue:
		return value.GetBoolValue()
	case *types.Value_NumberValue:
		return value.GetNumberValue()
	case *types.Value_StringValue:
		return value.GetStringValue()
	case *types.Value_ListValue:
		val := value.GetListValue().GetValues()
		vs := make([]interface{}, 0, len(val))
		for _, v := range val {
			vs = append(vs, ToGo(v))
		}
		return vs
	case *types.Value_StructValue:
		val := value.GetStructValue().GetFields()
		vs := make(map[string]interface{}, len(val))
		for n, v := range val {
			vs[n] = ToGo(v)
		}
		return vs
	default:
		panic(fmt.Errorf("cannot convert from type %T: %+v", value.GetKind(), value))
	}
}
