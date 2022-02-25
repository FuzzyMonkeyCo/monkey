package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestOurStarlarkVerbs(t *testing.T) {
	for category, names := range starlarkExtendedUniverse {
		var got []string
		switch category {
		case "bytes":
			var x starlark.Bytes
			got = x.AttrNames()
		case "dict":
			var x *starlark.Dict
			got = x.AttrNames()
		case "list":
			var x *starlark.List
			got = x.AttrNames()
		case "string":
			var x starlark.String
			got = x.AttrNames()
		case "set":
			var x *starlark.Set
			got = x.AttrNames()
		default:
			panic(category)
		}
		require.EqualValues(t, names, got, category)
	}
}

func TestCompareLimit(t *testing.T) {
	defer func(prev int) { starlarkCompareLimit = prev }(starlarkCompareLimit)
	starlarkCompareLimit = 1

	_, err := newFakeMonkey(t, `
x = [[37]]
assert that(x).is_equal_to(x)
`[1:]+someOpenAPI3Model)

	require.Equal(t, 1, starlark.CompareLimit)

	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:2:27: in <toplevel>
Error in is_equal_to: comparison exceeded maximum recursion depth`[1:])

}
