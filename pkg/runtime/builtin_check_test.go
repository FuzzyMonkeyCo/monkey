//+build fakefs

package runtime

import (
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/stretchr/testify/require"
)

const iters = 5

func TestCheckNameIsPresent(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	hook = lambda ctx: None,
)`)
	require.EqualError(t, err, `Check: missing argument for name`)
	require.Nil(t, rt)
}

func TestCheckNameIsLegal(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "bla bla",
	hook = lambda ctx: None,
)`)
	require.EqualError(t, err, `bad name for Check: string contains spacing characters: "bla bla"`)
	require.Nil(t, rt)
}

func TestCheckHookHasArityOf1(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "hook_has_arity_of_1",
	hook = lambda a, b, c: None,
)`)
	require.EqualError(t, err, `hook for Check "hook_has_arity_of_1" must have only one param: ctx`)
	require.Nil(t, rt)
}

func TestCheckStateMustBeDict(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "state_must_be_dict",
	hook = lambda ctx: None,
	state = 42,
)`)
	require.EqualError(t, err, `Check: for parameter "state": got int, want dict`)
	require.Nil(t, rt)
}

func TestCheckDoesNothing(t *testing.T) {
	name := "does_nothing"
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "` + name + `",
	hook = lambda ctx: None,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_skipped, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(2), v.ExecutionSteps)
	}
}

func TestCheckMutatesExactlyOnce(t *testing.T) {
	name := "mutates_exactly_once"
	rt, err := newFakeMonkey(simplestPrelude + `
def set_state(ctx):
	ctx.state["ah"] = 42

Check(
	name = "` + name + `",
	hook = set_state,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for i := range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		if i == 0 {
			require.Equal(t, fm.Clt_CallVerifProgress_success, v.Status)
		} else {
			require.Equal(t, fm.Clt_CallVerifProgress_skipped, v.Status)
		}
		require.Equal(t, true, v.UserProperty)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(9), v.ExecutionSteps)
	}
}

func TestCheckErrorWhenNonDictStateAssignment(t *testing.T) {
	name := "good_error"
	rt, err := newFakeMonkey(simplestPrelude + `
def set_state(ctx):
	ctx.state = 42

Check(
	name = "` + name + `",
	hook = set_state,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Equal(t, []string{`state for Check "good_error" must be dict`}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(4), v.ExecutionSteps)
	}
}

func TestCheckMutatesNever(t *testing.T) {
	name := "mutates_never"
	rt, err := newFakeMonkey(simplestPrelude + `
def set_state(ctx):
	ctx.state["key"] = "value"

Check(
	name = "` + name + `",
	hook = set_state,
	state = {"key": "value"},
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_skipped, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(9), v.ExecutionSteps)
	}
}

func TestCheckAccessesStateThenRequest(t *testing.T) {
	name := "accesses_state_then_request"
	rt, err := newFakeMonkey(simplestPrelude + `
def hook(ctx):
	ctx.state["ns"] = ctx.response.elapsed_ns
	assert.that(ctx.request.method).is_equal_to("GET")

Check(
	name = "` + name + `",
	hook = hook,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Equal(t, []string{
			"Traceback (most recent call last):",
			"  fuzzymonkey.star:10:17: in hook",
			"Error: cannot access ctx.request after accessing ctx.state",
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(13), v.ExecutionSteps)
	}
}

func TestCheckMutatesAndAsserts(t *testing.T) {
	name := "mutates_and_asserts"
	rt, err := newFakeMonkey(simplestPrelude + `
def hook(ctx):
	method = ctx.request.method
	ctx.state["ns"] = ctx.response.elapsed_ns
	assert.that(method).is_equal_to("GET")

Check(
	name = "` + name + `",
	hook = hook,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_success, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(26), v.ExecutionSteps)
	}
}

func TestCheckJustAssertsTheTruth(t *testing.T) {
	name := "just_asserts_the_truth"
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "` + name + `",
	hook = lambda ctx: assert.that(ctx.request.method).is_equal_to("GET"),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_success, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(13), v.ExecutionSteps)
	}
}

func TestCheckJustAssertsWrong(t *testing.T) {
	name := "just_asserts_wrong"
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "` + name + `",
	hook = lambda ctx: assert.that(ctx.request.method).is_not_equal_to("GET"),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Equal(t, []string{
			"Traceback (most recent call last):",
			"  fuzzymonkey.star:10:68: in lambda",
			`Error in is_not_equal_to: Not true that <"GET"> is not equal to <"GET">.`,
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(12), v.ExecutionSteps)
	}
}

func TestCheckIncorrectAssert(t *testing.T) {
	name := "incorrect_assert"
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "` + name + `",
	hook = lambda ctx: assert.that(ctx.request.method),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, true, v.UserProperty)
		require.Equal(t, []string{
			"fuzzymonkey.star:10:32: assert.that(...) is missing an assertion",
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(7), v.ExecutionSteps)
	}
}
