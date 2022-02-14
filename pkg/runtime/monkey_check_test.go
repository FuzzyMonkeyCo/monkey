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
	rt, err := newFakeMonkey(t, `
monkey.check(
    after_response = lambda ctx: None,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: check: missing argument for name`[1:])
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWithSpaces(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "bla bla",
    after_response = lambda ctx: None,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: only characters from `[1:]+tags.Alphabet+` should be in "bla bla"`)
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWhenEmpty(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "",
    after_response = lambda ctx: None,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: string is empty`[1:])
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWithNonASCIIChars(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "ééé",
    after_response = lambda ctx: None,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: only characters from `[1:]+tags.Alphabet+` should be in "ééé"`)
	require.Nil(t, rt)
}

func TestCheckNameIsIllegalWhenTooLong(t *testing.T) {
	name := strings.Repeat("blipblop", 32)
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "`[1:]+name+`",
    after_response = lambda ctx: None,
)
`+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: string is too long: "`[1:]+name+`"`)
	require.Nil(t, rt)
}

func TestCheckHookHasArityOf1(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "hook_has_arity_of_1",
    after_response = lambda a, b, c: None,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: after_response for check "hook_has_arity_of_1" must have only one param: ctx`[1:])
	require.Nil(t, rt)
}

func TestCheckStateMustBeDict(t *testing.T) {
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "state_must_be_dict",
    after_response = lambda ctx: None,
    state = 42,
)
`[1:]+someOpenAPI3Model)
	require.EqualError(t, err, `
Traceback (most recent call last):
  fuzzymonkey.star:1:13: in <toplevel>
Error in check: check: for parameter "state": got int, want dict`[1:])
	require.Nil(t, rt)
}

func TestCheckDoesNothing(t *testing.T) {
	name := "does_nothing"
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "`[1:]+name+`",
    after_response = lambda ctx: None,
)
`+someOpenAPI3Model)
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
	rt, err := newFakeMonkey(t, `
def set_state(ctx):
    ctx.state["ah"] = 42

monkey.check(
    name = "`[1:]+name+`",
    after_response = set_state,
)
`+someOpenAPI3Model)
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
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "`[1:]+name+`",
    after_response = lambda ctx: print("bla"),
)
`+someOpenAPI3Model)
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
	rt, err := newFakeMonkey(t, `
def set_state(ctx):
    ctx.state = 42

monkey.check(
    name = "`[1:]+name+`",
    after_response = set_state,
)
`+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Equal(t, []string{
			"runtime.userError",
			`state for check "good_error" must be dict`,
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(4), v.ExecutionSteps)
	}
}

func TestCheckErrorWhenNonProtoCompatibleStateAssignment(t *testing.T) {
	name := "check_error_when_non_proto_compatible_state_assignment"
	rt, err := newFakeMonkey(t, `
def thing(ctx):
    ctx.state["some_key"] = {"some_other_key": set([4, 2])}

monkey.check(
    name = "`[1:]+name+`",
    after_response = thing,
)
`+someOpenAPI3Model)
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
	rt, err := newFakeMonkey(t, `
def set_state(ctx):
    ctx.state["key"] = "value"

monkey.check(
    name = "`[1:]+name+`",
    after_response = set_state,
    state = {"key": "value"},
)
`+someOpenAPI3Model)
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

func TestCheckStateClears(t *testing.T) {
	name := "state_clears"
	rt, err := newFakeMonkey(t, `
def state_clears(ctx):
    assert that(ctx.state).has_size(1)
    ctx.state.clear()

monkey.check(
    name = "`[1:]+name+`",
    after_response = state_clears,
    state = {"key": "value"},
)
`+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for i := range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.NotEmpty(t, v.ElapsedNs)
		if i == 0 {
			require.Equal(t, "success", v.Status.String())
			require.Empty(t, v.Reason)
			require.Equal(t, 19, int(v.ExecutionSteps))
		} else {
			require.Equal(t, "failure", v.Status.String())
			require.Equal(t, []string{
				"*starlark.EvalError",
				"Traceback (most recent call last):",
				"  fuzzymonkey.star:2:36: in state_clears",
				"Error in has_size: Not true that <{}> has a size of <1>. It is <0>.",
			}, v.Reason)
			require.Equal(t, 11, int(v.ExecutionSteps))
		}
	}
}

func TestCheckAccessesStateThenRequest(t *testing.T) {
	name := "accesses_state_then_request"
	rt, err := newFakeMonkey(t, `
def hook(ctx):
    ctx.state["ns"] = ctx.response.elapsed_ns
    assert that(ctx.request.method).is_equal_to("GET")

monkey.check(
    name = "`[1:]+name+`",
    after_response = hook,
)
`+someOpenAPI3Model)
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
			"  fuzzymonkey.star:3:20: in hook",
			"Error: cannot access ctx.request after accessing ctx.state",
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(13), v.ExecutionSteps)
	}
}

func TestCheckMutatesAndAsserts(t *testing.T) {
	name := "mutates_and_asserts"
	rt, err := newFakeMonkey(t, `
def hook(ctx):
    method = ctx.request.method
    ctx.state["ns"] = ctx.response.elapsed_ns
    assert that(method).is_equal_to("GET")

monkey.check(
    name = "`[1:]+name+`",
    after_response = hook,
)
`+someOpenAPI3Model)
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
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "`[1:]+name+`",
    after_response = lambda ctx: assert that(ctx.request.method).is_equal_to("GET"),
)
`+someOpenAPI3Model)
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
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "`[1:]+name+`",
    after_response = lambda ctx: assert that(ctx.request.method).is_not_equal_to("GET"),
)
`+someOpenAPI3Model)
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
			"  fuzzymonkey.star:3:81: in lambda",
			`Error in is_not_equal_to: Not true that <"GET"> is not equal to <"GET">.`,
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(12), v.ExecutionSteps)
	}
}

func TestCheckIncorrectAssert(t *testing.T) {
	name := "incorrect_assert"
	rt, err := newFakeMonkey(t, `
monkey.check(
    name = "`[1:]+name+`",
    after_response = lambda ctx: assert that(ctx.request.method),
)
`+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)

	for range make([]struct{}, iters) {
		v := rt.runFakeUserCheck(t, name)
		require.Equal(t, name, v.Name)
		require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
		require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
		require.Equal(t, []string{
			"*starlarktruth.UnresolvedError",
			"Traceback (most recent call last):",
			"  fuzzymonkey.star:3:45: assert that(...) is missing an assertion",
		}, v.Reason)
		require.NotEmpty(t, v.ElapsedNs)
		require.Equal(t, uint64(7), v.ExecutionSteps)
	}
}
