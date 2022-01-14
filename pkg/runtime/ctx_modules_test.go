package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCtxUsage(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
def ctxchecks(ctx):
	assert.that(type(ctx)).is_equal_to("ctx")

	assert.that(type(ctx.request)).is_equal_to("ctx_request")
	assert.that(ctx.request).does_not_have_attribute("body")

	assert.that(type(ctx.response)).is_equal_to("ctx_response")
	assert.that(ctx.response).has_attribute("body")
	assert.that(ctx.response.elapsed_ms).is_within(50).of(1)

	assert.that(ctx.state).is_of_type("dict")

Check(
	name = "ctx_usage",
	after_response = ctxchecks,
)`[1:])
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_usage")
	require.Empty(t, v.Reason)
}
