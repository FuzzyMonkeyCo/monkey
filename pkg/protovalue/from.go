package protovalue

import (
	"fmt"

	"github.com/gogo/protobuf/types"
)

// FromGo compiles a Go value to google.protobuf.Value.
// It panics on unexpected types.
// See https://developers.google.com/protocol-buffers/docs/reference/google.protobuf#google.protobuf.Value
func FromGo(value interface{}) *types.Value {
	if value == nil {
		return &types.Value{Kind: &types.Value_NullValue{
			NullValue: types.NullValue_NULL_VALUE}}
	}
	switch val := value.(type) {
	case bool:
		return &types.Value{Kind: &types.Value_BoolValue{
			BoolValue: val}}
	case uint8:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case int8:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case uint16:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case int16:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case uint32:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case int32:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case uint64:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case int64:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case float32:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: float64(val)}}
	case float64:
		return &types.Value{Kind: &types.Value_NumberValue{
			NumberValue: val}}
	case string:
		return &types.Value{Kind: &types.Value_StringValue{
			StringValue: val}}
	case []interface{}:
		vs := make([]*types.Value, 0, len(val))
		for _, v := range val {
			vs = append(vs, FromGo(v))
		}
		return &types.Value{Kind: &types.Value_ListValue{ListValue: &types.ListValue{Values: vs}}}
	case map[string]interface{}:
		vs := make(map[string]*types.Value, len(val))
		for n, v := range val {
			vs[n] = FromGo(v)
		}
		return &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{Fields: vs}}}
	default:
		panic(fmt.Errorf("cannot convert from value type: %T", val))
	}
}
