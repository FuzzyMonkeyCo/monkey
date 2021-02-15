package runtime

import (
	"errors"
	"fmt"

	"go.starlark.net/starlark"
)

// TODO? easy access to generated parameters. For instance:
// post_id = ctx.request["parameters"]["path"]["{id}"] (note decoded int)

func newCtx(state, request, response starlark.Value) starlark.Value {
	request.Freeze()
	response.Freeze()
	// state is mutated through checks
	return &ctxModule{
		request:  request,
		response: response,
		state:    state,
	}
}

// Modified https://github.com/google/starlark-go/blob/ebe61bd709bf/starlarkstruct/module.go
type ctxModule struct {
	accessedState     bool
	request, response starlark.Value
	state             starlark.Value
	// specs             starlark.Value
	//TODO: CLI filter `--only="starlark.expr(ctx.specs)"`
	//TODO: ctx.specs stops being accessible on first ctx.state access
}

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
