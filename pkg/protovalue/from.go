package protovalue

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// FromGo compiles a Go value to google.protobuf.Value.
// It panics on unexpected types.
// See https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#google.protobuf.Value
func FromGo(value interface{}) *structpb.Value {
	if value == nil {
		return &structpb.Value{Kind: &structpb.Value_NullValue{
			NullValue: structpb.NullValue_NULL_VALUE}}
	}
	switch val := value.(type) {
	case bool:
		return &structpb.Value{Kind: &structpb.Value_BoolValue{
			BoolValue: val}}

	case uint8:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case int8:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case uint16:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case int16:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case uint32:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case int32:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case uint64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case int64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}

	case float32:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(val)}}
	case float64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: val}}

	case string:
		return &structpb.Value{Kind: &structpb.Value_StringValue{
			StringValue: val}}

	case []interface{}:
		vs := make([]*structpb.Value, 0, len(val))
		for _, v := range val {
			vs = append(vs, FromGo(v))
		}
		return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: vs}}}

	case map[string]interface{}:
		vs := make(map[string]*structpb.Value, len(val))
		for n, v := range val {
			vs[n] = FromGo(v)
		}
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: vs}}}

	default:
		panic(fmt.Errorf("cannot convert from value type: %T", val))
	}
}
