package starlarkunpacked

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestUnpackingUniqueStrings(t *testing.T) {
	want := starlark.NewList([]starlark.Value{starlark.String("Oh"), starlark.String("nice!")})

	var sl UniqueStrings
	err := starlark.UnpackArgs("unpack", starlark.Tuple{want}, nil, "sl", &sl)
	require.NoError(t, err)

	require.Equal(t, []string{"Oh", "nice!"}, sl.GoStrings())
}

func TestUnpackingUniqueStringsWithBadItem(t *testing.T) {
	want := starlark.NewList([]starlark.Value{starlark.String("Oh"), starlark.MakeInt(42)})
	var sl UniqueStrings
	err := starlark.UnpackArgs("nope", starlark.Tuple{want}, nil, "sl", &sl)
	require.EqualError(t, err, `nope: for parameter sl: got int, want string`)
}

func TestUnpackingUniqueStringsWithDuplicateItem(t *testing.T) {
	want := starlark.NewList([]starlark.Value{starlark.String("Oh"), starlark.String("Oh")})
	var sl UniqueStrings
	err := starlark.UnpackArgs("dupe", starlark.Tuple{want}, nil, "sl", &sl)
	require.EqualError(t, err, `dupe: for parameter sl: "Oh" appears more than once`)
}
