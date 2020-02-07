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

func testEach(t *testing.T, m map[string]error) {
	for code, expectedErr := range m {
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

func TestComparables(t *testing.T) {
	testEach(t, map[string]error{
		//
		`AssertThat(5).isAtLeast(3)`: nil,
		`AssertThat(5).isAtLeast(5)`: nil,
		`AssertThat(5).isAtLeast(8)`: NewTruthAssertion("Not true that <5> is at least <8>."),
		//
		`AssertThat(5).isAtMost(5)`: nil,
		`AssertThat(5).isAtMost(8)`: nil,
		`AssertThat(5).isAtMost(3)`: NewTruthAssertion("Not true that <5> is at most <3>."),
		//
		`AssertThat(5).isGreaterThan(3)`: nil,
		`AssertThat(5).isGreaterThan(5)`: NewTruthAssertion("Not true that <5> is greater than <5>."),
		`AssertThat(5).isGreaterThan(8)`: NewTruthAssertion("Not true that <5> is greater than <8>."),
		//
		`AssertThat(5).isLessThan(8)`: nil,
		`AssertThat(5).isLessThan(5)`: NewTruthAssertion("Not true that <5> is less than <5>."),
		`AssertThat(5).isLessThan(3)`: NewTruthAssertion("Not true that <5> is less than <3>."),
		//
		`AssertThat(5).isAtLeast(None)`:     NewInvalidAssertion("isAtLeast"),
		`AssertThat(5).isAtMost(None)`:      NewInvalidAssertion("isAtMost"),
		`AssertThat(5).isGreaterThan(None)`: NewInvalidAssertion("isGreaterThan"),
		`AssertThat(5).isLessThan(None)`:    NewInvalidAssertion("isLessThan"),
		//
	})
}
