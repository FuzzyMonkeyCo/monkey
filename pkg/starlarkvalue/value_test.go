package starlarkvalue

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestNonJSONLikeFails(t *testing.T) {
	err := ProtoCompatible(starlark.None)
	require.NoError(t, err)

	k := starlark.MakeInt(42)
	err = ProtoCompatible(k)
	require.NoError(t, err)

	v := starlark.String("bla")
	err = ProtoCompatible(v)
	require.NoError(t, err)

	d := starlark.NewDict(1)
	err = d.SetKey(k, v)
	require.NoError(t, err)

	err = ProtoCompatible(d)
	require.EqualError(t, err, `want string key, got: (dict) {42: "bla"}`)

	s := starlark.NewSet(1)
	err = ProtoCompatible(s)
	require.EqualError(t, err, `incompatible value (set): set([])`)
}
