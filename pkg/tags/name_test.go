package tags

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegalName(t *testing.T) {
	require.NoError(t, LegalName("some_name"))
	require.NoError(t, LegalName("92"))
	require.NoError(t, LegalName("_ah"))
	require.NoError(t, LegalName("ah"))
	require.Error(t, LegalName("Ah"))
	require.Error(t, LegalName("ah ah"))
	require.Error(t, LegalName("ah-ah"))
	require.Error(t, LegalName("!!"))
}
