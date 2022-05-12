# Invariants of our APIs expressed in a Python-like language

assert that(monkey.env("TESTING_WHAT", "demo")).is_equal_to("demo")
SPEC = "pkg/modeler/openapiv3/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml"
print("Using {}.".format(SPEC))

monkey.openapi3(
    name = "my_spec",
    # Note: references to schemas in `file` are resolved relative to file's location.
    file = SPEC,
    host = "https://jsonplaceholder.typicode.com",
    # header_authorization = "Bearer {}".format(monkey.env("DEV_API_TOKEN")),
)

# Note: exec commands are executed in shells sharing the same environment variables,
# with `set -e` and `set -o pipefail` flags on.

# List here the commands to run so that the service providing "my_spec"
# can be restored to its initial state.
monkey.shell(
    name = "example_resetter",

    # Link to above defined spec.
    provides = ["my_spec"],

    # The following gets executed once per test
    #   so have these commands complete as fast as possible.
    # For best results, tests should start with a clean slate
    #   so limit filesystem access, usage of $RANDOM and non-reproducibility.
    reset = """
echo Resetting state...
    """,
)

## Ensure some general property

def ensure_lowish_response_time(ms):
    def responds_in_a_timely_manner(ctx):
        assert that(ctx.response.elapsed_ms).is_at_most(ms)

    return responds_in_a_timely_manner

monkey.check(
    name = "responds_in_a_timely_manner",
    after_response = ensure_lowish_response_time(500),
    tags = ["timings"],
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
        post_id = url.split("/")[-1]
        post = ctx.response.body

        # Ensure post ID in response matches ID in URL (an API contract):
        assert that(str(int(post["id"]))).is_equal_to(post_id)

        # Verify that retrieved post matches local model
        if post_id in ctx.state:
            assert that(post).is_equal_to(ctx.state[post_id])

        return

    if all([
        ctx.request.method == "GET",
        url.endswith("/posts"),
        ctx.response.status_code == 200,
    ]):
        # Store posts in state
        for post in ctx.response.body:
            post_id = str(int(post["id"]))
            ctx.state[post_id] = post
        print("State contains {} posts".format(len(ctx.state)))

monkey.check(
    name = "some_props",
    after_response = stateful_model_of_posts,
)

## Encapsulation: ensure each monkey.check owns its own ctx.state.

def encapsulation_1_of_2(ctx):
    """Show that state is not shared with encapsulation_2_of_2"""
    assert that(ctx.state).is_empty()

monkey.check(
    name = "encapsulation_1_of_2",
    after_response = encapsulation_1_of_2,
    tags = ["encapsulation"],
)

monkey.check(
    name = "encapsulation_2_of_2",
    after_response = lambda ctx: None,
    state = {"data": 42},
    tags = ["encapsulation"],
)

## A test that always fails

def this_always_fails(ctx):
    assert that(ctx).is_none()

monkey.check(
    name = "always_fails",
    after_response = this_always_fails,
    tags = ["failing"],
)
