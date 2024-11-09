package runtime

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
)

// cxRequestAfterResponse is the `ctx.request` starlark value accessible after executing a call
type cxRequestAfterResponse struct {
	ty string

	attrs     starlark.StringDict
	attrnames []string

	protoBodyDecoded *structpb.Value
	body             starlark.Value

	protoHeaders []*fm.HeaderPair
	headers      starlark.Value //FIXME: cxHeaders + Freeze
}

func newCxRequestAfterResponse(i *fm.Clt_CallRequestRaw_Input) (cr *cxRequestAfterResponse) {
	cr = &cxRequestAfterResponse{
		attrs: make(starlark.StringDict, 3),
	}
	switch x := i.GetInput().(type) {

	case *fm.Clt_CallRequestRaw_Input_HttpRequest_:
		cr.ty = cxRequestHttp

		reqProto := i.GetHttpRequest()
		cr.attrs["method"] = starlark.String(reqProto.Method)
		cr.attrs["url"] = starlark.String(reqProto.Url)
		cr.attrs["content"] = starlark.String(reqProto.Body)
		cr.protoHeaders = reqProto.Headers
		if reqProto.Body != nil {
			cr.protoBodyDecoded = reqProto.BodyDecoded
		}

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, i))
	}
	return
}

var _ starlark.HasAttrs = (*cxRequestAfterResponse)(nil)

func (m *cxRequestAfterResponse) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable: %s", m.Type())
}
func (m *cxRequestAfterResponse) String() string       { return "request_after_response" }
func (m *cxRequestAfterResponse) Truth() starlark.Bool { return true }
func (m *cxRequestAfterResponse) Type() string         { return m.ty }

func (m *cxRequestAfterResponse) Freeze() {
	m.attrs.Freeze()
	if m.body != nil {
		m.body.Freeze()
	}
	if m.headers != nil {
		m.headers.Freeze()
	}
}

func (m *cxRequestAfterResponse) AttrNames() []string {
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

func (m *cxRequestAfterResponse) Attr(name string) (starlark.Value, error) {
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
