package runtime

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestEnvReadsVar(t *testing.T) {
	err := os.Setenv("SOME_VAR", "42")
	require.NoError(t, err)
	defer func() {
		err := os.Unsetenv("SOME_VAR")
		require.NoError(t, err)
	}()

	rt, err := newFakeMonkey(t, `
value1 = monkey.env("SOME_VAR")
value2 = monkey.env("SOME_VAR")
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Equal(t, rt.globals["value1"], starlark.String("42"))
	require.Equal(t, rt.globals["value2"], starlark.String("42"))
}

func TestEnvReadsVarButIncorrectDefault(t *testing.T) {
	err := os.Setenv("SOME_VAR", "42")
	require.NoError(t, err)
	defer func() {
		err := os.Unsetenv("SOME_VAR")
		require.NoError(t, err)
	}()

	rt, err := newFakeMonkey(t, `
value = monkey.env("SOME_VAR", None)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:19: in <toplevel>
Error in env: expected string, got NoneType: None`[1:])
	require.Nil(t, rt)
}

func TestEnvReadsVarGoodDefault(t *testing.T) {
	err := os.Setenv("SOME_VAR", "42")
	require.NoError(t, err)
	defer func() {
		err := os.Unsetenv("SOME_VAR")
		require.NoError(t, err)
	}()

	rt, err := newFakeMonkey(t, `
value = monkey.env("SOME_VAR", "")
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Equal(t, rt.globals["value"], starlark.String("42"))
}

func TestEnvUnsetVarNoDefault(t *testing.T) {
	rt, err := newFakeMonkey(t, `
value = monkey.env("SOME_VAR")
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:19: in <toplevel>
Error in env: unset environment variable: "SOME_VAR"`[1:])
	require.Nil(t, rt)
}

func TestEnvUnsetVarWithDefault(t *testing.T) {
	rt, err := newFakeMonkey(t, `
value = monkey.env("SOME_VAR", "orelse")
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Equal(t, rt.globals["value"], starlark.String("orelse"))
}
