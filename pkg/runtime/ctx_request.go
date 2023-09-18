package runtime

import (
	"fmt"
	"strings"

	"go.starlark.net/starlark"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
)

const (
	ctxHttpRequest  = "http_request"
	ctxHttpResponse = "http_response"
)

// ctxRequest_ represents input request data as a Starlark value for user assertions or mutation.
type ctxRequest_ struct {
	ty string

	method, url starlark.String
	headers     *ctxHeader
	body        starlark.Value
}

func newCtxRequest(input *fm.Srv_Call_Input) *ctxRequest_ {
	switch x := input.GetInput().(type) {

	case *fm.Srv_Call_Input_HttpRequest_:
		r := input.GetHttpRequest()
		var body starlark.Value
		if x := r.GetBody(); x != nil {
			body = starlarkvalue.FromProtoValue(x)
		}
		return &ctxRequest_{
			ty:      ctxHttpRequest,
			method:  starlark.String(r.GetMethod()),
			url:     starlark.String(r.GetUrl()),
			headers: newCtxHeader(r.GetHeaders()),
			//content: absent as encoding will only happen later
			body: body,
		}, nil

	default:
		panic(fmt.Errorf("unhandled output %T: %+v", x, input))
	}
}

func (cr *ctxRequest_) IntoProto(err error) *fm.Clt_CallRequestRaw {
	input:=

	var reason []string
	if err != nil {
		reason = strings.Split(err.Error(), "\n")
	}

	return &fm.Clt_CallRequestRaw{
		Input:  &fm.Clt_CallRequestRaw_Input{Input: input},
		Reason: reason,
	}
}

var _ starlark.HasAttrs = (*ctxRequest_)(nil)

func (cr *ctxRequest_) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: %s", cr.Type()) }
func (cr *ctxRequest_) String() string        { return "ctx_request" }
func (cr *ctxRequest_) Truth() starlark.Bool  { return true }
func (cr *ctxRequest_) Type() string          { return cr.ty }

func (cr *ctxRequest_) Freeze() {
	cr.body.Freeze()
	cr.headers.Freeze()
}

func (cr *ctxRequest_) AttrNames() []string {
	return []string{
		// Keep 'em sorted
		"body",
		"headers",
		"method",
		"url",
	}
}

func (cr *ctxRequest_) Attr(name string) (starlark.Value, error) {
	switch name {
	case "body":
		if cr.body == nil {
			return starlark.None, nil
		}
		return cr.body, nil
	case "headers":
		return cr.headers, nil
	case "method":
		return cr.method, nil
	case "url":
		return cr.url, nil
	default:
		return nil, nil // no such method
	}
}
