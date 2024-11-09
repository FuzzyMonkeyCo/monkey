package runtime

import (
	"fmt"
	"log"
	"strings"

	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/starlarkvalue"
)

const (
	cxRequestHttp = "http_request"
)

// cxRequestBeforeRequest is the `ctx.request` starlark value accessible before executing a call
type cxRequestBeforeRequest struct {
	ty string

	method, url starlark.String
	headers     *cxHead
	body        *structpb.Value //FIXME: starlark.Value + test that edits .body (as num and as dict/list, and as set)
}

func newCxRequestBeforeRequest(input *fm.Srv_Call_Input) *cxRequestBeforeRequest {
	switch x := input.GetInput().(type) {

	case *fm.Srv_Call_Input_HttpRequest_:
		r := input.GetHttpRequest()
		return &cxRequestBeforeRequest{
			ty:      cxRequestHttp,
			method:  starlark.String(r.GetMethod()),
			url:     starlark.String(r.GetUrl()),
			headers: newcxHead(r.GetHeaders()),
			//content: absent as encoding will only happen later
			body: r.GetBody(),
		}

	default:
		panic(fmt.Errorf("unhandled input %T: %+v", x, input))
	}
}

func (cr *cxRequestBeforeRequest) IntoProto(err error) *fm.Clt_CallRequestRaw {
	var reason []string
	if err != nil {
		reason = strings.Split(err.Error(), "\n")
	}

	input := func() *fm.Clt_CallRequestRaw_Input {
		switch cr.ty {
		case cxRequestHttp:
			var body []byte
			if cr.body != nil {
				if body, err = protojson.Marshal(cr.body); err != nil {
					log.Println("[ERR]", err)
					// return after sending msg
					if len(reason) == 0 && err != nil {
						reason = strings.Split(err.Error(), "\n")
					}
				}
			}
			//TODO: impl ToProtoValue
			return &fm.Clt_CallRequestRaw_Input{
				Input: &fm.Clt_CallRequestRaw_Input_HttpRequest_{
					HttpRequest: &fm.Clt_CallRequestRaw_Input_HttpRequest{
						Method:      string(cr.method),
						Url:         string(cr.url),
						Headers:     cr.headers.IntoProto(),
						Body:        body,
						BodyDecoded: cr.body,
					}}}

		default:
			panic(fmt.Errorf("unhandled input %s: %+v", cr.ty, cr))

		}
	}()

	return &fm.Clt_CallRequestRaw{
		Input:  input,
		Reason: reason,
	}
}

var _ starlark.HasAttrs = (*cxRequestBeforeRequest)(nil)

func (cr *cxRequestBeforeRequest) Hash() (uint32, error) {
	return 0, fmt.Errorf("unhashable: %s", cr.Type())
}
func (cr *cxRequestBeforeRequest) String() string       { return "request_before_request" }
func (cr *cxRequestBeforeRequest) Truth() starlark.Bool { return true }
func (cr *cxRequestBeforeRequest) Type() string         { return cr.ty }

func (cr *cxRequestBeforeRequest) Freeze() {
	// cr.body.Freeze() FIXME
	cr.headers.Freeze()
}

func (cr *cxRequestBeforeRequest) AttrNames() []string {
	return []string{ // Keep 'em sorted
		"body",
		"headers",
		"method",
		"url",
	}
}

func (cr *cxRequestBeforeRequest) Attr(name string) (starlark.Value, error) {
	switch name {
	case "body":
		var body starlark.Value = starlark.None
		if cr.body != nil {
			body = starlarkvalue.FromProtoValue(cr.body)
		}
		body.Freeze()
		// TODO: call ProtoCompatible here
		return body, nil
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
