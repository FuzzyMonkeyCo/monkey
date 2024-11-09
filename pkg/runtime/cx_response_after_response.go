package runtime

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
)

const (
	cxResponseHttp = "http_response"
)

// cxResponseAfterResponse is the `ctx.response` starlark value accessible after executing a call
type cxResponseAfterResponse struct {
	ty string

	attrs     starlark.StringDict
	attrnames []string

	protoBodyDecoded *structpb.Value
	body             starlark.Value

	protoHeaders []*fm.HeaderPair
	headers      starlark.Value //FIXME: cxHeaders + Freeze
}

func newCxResponseAfterResponse(o *fm.Clt_CallResponseRaw_Output) (cr *cxResponseAfterResponse) {
	cr = &cxResponseAfterResponse{
		attrs: make(starlark.StringDict, 5),
	}
	switch x := o.GetOutput().(type) {

	case *fm.Clt_CallResponseRaw_Output_HttpResponse_:
		cr.ty = cxResponseHttp

		repProto := o.GetHttpResponse()
		cr.attrs["status_code"] = starlark.MakeUint(uint(repProto.StatusCode))
		cr.attrs["reason"] = starlark.String(repProto.Reason)
		cr.attrs["content"] = starlark.String(repProto.Body)
		cr.attrs["elapsed_ns"] = starlark.MakeInt64(repProto.ElapsedNs)
		cr.attrs["elapsed_ms"] = starlark.MakeInt64(repProto.ElapsedNs / 1.e6)
		// "error": repProto.Error Checks make this unreachable
		// "history" :: []Rep (redirects)?
		cr.protoHeaders = repProto.Headers
		if repProto.Body != nil {
			cr.protoBodyDecoded = repProto.BodyDecoded
		}

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, o))
	}
	return
}

var _ starlark.HasAttrs = (*cxResponseAfterResponse)(nil)

func (m *cxResponseAfterResponse) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable: %s", m.Type())
}
func (m *cxResponseAfterResponse) String() string       { return "response_after_response" }
func (m *cxResponseAfterResponse) Truth() starlark.Bool { return true }
func (m *cxResponseAfterResponse) Type() string         { return m.ty }

func (m *cxResponseAfterResponse) Freeze() {
	m.attrs.Freeze()
	if m.body != nil {
		m.body.Freeze()
	}
	if m.headers != nil {
		m.headers.Freeze()
	}
}

func (m *cxResponseAfterResponse) AttrNames() []string {
	if m.attrnames == nil {
		names := append(m.attrs.Keys(), "headers")
		if m.protoBodyDecoded != nil {
			names = append(names, "body")
		}
		sort.Strings(names)
		m.attrnames = names
	}
	return m.attrnames
}

func (m *cxResponseAfterResponse) Attr(name string) (starlark.Value, error) {
	switch {
	case name == "body" && m.protoBodyDecoded != nil:
		if m.body == nil {
			m.body = starlarkvalue.FromProtoValue(m.protoBodyDecoded)
			m.body.Freeze()
		}
		return m.body, nil

	case name == "headers":
		if m.headers == nil {
			var err error
			if m.headers, err = headerPairs(m.protoHeaders); err != nil {
				return nil, err
			}
			m.headers.Freeze()
		}
		return m.headers, nil

	default:
		if v := m.attrs[name]; v != nil {
			return v, nil
		}
		return nil, nil // no such method
	}
}
