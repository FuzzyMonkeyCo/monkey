package runtime

import (
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"
)

const simplestPrelude = `
OpenAPIv3(
    name = "some_model",
    file = "pkg/modeler/openapiv3/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml",
    host = "https://jsonplaceholder.typicode.com",
)
`

func newFakeMonkey(code string) (*Runtime, error) {
	initExec()

	localCfgData = []byte(code) // Mocks fuzzymonkey.star contents

	return NewMonkey("monkeh", nil)
}

func (rt *Runtime) runFakeUserCheck(t *testing.T, chkname string) *fm.Clt_CallVerifProgress {
	chk, ok := rt.checks[chkname]
	require.True(t, ok)

	tagsFilter, err := tags.NewFilter(false, false, nil, nil)
	require.NoError(t, err)

	ctxer2 := ctxCurry(&fm.Clt_CallRequestRaw_Input{
		Input: &fm.Clt_CallRequestRaw_Input_HttpRequest_{
			HttpRequest: &fm.Clt_CallRequestRaw_Input_HttpRequest{
				Method: "GET",
				Url:    "https://jsonplaceholder.typicode.com/todos/0",
				Headers: map[string]*fm.Clt_CallRequestRaw_Input_HttpRequest_HeaderValues{
					"Accept": {Values: []string{"application/json"}},
				},
			},
		},
	})
	ctxer1 := ctxer2(&fm.Clt_CallResponseRaw_Output{
		Output: &fm.Clt_CallResponseRaw_Output_HttpResponse_{
			HttpResponse: &fm.Clt_CallResponseRaw_Output_HttpResponse{
				StatusCode: 404,
				Reason:     "404 Not Found",
				Headers: map[string]*fm.Clt_CallResponseRaw_Output_HttpResponse_HeaderValues{
					"Access-Control-Allow-Credentials": {Values: []string{"true"}},
					"Age":                              {Values: []string{"0"}},
					"Cache-Control":                    {Values: []string{"max-age=43200"}},
					"Cf-Cache-Status":                  {Values: []string{"HIT"}},
					"Cf-Ray":                           {Values: []string{"6220ea3cdda30686-LHR"}},
					"Cf-Request-Id":                    {Values: []string{"084870ba0b00000686761ff000000001"}},
					"Connection":                       {Values: []string{"keep-alive"}},
					"Content-Length":                   {Values: []string{"2"}},
					"Content-Type":                     {Values: []string{"application/json; charset=utf-8"}},
					"Date":                             {Values: []string{"Mon, 15 Feb 2021 17:58:05 GMT"}},
					"Etag":                             {Values: []string{`W/"2-vyGp6PvFo4RvsFtPoIWeCReyIC8"`}},
					"Expect-Ct":                        {Values: []string{`max-age=604800, report-uri="https://report-uri.cloudflare.com/cdn-cgi/beacon/expect-ct"`}},
					"Expires":                          {Values: []string{"-1"}},
				},
				Body:        []byte("{}"),
				BodyDecoded: &types.Value{Kind: &types.Value_StructValue{StructValue: &types.Struct{}}},
				ElapsedNs:   776674,
			},
		},
	})

	print := func(msg string) { t.Logf("PRINT%s", msg) }
	return rt.runUserCheckWrapper(chkname, chk, print, tagsFilter, ctxer1, 1337)
}
