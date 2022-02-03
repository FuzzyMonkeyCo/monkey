package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ensureFormattedAs(t *testing.T, pre, post string) {
	t.Helper()

	localCfgData = []byte(pre + " ") // trigger need for fmt
	err := Format(true)
	require.NoError(t, err)

	require.Equal(t, post, string(localCfgData))
}
