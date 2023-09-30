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
	ctxHttpRequest  = "http_request"
	ctxHttpResponse = "http_response"
)

// ctxRequest_ represents input request data as a Starlark value for user assertions or mutation.
type ctxRequest_ struct {
	ty string

	method, url starlark.String
	headers     *ctxHeader
	body        *structpb.Value //  starlark.Value FIXME
	// FIXME: try starlark script that edits .body (as num and as dict/list, and as set)
}

func newCtxRequest(input *fm.Srv_Call_Input) *ctxRequest_ {
	switch x := input.GetInput().(type) {

	case *fm.Srv_Call_Input_HttpRequest_:
		r := input.GetHttpRequest()
		return &ctxRequest_{
			ty:      ctxHttpRequest,
			method:  starlark.String(r.GetMethod()),
			url:     starlark.String(r.GetUrl()),
			headers: newCtxHeader(r.GetHeaders()),
			//content: absent as encoding will only happen later
			body: r.GetBody(),
		}

	default:
		panic(fmt.Errorf("unhandled input %T: %+v", x, input))
	}
}

func (cr *ctxRequest_) IntoProto(err error) *fm.Clt_CallRequestRaw {
	var reason []string
	if len(reason) == 0 && err != nil {
		reason = strings.Split(err.Error(), "\n")
	}

	input := func() *fm.Clt_CallRequestRaw_Input_HttpRequest_ {
		switch cr.ty {
		case "http_request":
			var body []byte
			if body, err = protojson.Marshal(cr.body); err != nil {
				log.Println("[ERR]", err)
				// return after sending msg
				if len(reason) == 0 && err != nil {
					reason = strings.Split(err.Error(), "\n")
				}
			}
			return &fm.Clt_CallRequestRaw_Input_HttpRequest_{
				HttpRequest: &fm.Clt_CallRequestRaw_Input_HttpRequest{
					Method:      string(cr.method),
					Url:         string(cr.url),
					Headers:     cr.headers.IntoProto(),
					Body:        body,
					BodyDecoded: cr.body,
				}}

		default:
			panic(fmt.Errorf("unhandled input %s: %+v", cr.ty, cr))

		}
	}()

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
	// cr.body.Freeze() FIXME
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
		return starlarkvalue.FromProtoValue(cr.body), nil // FIXME: readonly
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
