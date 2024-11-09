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
}

func newCxRequestAfterResponse(i *fm.Clt_CallRequestRaw_Input) (cr *cxRequestAfterResponse) {
	switch x := i.GetInput().(type) {

	case *fm.Clt_CallRequestRaw_Input_HttpRequest_:
		cr = &cxRequestAfterResponse{
			ty:    cxRequestHttp,
			attrs: make(starlark.StringDict, 4),
		}

		reqProto := i.GetHttpRequest()
		cr.attrs["method"] = starlark.String(reqProto.Method)
		cr.attrs["url"] = starlark.String(reqProto.Url)
		cr.attrs["content"] = starlark.String(reqProto.Body)
		headers := newcxHead(reqProto.Headers)
		headers.Freeze()
		cr.attrs["headers"] = headers
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
}

func (m *cxRequestAfterResponse) AttrNames() []string {
	if m.attrnames == nil {
		names := m.attrs.Keys()
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
	default:
		return m.attrs[name], nil
	}
}
