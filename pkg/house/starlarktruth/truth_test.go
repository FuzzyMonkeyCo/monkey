package starlarktruth

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

var predeclared = starlark.StringDict{
	"AssertThat": starlark.NewBuiltin("AssertTthat", AssertThat),
}

func helper(t *testing.T, program string) (starlark.StringDict, error) {
	thread := &starlark.Thread{
		Name:  t.Name(),
		Print: func(_ *starlark.Thread, msg string) { t.Logf("--> %s", msg) },
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
	for _, data := range []string{
		`AssertThat(5).isAtMost(5)`,
		`AssertThat(5).isAtMost(8)`,
		`AssertThat(5).isAtMost(3)`,
	} {
		t.Run(data, func(t *testing.T) {
			globals, err := helper(t, data)
			require.Empty(t, globals)
			require.NotNil(t, err)
			require.IsType(t, &TruthAssertion{}, err)
		})
	}

}
