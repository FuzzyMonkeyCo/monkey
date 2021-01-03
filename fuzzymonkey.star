# Invariants of our APIs expressed in a Python-like language

print("$THIS_ENVIRONMENT_VARIABLE is", Env("THIS_ENVIRONMENT_VARIABLE", "not set"))

host, spec = "https://jsonplaceholder.typicode.com/", None
mode = Env("TESTING_WHAT")
if mode == "":
    spec = "pkg/modeler/openapiv3/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml"
elif mode == "other-thing":
    pass
else:
    fail("Unhandled testing mode '{}'".format(mode))
print("Now testing {}.".format(spec))

OpenAPIv3(
    name = "my_model",
    file = spec,
    host = host,
    # header_authorization = 'Bearer ' + ...,
    ExecReset = """
    printf 'Resetting state...
'
    """,
)

## Ensure some general property

# def generallyRootOfXSquaredIsX(State, response):
#     x = response["json"]['id']
#     if sqrt(x*x) != x:
#         fail("sqrt({}) != {}".format(x*x, x))

# TriggerActionAfterProbe(
#     name = 'sqrt(x * x) == x',
#     predicate = lambda State, response: True,
#     action = generallyRootOfXSquaredIsX,
# )

## Express stateful properties

# State is optional but has to be a Dict.
State = {
    "posts": {},
}

def actionAfterPosts(State, response):
    # When entering actions, response has already been validated and decoded.
    for post in response["json"]:
        # Set some state
        State["posts"][post["id"]] = post
    print("State has {} items".format(len(State["posts"])))
    return State

def ensureIdMatchesURL(State, response):
    # TODO: easy access to generated parameters. For instance:
    # post_id = response["request"]["parameters"]["path"]["{id}"] (note decoded int)
    post_id = int(response["request"]["url"].split("/")[-1])
    post = response["json"]

    # Implied: post_id in State['posts'] and post == State['posts'][post_id]
    # Ensure an API contract:
    AssertThat(post["id"]).isEqualTo(post_id)

def actionAfterGetExistingPost(State, response):
    post_id = int(response["request"]["url"].split("/")[-1])
    post = response["json"]

    # Verify that retrieved post matches local model
    AssertThat(State["posts"]).contains(post_id)
    AssertThat(post).isEqualTo(State["posts"][post_id])

TriggerActionAfterProbe(
    name = "Collect things",
    probe = ("monkey", "http", "response"),
    predicate = lambda State, response: all([
        response["request"]["method"] == "GET",
        response["request"]["url"].endswith("/posts"),
        response["status_code"] == 200,
    ]),
    # predicate = None,
    # match = {
    #     'request': {'method': 'GET', 'path': '/posts'},
    #     'status_code': 200,
    # },
    action = actionAfterPosts,
)

for action in [ensureIdMatchesURL, actionAfterGetExistingPost]:
    TriggerActionAfterProbe(
        # name = 'Ensure things match collected',
        probe = ("http", "response"),
        predicate = lambda State, response: all([
            response["request"]["method"] == "GET",
            response["request"]["url"].find("/posts/") != -1,
            response["status_code"] in range(200, 299),
            "id" in response["json"] and response["json"]["id"] in State["posts"],
        ]),
        # match = None,
        action = action,
    )

TriggerActionAfterProbe(
    probe = ("http", "response"),
    predicate = lambda State, response: all([
        response["request"]["method"] == "GET",
        response["request"]["url"].find("/posts/") != -1,
        response["request"]["url"].endswith("/comments"),
        response["status_code"] in range(200, 299),
    ]),
    action = lambda State, response: None,
)

## MISC

def sharing_1_2(State, response):
    """Sharing 1-2: ensure argument mutation doesn't corrupt model state
    """
    if "sharing" in State and State["sharing"] == 42:
        fail("State['sharing'] must not already be set")
    State["sharing"] = 42
    if not ("sharing" in State and State["sharing"] == 42):
        fail("State argument is not mutable")

TriggerActionAfterProbe(
    name = "sharing 1/2",
    predicate = lambda State, response: True,
    action = sharing_1_2,
)

def sharing_2_2(State, response):
    if "sharing" in State and State["sharing"] == 42:
        fail("State mutation must only happen through `return`")

TriggerActionAfterProbe(
    name = "sharing 2/2",
    predicate = lambda State, response: True,
    action = sharing_2_2,
)

### A test that always fails

# TriggerActionAfterProbe(
#     name = 'always failling',
#     predicate = lambda State, response: True,
#     action = lambda State, response: fail("Always fail!"),
# )
