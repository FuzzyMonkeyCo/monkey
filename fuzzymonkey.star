# Invariants of our APIs expressed in a Python-like language

print("$THIS_ENVIRONMENT_VARIABLE is", Env("THIS_ENVIRONMENT_VARIABLE", "not set"))

host, spec = "https://jsonplaceholder.typicode.com", None
mode = Env("TESTING_WHAT", "")
if mode == "":
    spec = "pkg/modeler/openapiv3/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml"
elif mode == "other-thing":
    pass
else:
    assert.that(mode).is_equal_to("unhandled testing mode")
print("Now testing {}.".format(spec))

OpenAPIv3(
    name = "my_model",
    file = spec,
    host = "{host}:{port}".format(host = host, port = Env("DEV_PORT", "443")),
    # header_authorization = "Bearer {}".format(Env("DEV_API_TOKEN")),

    # Note: exec commands are executed in shells sharing the same environment variables,
    # with `set -e` and `set -o pipefail` flags on.

    # The following get executed once per test
    #   so have these commands complete as fast as possible.
    # Also, make sure that each test starts from a clean slate
    #   otherwise results will be unreliable.
    ExecReset = """
    echo Resetting state...
    """,
)

## Ensure some general property

Check(
    name = "responds_in_a_timely_manner",
    hook = lambda ctx: assert.that(ctx.response.elapsed_ns).is_at_most(500e6),
    tags = ["timing"],
)

## Express stateful properties

def stateful_model_of_posts(ctx):
    """Properties on posts. State collects posts returned by API."""

    # NOTE: response has already been decoded & validated for us.

    url = ctx.request.url

    if all([
        ctx.request.method == "GET",
        "/posts/" in url and url[-1] in "1234567890",  # /posts/{post_id}
        ctx.response.status_code in range(200, 299),
    ]):
        post_id = int(url.split("/")[-1])
        post = ctx.response.body

        # Ensure post ID in response matches ID in URL (an API contract):
        assert.that(post["id"]).is_equal_to(post_id)

        # Verify that retrieved post matches local model
        if post_id in ctx.state:
            assert.that(post).is_equal_to(ctx.state[post_id])

        return

    if all([
        ctx.request.method == "GET",
        url.endswith("/posts"),
        ctx.response.status_code == 200,
    ]):
        # Store posts in state
        for post in ctx.response.body:
            post_id = int(post["id"])
            ctx.state[post_id] = post
        print("State contains {} posts".format(len(ctx.state)))

Check(
    name = "some_props",
    hook = stateful_model_of_posts,
)

## Encapsulation: ensure each Check owns its own ctx.state.

def encapsulation_1_of_2(ctx):
    """Show that state is not shared with encapsulation_2_of_2"""
    assert.that(ctx.state).is_empty()

Check(
    name = "encapsulation_1_of_2",
    hook = encapsulation_1_of_2,
    tags = ["encapsulation"],
)

Check(
    name = "encapsulation_2_of_2",
    hook = lambda ctx: None,
    state = {"data": 42},
    tags = ["encapsulation"],
)

## A test that always fails

Check(
    name = "always_fails",
    hook = lambda ctx: assert.that(None).is_not_none(),
    tags = ["failing"],
)
