package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStarfileCanBeChanged(t *testing.T) {
	defer func(prev string) { starFile = prev }(starFile)
	starFile = "fm.star"

	code := someOpenAPI3Model[1:]
	rt, err := newFakeMonkey(t, code)
	require.NoError(t, err)

	ctx := context.Background()
	passed, err := rt.fakeUserChecks(ctx, t)
	require.NoError(t, err)
	require.True(t, passed)

	ensureFormattedAs(t, code, code)
}

func TestStarfileCanBeChangedAndShowsUpInBT(t *testing.T) {
	defer func(prev string) { starFile = prev }(starFile)
	starFile = "bla.star"

	code := `
monkey.check(
    name = "some_check_has_typo",
    after_reponse = lambda ctx: 42,
)
`[1:] + someOpenAPI3Model

	_, err := newFakeMonkey(t, code)
	require.EqualError(t, err, `
Traceback (most recent call last):
  bla.star:1:13: in <toplevel>
Error in check: check: unexpected keyword argument "after_reponse" (did you mean after_response?)`[1:])
}
