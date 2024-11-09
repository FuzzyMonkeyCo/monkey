package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
)

func TestCtxUsage(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctxchecks(ctx):
    """
    Ensure properties of ctx such as attributes, their types, presence...

    Args:
      ctx: the context that Monkey provides.
    """
    assert that(type(ctx)).is_equal_to("ctx")

    assert that(type(ctx.request)).is_equal_to("http_request")
    assert that(ctx.request).does_not_have_attribute("body")

    assert that(type(ctx.response)).is_equal_to("http_response")
    assert that(ctx.response.status_code).is_equal_to(404)
    assert that(ctx.response.elapsed_ms).is_within(50).of(1)
    assert that(ctx.response).has_attribute("body")
    assert that(ctx.response.body).contains_key("error")

    assert that(ctx.state).is_of_type("dict")

monkey.check(
    name = "ctx_usage",
    after_response = ctxchecks,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_usage")
	require.Empty(t, v.Reason)
}

func TestCtxRequestHeadersFrozen(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctx_request_headers_frozen(ctx):
    """
    A request's headers are immutable.

    Args:
      ctx: the context that Monkey provides.
    """
    assert that(dict(ctx.request.headers)).has_size(1)
    JSON_MIME = "application/json"
    assert that(ctx.request.headers.get("Accept")).is_equal_to(JSON_MIME)
    ctx.request.headers.set("set", ["some", "values"])

monkey.check(
    name = "ctx_request_headers_frozen",
    after_response = ctx_request_headers_frozen,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_request_headers_frozen")
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"*starlark.EvalError",
		"Traceback (most recent call last):",
		"  fuzzymonkey.star:11:24: in ctx_request_headers_frozen",
		"Error: cannot set frozen hash table",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}

func TestCtxRequestBodyFrozen(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctx_request_body_frozen(ctx):
    assert that(ctx.request).has_attribute("body")
    assert that(ctx.request.body).does_not_contain_key("set")
    ctx.request.body["set"] = ["some", "values"]
    # assert that(ctx.request.body).does_not_contain_key("set")

monkey.check(
    name = "ctx_request_body_frozen",
    after_response = ctx_request_body_frozen,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_request_body_frozen")
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"*starlark.EvalError",
		"Traceback (most recent call last):",
		"  fuzzymonkey.star:4:21: in ctx_request_body_frozen",
		"Error: cannot insert into frozen hash table",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}

func TestCtxRequestBodyFrozenBis(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctx_request_body_frozen_bis(ctx):
    assert that(ctx.request).has_attribute("body")
    rep = ctx.request.body
    assert that(rep).does_not_contain_key("set")
    rep["set"] = ["some", "values"]
    # assert that(rep).does_not_contain_key("set")
    # assert that(ctx.request.body).does_not_contain_key("set")

monkey.check(
    name = "ctx_request_body_frozen_bis",
    after_response = ctx_request_body_frozen_bis,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_request_body_frozen_bis")
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"*starlark.EvalError",
		"Traceback (most recent call last):",
		"  fuzzymonkey.star:5:8: in ctx_request_body_frozen_bis",
		"Error: cannot insert into frozen hash table",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}

func TestCtxResponseBodyFrozen(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctx_response_body_frozen(ctx):
    assert that(ctx.response).has_attribute("body")
    assert that(ctx.response.body).does_not_contain_key("set")
    ctx.response.body["set"] = ["some", "values"]
    # assert that(ctx.response.body).does_not_contain_key("set")

monkey.check(
    name = "ctx_response_body_frozen",
    after_response = ctx_response_body_frozen,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_response_body_frozen")
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"*starlark.EvalError",
		"Traceback (most recent call last):",
		"  fuzzymonkey.star:4:22: in ctx_response_body_frozen",
		"Error: cannot insert into frozen hash table",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}

func TestCtxResponseHeadersFrozen(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctx_response_headers_frozen(ctx):
    """
    A response's headers are immutable.

    Args:
      ctx: the context that Monkey provides.
    """
    assert that(dict(ctx.response.headers)).has_size(11)
    HEADERS = [
        "Access-Control-Allow-Credentials",
        "Age",
        "Cache-Control",
        "Cf-Cache-Status",
        "Cf-Ray",
        "Content-Length",
        "Content-Type",
        "Date",
        "Etag",
        "Expect-Ct",
        "Expires",
    ]
    assert that(dict(ctx.response.headers)).contains_all_in(HEADERS).in_order()
    assert that(ctx.response.headers.get("Age")).is_equal_to("0")
    ctx.response.headers.set("set", ["some", "values"])

monkey.check(
    name = "ctx_response_headers_frozen",
    after_response = ctx_response_headers_frozen,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_response_headers_frozen")
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"*starlark.EvalError",
		"Traceback (most recent call last):",
		"  fuzzymonkey.star:24:25: in ctx_response_headers_frozen",
		"Error: cannot set frozen hash table",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}

func TestCtxResponseBodyFrozenBis(t *testing.T) {
	rt, err := newFakeMonkey(t, `
def ctx_response_body_frozen_bis(ctx):
    assert that(ctx.response).has_attribute("body")
    rep = ctx.response.body
    assert that(rep).does_not_contain_key("set")
    rep["set"] = ["some", "values"]
    # assert that(rep).does_not_contain_key("set")
    # assert that(ctx.response.body).does_not_contain_key("set")

monkey.check(
    name = "ctx_response_body_frozen_bis",
    after_response = ctx_response_body_frozen_bis,
)
`[1:]+someOpenAPI3Model)
	require.NoError(t, err)
	require.Len(t, rt.checks, 1)
	v := rt.runFakeUserCheck(t, "ctx_response_body_frozen_bis")
	require.Equal(t, fm.Clt_CallVerifProgress_failure, v.Status)
	require.Equal(t, fm.Clt_CallVerifProgress_after_response, v.Origin)
	require.Equal(t, []string{
		"*starlark.EvalError",
		"Traceback (most recent call last):",
		"  fuzzymonkey.star:5:8: in ctx_response_body_frozen_bis",
		"Error: cannot insert into frozen hash table",
	}, v.Reason)
	require.NotEmpty(t, v.ElapsedNs)
	require.NotEmpty(t, v.ExecutionSteps)
}
