package starlarkvalue

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestNonJSONLikeFails(t *testing.T) {
	require.NoError(t, ProtoCompatible(starlark.None))
	d := starlark.NewDict(1)
	k := starlark.MakeInt(42)
	require.NoError(t, ProtoCompatible(k))
	v := starlark.String("bla")
	require.NoError(t, ProtoCompatible(v))
	err := d.SetKey(k, v)
	require.NoError(t, err)
	require.EqualError(t, ProtoCompatible(d), `want string key, got: (dict) {42: "bla"}`)
}
