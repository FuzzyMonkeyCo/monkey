package protovalue

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// ToGo compiles a google.protobuf.Value value to Go.
// It panics on unexpected types.
// See https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#google.protobuf.Value
func ToGo(value *structpb.Value) interface{} {
	switch value.GetKind().(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_BoolValue:
		return value.GetBoolValue()
	case *structpb.Value_NumberValue:
		return value.GetNumberValue()
	case *structpb.Value_StringValue:
		return value.GetStringValue()
	case *structpb.Value_ListValue:
		val := value.GetListValue().GetValues()
		vs := make([]interface{}, 0, len(val))
		for _, v := range val {
			vs = append(vs, ToGo(v))
		}
		return vs
	case *structpb.Value_StructValue:
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
