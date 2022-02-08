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
