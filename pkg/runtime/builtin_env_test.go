//go:build fakefs
// +build fakefs

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

	rt, err := newFakeMonkey(simplestPrelude + `
value1 = Env("SOME_VAR")
value2 = Env("SOME_VAR")
`)
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

	rt, err := newFakeMonkey(simplestPrelude + `value = Env("SOME_VAR", None)`)
	require.EqualError(t, err, `expected string, got NoneType: None`)
	require.Nil(t, rt)
}

func TestEnvReadsVarGoodDefault(t *testing.T) {
	err := os.Setenv("SOME_VAR", "42")
	require.NoError(t, err)
	defer func() {
		err := os.Unsetenv("SOME_VAR")
		require.NoError(t, err)
	}()

	rt, err := newFakeMonkey(simplestPrelude + `value = Env("SOME_VAR", "")`)
	require.NoError(t, err)
	require.Equal(t, rt.globals["value"], starlark.String("42"))
}

func TestEnvUnsetVarNoDefault(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `value = Env("SOME_VAR")`)
	require.EqualError(t, err, `unset environment variable: "SOME_VAR"`)
	require.Nil(t, rt)
}

func TestEnvUnsetVarWithDefault(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `value = Env("SOME_VAR", "orelse")`)
	require.NoError(t, err)
	require.Equal(t, rt.globals["value"], starlark.String("orelse"))
}
