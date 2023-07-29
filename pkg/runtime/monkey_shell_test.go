package runtime

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FuzzyMonkeyCo/monkey/pkg/cwid"
	"github.com/FuzzyMonkeyCo/monkey/pkg/progresser/ci"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
)

// generic over resetters

func TestResetterPositionalArgsAreForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell("hi", name = "bla")
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell(...) does not take positional arguments, only named ones`[1:])
	require.Nil(t, rt)
}

func TestResetterNamesMustBeLegal(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blip blop",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: only characters from `[1:]+tags.Alphabet+` should be in "blip blop"`)
	require.Nil(t, rt)
}

func TestResetterNamesMustBeUnique(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blip",
    provides = ["some_model"],
)
monkey.shell(
    name = "blip",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:5:13: in <toplevel>
Error in shell: a resetter named blip already exists`[1:])
	require.Nil(t, rt)

	rt, err = newFakeMonkey(t, `
monkey.shell(
    name = "blip",
    provides = ["some_model"],
)
monkey.shell(
    name = "blop",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.resetters, 2)
}

// name

func TestShellNameIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    reset = "true",
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: missing argument for name`[1:])
	require.Nil(t, rt)
}

func TestShellNameTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = 42.1337,
    provides = ["some_model"],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "name": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwargs

func TestShellAdditionalKwardsForbidden(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    wef = "bla",
)
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: unexpected keyword argument "wef"`[1:])
	require.Nil(t, rt)
}

// kwarg: provides

func TestShellProvidesIsRequired(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: missing argument for provides`[1:])
	require.Nil(t, rt)
}

func TestShellProvidesTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = [42.1337],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "provides": got float, want string`[1:])
	require.Nil(t, rt)
}

func TestShellProvidesNonEmpty(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = [],
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "provides": must not be empty`[1:])
	require.Nil(t, rt)
}

// kwarg: start

func TestShellStartTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    start = 42.1337,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "start": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwarg: reset

func TestShellResetTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    reset = 42.1337,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "reset": got float, want string`[1:])
	require.Nil(t, rt)
}

// kwarg: stop

func TestShellStopTyping(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    stop = 42.1337,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in shell: shell: for parameter "stop": got float, want string`[1:])
	require.Nil(t, rt)
}

// execution

func TestShellResets(t *testing.T) {
	type testcase struct {
		code     string
		expected error
	}

	as := func(ss []string) [][]byte {
		xs := make([][]byte, 0, len(ss))
		for _, s := range ss {
			xs = append(xs, []byte(s))
		}
		return xs
	}

	repeated := strings.Repeat(":\n", 42)

	for _, tst := range []testcase{
		{":", nil},
		{"true", nil},
		{"sleep .1", nil},
		{"echo Hello; echo hi >&2; false", resetter.NewError(as([]string{"echo Hello!", "echo hi >&2", "false"}))},
		{"false", resetter.NewError(as([]string{"false"}))},
		{repeated + "false", resetter.NewError(as(append(strings.Split(repeated, "\n"), "false")))},
	} {
		t.Run(tst.code, func(t *testing.T) {
			rt, err := newFakeMonkey(t, fmt.Sprintf(`
monkey.shell(
    name = "blop",
    provides = ["some_model"],
    reset = """%s""",
)
`[1:]+someOpenAPI3Model, tst.code))
			require.NoError(t, err)
			require.Len(t, rt.resetters, 1)
			require.Equal(t, []string{"some_model"}, rt.resetters["blop"].Provides())
			require.Contains(t, someOpenAPI3Model, `"some_model"`)
			require.Len(t, rt.selectedResetters, 0)

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// In the order written in main.go
			err = rt.Lint(ctx, false)
			require.NoError(t, err)
			err = rt.FilterEndpoints(nil)
			require.NoError(t, err)

			//todo: move some code to lang_test.go + share with fuzz.go / exec.go

			require.Len(t, rt.selectedResetters, 0)
			var selected []string
			err = rt.forEachSelectedResetter(ctx, func(name string, rsttr resetter.Interface) error {
				selected = append(selected, name)
				return nil
			})
			require.NoError(t, err)
			require.Len(t, rt.selectedResetters, 1)
			require.Equal(t, []string{"blop"}, selected)

			rt.progress = &ci.Progresser{}
			rt.client = &fakeClient{}

			err = cwid.MakePwdID(rt.binTitle, ".", 0)
			require.NoError(t, err)
			require.NotEmpty(t, cwid.Prefixed())

			scriptErr, err := rt.reset(ctx)
			require.NoError(t, err)
			require.Len(t, rt.selectedResetters, 1)
			if tst.expected != nil {
				require.IsType(t, resetter.NewError(nil), scriptErr)
				e := tst.code
				e = strings.ReplaceAll(e, "\n", ";")
				e = strings.ReplaceAll(e, "; ", ";")
				e = strings.ReplaceAll(e, " >&2", "")
				require.EqualError(t, scriptErr, e)
			} else {
				require.NoError(t, scriptErr)
			}
		})
	}
}
