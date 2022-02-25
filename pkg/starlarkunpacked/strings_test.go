package starlarkunpacked

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestUnpackingStrings(t *testing.T) {
	want := starlark.NewList([]starlark.Value{starlark.String("Oh"), starlark.String("nice!")})

	var sl Strings
	err := starlark.UnpackArgs("unpack", starlark.Tuple{want}, nil, "sl", &sl)
	require.NoError(t, err)

	require.Equal(t, []string{"Oh", "nice!"}, sl.GoStrings())
}

func TestUnpackingStringsWithBadItem(t *testing.T) {
	want := starlark.NewList([]starlark.Value{starlark.String("Oh"), starlark.MakeInt(42)})
	var sl Strings
	err := starlark.UnpackArgs("nope", starlark.Tuple{want}, nil, "sl", &sl)
	require.EqualError(t, err, `nope: for parameter sl: got int, want string`)
}
