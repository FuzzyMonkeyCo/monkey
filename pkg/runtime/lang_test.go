package runtime

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/ci"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const someOpenAPI3Model = `
monkey.openapi3(
    name = "some_model",
    file = "pkg/modeler/openapiv3/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml",
    host = "https://jsonplaceholder.typicode.com",
)
`

var starFile = "fuzzymonkey.star"

func newFakeMonkey(t *testing.T, code string) (*Runtime, error) {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	initExec()
	starlark.Universe["monkeh_sleep"] = starlark.NewBuiltin("monkeh_sleep", monkehSleep)

	starfileData = []byte(code) // Mocks fuzzymonkey.star contents

	if t != nil {
		ensureFormattedAs(t, code, code)
	}

	return NewMonkey("monkeh", starFile, nil)
}

func monkehSleep(th *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var ms starlark.Int
	if err := starlark.UnpackPositionalArgs(b.Name(), args, kwargs, 1, &ms); err != nil {
		return nil, err
	}
	d, ok := ms.Uint64()
	if !ok {
		return nil, fmt.Errorf("not milliseconds: %s", ms.String())
	}
	ctx := th.Local("ctx").(context.Context)
	select {
	case <-ctx.Done():
	case <-time.After(time.Duration(d) * time.Millisecond):
	}
	return starlark.None, nil
}

func checkPrint(t *testing.T) func(string) {
	return func(msg string) { t.Logf("PRINT%s", msg) }
}

func (rt *Runtime) runFakeUserCheck(t *testing.T, chkname string) *fm.Clt_CallVerifProgress {
	chk, ok := rt.checks[chkname]
	require.True(t, ok)

	tagsFilter, err := tags.NewFilter(false, false, nil, nil)
	require.NoError(t, err)

	ctxer1 := ctxer1Maker(t, chkname)

	ctx := context.Background()
	var th *starlark.Thread
	for _, th = range rt.makeThreads(ctx) {
		break
	}

	return rt.runUserCheckWrapper(chkname, th, chk, checkPrint(t), tagsFilter, ctxer1, 1337)
}

func (rt *Runtime) fakeUserChecks(ctx context.Context, t *testing.T) (bool, error) {
	tagsFilter, err := tags.NewFilter(false, false, nil, nil)
	require.NoError(t, err)

	ctxer1 := ctxer1Maker(t, "")

	rt.progress = &ci.Progresser{}
	rt.client = &fakeClient{}

	return rt.userChecks(ctx, checkPrint(t), tagsFilter, ctxer1, 1337, time.Second)
}

type fakeClient struct{}

func (fc *fakeClient) Close()                                            {}
func (fc *fakeClient) Send(ctx context.Context, msg *fm.Clt) (err error) { return }
func (fc *fakeClient) Receive(ctx context.Context) (msg *fm.Srv, err error) {
	msg = &fm.Srv{FuzzingProgress: &fm.Srv_FuzzingProgress{}}
	return
}

func ctxer1Maker(t *testing.T, chkname string) ctxctor1 {
	switch {
	case strings.HasPrefix(chkname, "ctx_request_body_frozen"): // NOTE: request does not match model
		reqbody := []byte(`{"error": {"msg":"not found", "id":0, "category":"albums"}}`)
		var reqdecoded structpb.Value
		err := protojson.Unmarshal(reqbody, &reqdecoded)
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

		return ctxer2(&fm.Clt_CallResponseRaw_Output{
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
		var repdecoded structpb.Value
		err := protojson.Unmarshal(repbody, &repdecoded)
		require.NoError(t, err)
		return ctxer2(&fm.Clt_CallResponseRaw_Output{
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
}
