package starlarkvalue

import (
	"fmt"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToProtoValue converts a Starlark value to a Google Well-Known-Type value.
// Panics on unexpected starlark value.
func ToProtoValue(x starlark.Value) *structpb.Value {
	switch x := x.(type) {

	case starlark.NoneType:
		return &structpb.Value{Kind: &structpb.Value_NullValue{
			NullValue: structpb.NullValue_NULL_VALUE}}

	case starlark.Bool:
		return &structpb.Value{Kind: &structpb.Value_BoolValue{
			BoolValue: bool(x)}}

	case starlark.Float:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{
			NumberValue: float64(x)}}

	case starlark.Int:
		if x, ok := x.Int64(); ok {
			return &structpb.Value{Kind: &structpb.Value_NumberValue{
				NumberValue: float64(x)}}
		}
		panic(fmt.Errorf("unexpected value of type %s: %s", x.Type(), x.String()))

	case starlark.String:
		return &structpb.Value{Kind: &structpb.Value_StringValue{
			StringValue: x.GoString()}}

	case *starlark.List:
		n := x.Len()
		vs := make([]*structpb.Value, 0, n)
		for i := 0; i < n; i++ {
			vs = append(vs, ToProtoValue(x.Index(i)))
		}
		return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
			Values: vs}}}

	// case starlark.Tuple: // Encodes starlark.Tuple as array.
	// 	n := x.Len()
	// 	vs := make([]*structpb.Value, 0, n)
	// 	for i := 0; i < n; i++ {
	// 		vs = append(vs, ToProtoValue(x.Index(i)))
	// 	}
	// 	return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
	// 		Values: vs}}}

	case *starlark.Dict:
		vs := make(map[string]*structpb.Value, x.Len())
		for _, kv := range x.Items() {
			k := kv.Index(0).(starlark.String).GoString()
			v := ToProtoValue(kv.Index(1))
			vs[k] = v
		}
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
			Fields: vs}}}

	default:
		panic(fmt.Errorf("unexpected value of type %s: %s", x.Type(), x.String()))
	}
}
