package starlarktruth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func helper(t *testing.T, program string) (starlark.StringDict, error) {
	predeclared := starlark.StringDict{}
	NewModule(predeclared)
	thread := &starlark.Thread{
		Name: t.Name(),
		Print: func(_ *starlark.Thread, msg string) {
			t.Logf("--> %s", msg)
		},
		Load: func(_ *starlark.Thread, module string) (starlark.StringDict, error) {
			return nil, errors.New("load() unsupported")
		},
	}
	return starlark.ExecFile(thread, t.Name()+".star", program, predeclared)
	// if err != nil {
	// 	if evalErr, ok := err.(*starlark.EvalError); ok {
	// 		log.Fatal(evalErr.Backtrace())
	// 	}
	// 	log.Fatal(err)
	// }
	// require.NoError(t,err)

	// for _, name := range globals.Keys() {
	// 	v := globals[name]
	// 	t.Logf("%s (%s) = %s\n", name, v.Type(), v.String())
	// }
	// require.Len(t, globals, 0)
}

func TestComparables(t *testing.T) {
	for code, expectedErr := range map[string]error{
		`AssertThat(5).isAtMost(5)`: nil,
		`AssertThat(5).isAtMost(8)`: nil,
		`AssertThat(5).isAtMost(3)`: NewTruthAssertion("Not true that <5> is at most <3>."),
	} {
		t.Run(code, func(t *testing.T) {
			globals, err := helper(t, code)
			require.Empty(t, globals)
			if expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.EqualError(t, err, expectedErr.Error())
				require.True(t, errors.As(err, &expectedErr))
				require.IsType(t, expectedErr, err)
			}
		})
	}

}
