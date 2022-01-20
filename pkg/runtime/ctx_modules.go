package runtime

import (
	"errors"
	"fmt"
	"sort"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
	"github.com/gogo/protobuf/types"
	"go.starlark.net/starlark"
)

type (
	ctxctor2 func(*fm.Clt_CallResponseRaw_Output) ctxctor1
	ctxctor1 func(*starlark.Dict) *ctxModule
)

func ctxCurry(callInput *fm.Clt_CallRequestRaw_Input) ctxctor2 {
	request := inputAsValue(callInput)
	request.Freeze()
	return func(callOutput *fm.Clt_CallResponseRaw_Output) ctxctor1 {
		response := outputAsValue(callOutput)
		response.Freeze()
		return func(state *starlark.Dict) *ctxModule {
			// state is mutated through checks
			return &ctxModule{
				request:  request,
				response: response,
				state:    state,
			}
		}
	}
}

// Modified https://github.com/google/starlark-go/blob/ebe61bd709bf/starlarkstruct/module.go
type ctxModule struct {
	accessedState bool
	request       *ctxRequest
	response      *ctxResponse
	state         *starlark.Dict
	//TODO: specs             starlark.Value
	//TODO: CLI filter `--only="starlark.expr(ctx.specs)"`
	//TODO: ctx.specs stops being accessible on first ctx.state access
}

// TODO? easy access to generated parameters. For instance:
// post_id = ctx.request["parameters"]["path"]["{id}"] (note decoded int)

var _ starlark.HasAttrs = (*ctxModule)(nil)

func (m *ctxModule) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *ctxModule) String() string        { return "ctx" }
func (m *ctxModule) Truth() starlark.Bool  { return true }
func (m *ctxModule) Type() string          { return "ctx" }
func (m *ctxModule) AttrNames() []string   { return []string{"request", "response", "state"} }

func (m *ctxModule) Freeze() {
	m.request.Freeze()
	m.response.Freeze()
	m.state.Freeze()
}

func (m *ctxModule) Attr(name string) (starlark.Value, error) {
	switch name {
	case "request":
		if m.accessedState {
			return nil, errors.New("cannot access ctx.request after accessing ctx.state")
		}
		return m.request, nil
	case "response":
		if m.accessedState {
			return nil, errors.New("cannot access ctx.response after accessing ctx.state")
		}
		return m.response, nil
	case "state":
		m.accessedState = true
		return m.state, nil
	default:
		return nil, nil // no such method
	}
}

// ctxRequest represents request data as a Starlark value for user assertions.
type ctxRequest struct {
	attrs     starlark.StringDict
	attrnames []string

	protoBodyDecoded *types.Value
	body             starlark.Value

	protoHeaders []*fm.HeaderPair
	headers      starlark.Value
}

var _ starlark.HasAttrs = (*ctxRequest)(nil)

func (m *ctxRequest) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *ctxRequest) String() string        { return "ctx_request" }
func (m *ctxRequest) Truth() starlark.Bool  { return true }
func (m *ctxRequest) Type() string          { return "ctx_request" }

func (m *ctxRequest) Freeze() {
	m.attrs.Freeze()
	// NOTE: m.body.Freeze() in Attr()
	// NOTE: m.headers.Freeze() in Attr()
}

func (m *ctxRequest) AttrNames() []string {
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

func (m *ctxRequest) Attr(name string) (starlark.Value, error) {
	switch {
	case m.protoBodyDecoded != nil && name == "body":
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
		return m.attrs[name], nil
	}
}

func headerPairs(protoHeaders []*fm.HeaderPair) (starlark.Value, error) {
	d := starlark.NewDict(len(protoHeaders))

	for _, kvs := range protoHeaders {
		values := kvs.GetValues()
		vs := make([]starlark.Value, 0, len(values))
		for _, value := range values {
			vs = append(vs, starlark.String(value))
		}
		if err := d.SetKey(starlark.String(kvs.GetKey()), starlark.NewList(vs)); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func inputAsValue(i *fm.Clt_CallRequestRaw_Input) (cr *ctxRequest) {
	cr = &ctxRequest{
		attrs: make(starlark.StringDict, 3),
	}
	switch x := i.GetInput().(type) {

	case *fm.Clt_CallRequestRaw_Input_HttpRequest_:
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

// ctxResponse represents response data as a Starlark value for user assertions.
type ctxResponse struct {
	attrs     starlark.StringDict
	attrnames []string

	protoBodyDecoded *types.Value
	body             starlark.Value

	protoHeaders []*fm.HeaderPair
	headers      starlark.Value
}

var _ starlark.HasAttrs = (*ctxResponse)(nil)

func (m *ctxResponse) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *ctxResponse) String() string        { return "ctx_response" }
func (m *ctxResponse) Truth() starlark.Bool  { return true }
func (m *ctxResponse) Type() string          { return "ctx_response" }

func (m *ctxResponse) Freeze() {
	m.attrs.Freeze()
	// NOTE: m.body.Freeze() in Attr()
	// NOTE: m.headers.Freeze() in Attr()
}

func (m *ctxResponse) AttrNames() []string {
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

func (m *ctxResponse) Attr(name string) (starlark.Value, error) {
	switch {
	case m.protoBodyDecoded != nil && name == "body":
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
		return m.attrs[name], nil
	}
}

func outputAsValue(o *fm.Clt_CallResponseRaw_Output) (cr *ctxResponse) {
	cr = &ctxResponse{
		attrs: make(starlark.StringDict, 5),
	}
	switch x := o.GetOutput().(type) {

	case *fm.Clt_CallResponseRaw_Output_HttpResponse_:
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
