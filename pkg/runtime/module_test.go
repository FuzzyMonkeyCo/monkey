package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMonkeyModuleAttrsCount(t *testing.T) {
	names := (*module)(nil).AttrNames()
	require.Len(t, names, moduleAttrs)
}

func TestMonkeyModuleAttrsNamesAreInOrder(t *testing.T) {
	names := (*module)(nil).AttrNames()
	require.IsIncreasing(t, names)
}
