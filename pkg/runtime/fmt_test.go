package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ensureFormattedAs(t *testing.T, pre, post string) {
	t.Helper()

	starfileData = []byte(pre + " ") // trigger need for fmt
	err := Format(starFile, true)
	require.NoError(t, err)

	require.Equal(t, post, string(starfileData))
}
