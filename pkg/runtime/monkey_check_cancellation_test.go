package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckCancellationLongtime(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "takes_a_very_long_time",
    after_response = lambda ctx: monkeh_sleep(10),
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	ctx := context.Background()

	start := time.Now()
	passed, err := rt.fakeUserChecks(ctx, t)
	require.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond)

	require.NoError(t, err)
	require.True(t, passed)
}

func TestCheckCancellationLongtimeOut(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "takes_a_very_long_time",
    after_response = lambda ctx: monkeh_sleep(10),
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	start := time.Now()
	passed, err := rt.fakeUserChecks(ctx, t)
	require.Less(t, time.Since(start), 10*time.Millisecond)

	require.Error(t, err, context.DeadlineExceeded)
	require.False(t, passed)
}

func TestCheckCancellationRunsAllChecksToCompletion(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "always_fails",
    after_response = lambda ctx: assert that(42).is_none(),
)
monkey.check(
    name = "takes_a_very_long_time",
    after_response = lambda ctx: monkeh_sleep(10),
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 2)

	ctx := context.Background()

	start := time.Now()
	passed, err := rt.fakeUserChecks(ctx, t)
	require.GreaterOrEqual(t, time.Since(start), 10*time.Millisecond)

	require.NoError(t, err)
	require.False(t, passed)
}

func TestCheckCancellationToCompletionButWithinTimeout(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "always_fails",
    after_response = lambda ctx: assert that(42).is_none(),
)
monkey.check(
    name = "takes_a_very_long_time",
    after_response = lambda ctx: monkeh_sleep(10),
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 2)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	start := time.Now()
	passed, err := rt.fakeUserChecks(ctx, t)
	require.Less(t, time.Since(start), 10*time.Millisecond)

	require.Error(t, err, context.DeadlineExceeded)
	require.False(t, passed)
}
