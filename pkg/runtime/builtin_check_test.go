package runtime

import (
	"strings"
	"testing"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/tags"
	"github.com/stretchr/testify/require"
)

const iters = 5

func TestCheckNameIsPresent(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	after_response = lambda ctx: None,
)`)
	require.EqualError(t, err, `Check: missing argument for name`)
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWithSpaces(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "bla bla",
	after_response = lambda ctx: None,
)`)
	require.EqualError(t, err, `bad name for Check: string should only contain characters from `+tags.Alphabet+`: "bla bla"`)
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWhenEmpty(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "",
	after_response = lambda ctx: None,
)`)
	require.EqualError(t, err, `bad name for Check: string is empty`)
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWithNonASCIIChars(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "ééé",
	after_response = lambda ctx: None,
)`)
	require.EqualError(t, err, `bad name for Check: string should only contain characters from `+tags.Alphabet+`: "ééé"`)
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWhenTooLong(t *testing.T) {
	name := strings.Repeat("blipblop", 32)
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "` + name + `",
	after_response = lambda ctx: None,
)`)
	require.EqualError(t, err, `bad name for Check: string is too long: "`+name+`"`)
	require.Nil(t, rt)
}

func TestCheckHookHasArityOf1(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "hook_has_arity_of_1",
	after_response = lambda a, b, c: None,
)`)
	require.EqualError(t, err, `after_response for Check "hook_has_arity_of_1" must have only one param: ctx`)
	require.Nil(t, rt)
}

func TestCheckStateMustBeDict(t *testing.T) {
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "state_must_be_dict",
	after_response = lambda ctx: None,
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
	after_response = lambda ctx: None,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_skipped, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
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
	after_response = set_state,
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
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(9), v.ExecutionSteps)
	}
}

func TestCheckJustPrints(t *testing.T) {
	name := "just_prints"
	rt, err := newFakeMonkey(simplestPrelude + `
Check(
	name = "` + name + `",
	after_response = lambda ctx: print("bla"),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_success, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Empty(t, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(4), v.ExecutionSteps)
	}
}

func TestCheckErrorWhenNonDictStateAssignment(t *testing.T) {
	name := "good_error"
	rt, err := newFakeMonkey(simplestPrelude + `
def set_state(ctx):
	ctx.state = 42

Check(
	name = "` + name + `",
	after_response = set_state,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Equal(t, []string{
			"runtime.userError",
			`state for Check "good_error" must be dict`,
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(4), v.ExecutionSteps)
	}
}

func TestCheckErrorWhenNonProtoCompatibleStateAssignment(t *testing.T) {
	name := "check_error_when_non_proto_compatible_state_assignment"
	rt, err := newFakeMonkey(simplestPrelude + `
def thing(ctx):
	ctx.state["some_key"] = {"some_other_key": set([4, 2])}

Check(
	name = "` + name + `",
	after_response = thing,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, name)
	require.Equal(t, name, v.Name)
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"runtime.userError",
		"incompatible value (set): set([4, 2])",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}

func TestCheckMutatesNever(t *testing.T) {
	name := "mutates_never"
	rt, err := newFakeMonkey(simplestPrelude + `
def set_state(ctx):
	ctx.state["key"] = "value"

Check(
	name = "` + name + `",
	after_response = set_state,
	state = {"key": "value"},
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_skipped, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
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
	after_response = hook,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Equal(t, []string{
			"*starlark.EvalError",
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
	after_response = hook,
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_success, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
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
	after_response = lambda ctx: assert.that(ctx.request.method).is_equal_to("GET"),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_success, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
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
	after_response = lambda ctx: assert.that(ctx.request.method).is_not_equal_to("GET"),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Equal(t, []string{
			"*starlark.EvalError",
			"Traceback (most recent call last):",
			"  fuzzymonkey.star:10:78: in lambda",
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
	after_response = lambda ctx: assert.that(ctx.request.method),
)`)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Equal(t, []string{
			"starlarktruth.UnresolvedError",
			"fuzzymonkey.star:10:42: assert.that(...) is missing an assertion",
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(7), v.ExecutionSteps)
	}
}
