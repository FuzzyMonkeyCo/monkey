package runtime

import (
	"errors"
	"fmt"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
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
	attrs starlark.StringDict
}

var _ starlark.HasAttrs = (*ctxRequest)(nil)

func (m *ctxRequest) Hash() (uint32, error)                    { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *ctxRequest) String() string                           { return "ctx_request" }
func (m *ctxRequest) Truth() starlark.Bool                     { return true }
func (m *ctxRequest) Type() string                             { return "ctx_request" }
func (m *ctxRequest) AttrNames() []string                      { return m.attrs.Keys() }
func (m *ctxRequest) Freeze()                                  { m.attrs.Freeze() }
func (m *ctxRequest) Attr(name string) (starlark.Value, error) { return m.attrs[name], nil }

// ctxResponse represents response data as a Starlark value for user assertions.
type ctxResponse struct {
	attrs starlark.StringDict
}

var _ starlark.HasAttrs = (*ctxResponse)(nil)

func (m *ctxResponse) Hash() (uint32, error)                    { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *ctxResponse) String() string                           { return "ctx_response" }
func (m *ctxResponse) Truth() starlark.Bool                     { return true }
func (m *ctxResponse) Type() string                             { return "ctx_response" }
func (m *ctxResponse) AttrNames() []string                      { return m.attrs.Keys() }
func (m *ctxResponse) Freeze()                                  { m.attrs.Freeze() }
func (m *ctxResponse) Attr(name string) (starlark.Value, error) { return m.attrs[name], nil }
