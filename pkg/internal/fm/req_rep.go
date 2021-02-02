package fm

import (
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/protovalue"
	"github.com/gogo/protobuf/types"
)

// Request/Response fields loosely follows Python's `requests` API

// Expose returns request data one can use in their call checks.
func (i *Clt_CallRequestRaw_Input) Expose() (s *types.Struct) {
	switch x := i.GetInput().(type) {

	case *Clt_CallRequestRaw_Input_HttpRequest_:
		reqProto := i.GetHttpRequest()
		s = &types.Struct{
			Fields: map[string]*types.Value{
				"method":  protovalue.FromGo(reqProto.Method),
				"url":     protovalue.FromGo(reqProto.Url),
				"content": protovalue.FromGo(string(reqProto.Body)),
			},
		}

		headers := make(map[string]*types.Value, len(reqProto.Headers))
		for key, values := range reqProto.Headers {
			vs := make([]*types.Value, 0, len(values.GetValues()))
			for _, value := range values.GetValues() {
				vs = append(vs, protovalue.FromGo(value))
			}
			headers[key] = &types.Value{Kind: &types.Value_ListValue{
				ListValue: &types.ListValue{Values: vs}}}
		}
		s.Fields["headers"] = &types.Value{Kind: &types.Value_StructValue{
			StructValue: &types.Struct{Fields: headers}}}

		if len(reqProto.Body) != 0 {
			s.Fields["body"] = reqProto.BodyDecoded
		}

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, i))
	}
	return
}

// Expose returns response data one can use in their call checks.
func (o *Clt_CallResponseRaw_Output) Expose(req *types.Struct) (s *types.Struct) {
	switch x := o.GetOutput().(type) {

	case *Clt_CallResponseRaw_Output_HttpResponse_:
		repProto := o.GetHttpResponse()
		s = &types.Struct{
			Fields: map[string]*types.Value{
				"request":     {Kind: &types.Value_StructValue{StructValue: req}},
				"status_code": protovalue.FromGo(repProto.StatusCode),
				"reason":      protovalue.FromGo(repProto.Reason),
				"content":     protovalue.FromGo(string(repProto.Body)),
				"elapsed_ns":  protovalue.FromGo(repProto.ElapsedNs),
				// "error": protovalue.FromGo(repProto.Error), Checks make this unreachable
				// "history" :: []Rep (redirects)?
			},
		}

		headers := make(map[string]*types.Value, len(repProto.Headers))
		for key, values := range repProto.Headers {
			vs := make([]*types.Value, 0, len(values.GetValues()))
			for _, value := range values.GetValues() {
				vs = append(vs, protovalue.FromGo(value))
			}
			headers[key] = &types.Value{Kind: &types.Value_ListValue{
				ListValue: &types.ListValue{Values: vs}}}
		}
		s.Fields["headers"] = &types.Value{Kind: &types.Value_StructValue{
			StructValue: &types.Struct{Fields: headers}}}

		if len(repProto.Body) != 0 {
			s.Fields["body"] = repProto.BodyDecoded
		}

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, o))
	}
	return
}
