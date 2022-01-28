package runtime

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonkeyModuleAttrsCount(t *testing.T) {
	require.Len(t, (*module)(nil).AttrNames(), moduleAttrs)
}

func TestMonkeyModuleAttrsNamesAreInOrder(t *testing.T) {
	names := (*module)(nil).AttrNames()
	sort.Strings(names)

	require.Equal(t, names, (*module)(nil).AttrNames())
}
