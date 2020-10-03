package runtime

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestLangParents(t *testing.T) {
	initExec() // Sets up Starlark Universe
	predeclared := starlark.Universe

	thread := &starlark.Thread{
		Name: t.Name(),
		Print: func(_ *starlark.Thread, msg string) {
			t.Logf("--> %s", msg)
		},
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, errors.New("load() unsupported")
		},
	}

	code := `
def f(r):
	return r["json"][0]["key"]

x = {
	"request": {
		"method": "GET",
		"url": "http://some/path",
		"headers": [],
	},
	"status_code": 200,
	"reason": None,
	"elapsed_ns": 674154,
	"json": [
		{"key": [{"1":2}]},
		{"key2": [{"trois":4}]},
	],
}

if True:
	print(f(x))
	for y in x["json"]:
		for z in y:
			print([w for w in z])
`

	globals, err := starlark.ExecFile(thread, t.Name()+".star", code, predeclared)
	require.NoError(t, err)
	require.Len(t, globals, 4)

	t.Logf(">>> datapaths = %+v", datapaths)
	require.NotEmpty(t, datapaths.current)
}
