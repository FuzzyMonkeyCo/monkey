package runtime

import (
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
)

const someOpenAPI3Model = `
monkey.openapi3(
    name = "some_model",
    file = "pkg/modeler/openapiv3/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml",
    host = "https://jsonplaceholder.typicode.com",
)
`

func newFakeMonkey(code string) (*Runtime, error) {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	initExec()

	localCfgData = []byte(code) // Mocks fuzzymonkey.star contents

	return NewMonkey("monkeh", nil)
}

func (rt *Runtime) runFakeUserCheck(t *testing.T, chkname string) *fm.Clt_CallVerifProgress {
	chk, ok := rt.checks[chkname]
	require.True(t, ok)

	tagsFilter, err := tags.NewFilter(false, false, nil, nil)
	require.NoError(t, err)

	var ctxer1 ctxctor1
	switch {
	case strings.HasPrefix(chkname, "ctx_request_body_frozen"): // NOTE: request does not match model
		reqbody := []byte(`{"error": {"msg":"not found", "id":0, "category":"albums"}}`)
		var reqdecoded types.Value
		err = jsonpb.UnmarshalString(string(reqbody), &reqdecoded)
		require.NoError(t, err)
		ctxer2 := ctxCurry(&fm.Clt_CallRequestRaw_Input{
			Input: &fm.Clt_CallRequestRaw_Input_HttpRequest_{
				HttpRequest: &fm.Clt_CallRequestRaw_Input_HttpRequest{
					Method: "POST",
					Url:    "https://jsonplaceholder.typicode.com/albums",
					Headers: []*fm.HeaderPair{
						{Key: "Accept", Values: []string{"application/json"}},
					},
					Body:        reqbody,
					BodyDecoded: &reqdecoded,
				},
			},
		})

		ctxer1 = ctxer2(&fm.Clt_CallResponseRaw_Output{
			Output: &fm.Clt_CallResponseRaw_Output_HttpResponse_{
				HttpResponse: &fm.Clt_CallResponseRaw_Output_HttpResponse{
					StatusCode: 404,
					Reason:     "404 Not Found",
					Headers: []*fm.HeaderPair{
						{Key: "Content-Length", Values: []string{"0"}},
					},
					ElapsedNs: 37 * 1000 * 1000,
				},
			},
		})

	default:
		ctxer2 := ctxCurry(&fm.Clt_CallRequestRaw_Input{
			Input: &fm.Clt_CallRequestRaw_Input_HttpRequest_{
				HttpRequest: &fm.Clt_CallRequestRaw_Input_HttpRequest{
					Method: "GET",
					Url:    "https://jsonplaceholder.typicode.com/albums/0",
					Headers: []*fm.HeaderPair{
						{Key: "Accept", Values: []string{"application/json"}},
					},
				},
			},
		})

		repbody := []byte(`{"error": {"msg":"not found", "id":0, "category":"albums"}}`)
		var repdecoded types.Value
		err = jsonpb.UnmarshalString(string(repbody), &repdecoded)
		require.NoError(t, err)
		ctxer1 = ctxer2(&fm.Clt_CallResponseRaw_Output{
			Output: &fm.Clt_CallResponseRaw_Output_HttpResponse_{
				HttpResponse: &fm.Clt_CallResponseRaw_Output_HttpResponse{
					StatusCode: 404,
					Reason:     "404 Not Found",
					Headers: []*fm.HeaderPair{
						{Key: "Access-Control-Allow-Credentials", Values: []string{"true"}},
						{Key: "Age", Values: []string{"0"}},
						{Key: "Cache-Control", Values: []string{"max-age=43200"}},
						{Key: "Cf-Cache-Status", Values: []string{"HIT"}},
						{Key: "Cf-Ray", Values: []string{"6220ea3cdda30686-LHR"}},
						{Key: "Content-Length", Values: []string{strconv.Itoa(len(repbody))}},
						{Key: "Content-Type", Values: []string{"application/json; charset=utf-8"}},
						{Key: "Date", Values: []string{"Mon, 15 Feb 2021 17:58:05 GMT"}},
						{Key: "Etag", Values: []string{`W/"2-vyGp6PvFo4RvsFtPoIWeCReyIC8"`}},
						{Key: "Expect-Ct", Values: []string{`max-age=604800, report-uri="https://report-uri.cloudflare.com/cdn-cgi/beacon/expect-ct"`}},
						{Key: "Expires", Values: []string{"-1"}},
					},
					Body:        repbody,
					BodyDecoded: &repdecoded,
					ElapsedNs:   37 * 1000 * 1000,
				},
			},
		})
	}

	print := func(msg string) { t.Logf("PRINT%s", msg) }
	return rt.runUserCheckWrapper(chkname, chk, print, tagsFilter, ctxer1, 1337)
}
