package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssertTrickAtRoot(t *testing.T) {
	code := `
assert that("this")
assert    that("that")
assert	that("too")
`[1:]
	_, err := newFakeMonkey(nil, code)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:2:15: in <toplevel>
Error in that: fuzzymonkey.star:1:12: assert that(...) is missing an assertion`[1:])
	ensureFormattedAs(t, code, `
assert that("this")
assert that("that")
assert that("too")
`[1:])

	_, err = newFakeMonkey(t, `
assert that("this").is_equal_to("that")
`[1:])
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:32: in <toplevel>
Error in is_equal_to: Not true that <"this"> is equal to <"that">.`[1:])
}

func TestAssertTrick(t *testing.T) {
	code := `
def some_check(ctx):
    assert that(ctx).is_not_none()

monkey.check(
    name = "some_check",
    after_response = some_check,
)
`[1:] + someOpenAPI3Model
	rt, err := newFakeMonkey(t, code)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	ensureFormattedAs(t, code, code)
}

func TestAssertTrickSpaces(t *testing.T) {
	ensureFormattedAs(t, `
assert	that(42)
	assert	that(42)
if True:
  assert	that(42)
x = lambda y:	assert	that(42)

# Well, assert			that(42)
# assert that("this").is_equal_to("that")
p[42] = lambda q:     assert     that(42)
`[1:], `
assert that(42)
assert that(42)
if True:
    assert that(42)
x = lambda y: assert that(42)

# Well, assert			that(42)
# assert that("this").is_equal_to("that")
p[42] = lambda q: assert that(42)
`[1:])
}

func TestAssertTrickNaked(t *testing.T) {
	_, err := newFakeMonkey(nil, `assert 42`)
	require.EqualError(t, err, `fuzzymonkey.star:1:10: got int literal, want newline`)

	_, err = newFakeMonkey(nil, `assert  True`)
	require.EqualError(t, err, `fuzzymonkey.star:1:13: got identifier, want newline`)
}
