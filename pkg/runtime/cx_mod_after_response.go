package runtime

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

// TODO? easy access to generated parameters. For instance:
// post_id = ctx.request["parameters"]["path"]["{id}"] (note decoded int)

// cxModAfterResponse is the `ctx` starlark value accessible after executing a call
type cxModAfterResponse struct {
	accessedState bool
	request       *cxRequestAfterResponse
	response      *cxResponseAfterResponse
	state         *starlark.Dict
	//TODO: specs             starlark.Value => provide models as JSON for now until we find a suitable Python-ish API
	//TODO: CLI filter `--only="starlark.expr(ctx.specs)"`
	//TODO: ctx.specs stops being accessible on first ctx.state access
}

type (
	ctxctor2 func(*fm.Clt_CallResponseRaw_Output) ctxctor1
	ctxctor1 func(*starlark.Dict) *cxModAfterResponse
)

func ctxCurry(callInput *fm.Clt_CallRequestRaw_Input) ctxctor2 {
	request := newCxRequestAfterResponse(callInput)
	request.Freeze()
	return func(callOutput *fm.Clt_CallResponseRaw_Output) ctxctor1 {
		response := newCxResponseAfterResponse(callOutput)
		response.Freeze()
		return func(state *starlark.Dict) *cxModAfterResponse {
			// state is mutated through checks
			return &cxModAfterResponse{
				request:  request,
				response: response,
				state:    state,
			}
		}
	}
}

var _ starlark.HasAttrs = (*cxModAfterResponse)(nil)

func (m *cxModAfterResponse) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *cxModAfterResponse) String() string        { return "ctx_after_response" }
func (m *cxModAfterResponse) Truth() starlark.Bool  { return true }
func (m *cxModAfterResponse) Type() string          { return "ctx" }
func (m *cxModAfterResponse) AttrNames() []string   { return []string{"request", "response", "state"} }

func (m *cxModAfterResponse) Freeze() {
	m.request.Freeze()
	m.response.Freeze()
	m.state.Freeze()
}

func (m *cxModAfterResponse) Attr(name string) (starlark.Value, error) {
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
