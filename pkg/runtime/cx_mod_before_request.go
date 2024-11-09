package runtime

import (
	"fmt"

	"go.starlark.net/starlark"
)

func newCxModBeforeRequest(req *cxBeforeRequest) *cxModBeforeRequest {
	return &cxModBeforeRequest{
		request: req,
	}
}

// cxModBeforeRequest is the `ctx` starlark value accessible before executing a call
type cxModBeforeRequest struct {
	request *cxBeforeRequest
	// No response: this lives only before the request is attempted
	// No state: disallowed for now
	//TODO: specs
}

var _ starlark.HasAttrs = (*cxModBeforeRequest)(nil)

func (m *cxModBeforeRequest) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", m.Type()) }
func (m *cxModBeforeRequest) String() string        { return "ctx" }
func (m *cxModBeforeRequest) Truth() starlark.Bool  { return true }
func (m *cxModBeforeRequest) Type() string          { return "ctx_before_request" }
func (m *cxModBeforeRequest) AttrNames() []string   { return []string{"request"} }
func (m *cxModBeforeRequest) Freeze()               { m.request.Freeze() }

func (m *cxModBeforeRequest) Attr(name string) (starlark.Value, error) {
	switch name {
	case "request":
		return m.request, nil
	default:
		return nil, nil // no such method
	}
}
