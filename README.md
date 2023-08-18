# [monkey](https://github.com/FuzzyMonkeyCo/monkey) ~ FuzzyMonkeyCo's minion [![Goreport card](https://goreportcard.com/badge/github.com/FuzzyMonkeyCo/monkey)](https://goreportcard.com/report/github.com/FuzzyMonkeyCo/monkey)

[FuzzyMonkey](https://fuzzymonkey.co) is an automated API testing service that behaves as your users would and minimizes sequences of calls that lead to a violation of your software's properties.

[monkey](https://github.com/FuzzyMonkeyCo/monkey) is the official open source client that executes the tests FuzzyMonkey generates.

[![asciicast](https://asciinema.org/a/171571.png)](https://asciinema.org/a/171571?autoplay=1)

```
monkey M.m.p go1.20.7 linux amd64

Usage:
  monkey [-vvv]           env [VAR ...]
  monkey [-vvv] [-f STAR] fmt [-w]
  monkey [-vvv] [-f STAR] lint [--show-spec]
  monkey [-vvv] [-f STAR] exec (repl | start | reset | stop)
  monkey [-vvv] [-f STAR] schema [--validate-against=REF]
  monkey [-vvv] [-f STAR] fuzz [--intensity=N] [--seed=SEED]
                               [--label=KV]...
                               [--tags=TAGS | --exclude-tags=TAGS]
                               [--no-shrinking]
                               [--progress=PROGRESS]
                               [--time-budget-overall=DURATION]
                               [--only=REGEX]... [--except=REGEX]...
                               [--calls-with-input=SCHEMA]...  [--calls-without-input=SCHEMA]...
                               [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  monkey        [-f STAR] pastseed
  monkey        [-f STAR] logs [--previous=N]
  monkey [-vvv]           update
  monkey                  version | --version
  monkey                  help    | --help    | -h

Options:
  -v, -vv, -vvv                   Debug verbosity level
  -f STAR, --file=STAR            Name of the fuzzymonkey.star file
  version                         Show the version string
  update                          Ensures monkey is the latest version
  --intensity=N                   The higher the more complex the tests [default: 10]
  --time-budget-overall=DURATION  Stop testing after DURATION (e.g. '30s' or '5h')
  --seed=SEED                     Use specific parameters for the Random Number Generator
  --label=KV                      Labels that can help classification (format: key=value)
  --tags=TAGS                     Only run checks whose tags match at least one of these (comma separated)
  --exclude-tags=TAGS             Skip running checks whose tags match at least one of these (comma separated)
  --progress=PROGRESS             dots, bar, ci (defaults: dots)
  --only=REGEX                    Only test matching calls
  --except=REGEX                  Do not test these calls
  --calls-with-input=SCHEMA       Test calls which can take schema PTR as input
  --calls-without-output=SCHEMA   Test calls which never output schema PTR
  --validate-against=REF          Validate STDIN payload against given schema $ref
  --previous=N                    Select logs from Nth previous run [default: 1]

Try:
     export FUZZYMONKEY_API_KEY=fm_42
  monkey update
  monkey -f fm.star exec reset
  monkey fuzz --only /pets --calls-without-input=NewPet --seed=$(monkey pastseed)
  echo '"kitty"' | monkey schema --validate-against=#/components/schemas/PetKind
```

### Getting started

**Recommended way:** using [the GitHub Action](https://github.com/FuzzyMonkeyCo/setup-monkey).

Quick install:
```shell
curl -#fL https://git.io/FuzzyMonkey | BINDIR=/usr/local/bin sh

# or the equivalent:
curl -#fL https://raw.githubusercontent.com/FuzzyMonkeyCo/monkey/master/.godownloader.sh | BINDIR=/usr/local/bin sh
```

With Docker:
```shell
DOCKER_BUILDKIT=1 docker build -o=/usr/local/bin --platform=local https://github.com/FuzzyMonkeyCo/monkey.git

# or the faster:
DOCKER_BUILDKIT=1 docker build -o=/usr/local/bin --platform=local --build-arg PREBUILT=1 https://github.com/FuzzyMonkeyCo/monkey.git
```

Or simply install the [latest release](https://github.com/FuzzyMonkeyCo/monkey/releases/latest).

### Configuration

`monkey` uses [Starlark](https://github.com/bazelbuild/starlark) as its configuration language: a simple Python-like deterministic language.

#### Minimal example `fuzzymonkey.star` file


```python
OpenAPIv3(
  name = "dev_spec",
  file = "openapi/openapi.yaml",
  host = "http://localhost:3000",

  ExecReset = "curl -fsSL -X DELETE http://localhost:3000/api/1/items",
)
```

#### Demos

* [demo_erlang_cowboy_simpleREST](https://github.com/FuzzyMonkeyCo/demo_erlang_cowboy_simpleREST)

#### A more involved [`fuzzymonkey.star`](./fuzzymonkey.star)

```python
# Invariants of our APIs expressed in a Python-like language

assert that(monkey.env("TESTING_WHAT", "demo")).is_equal_to("demo")
SPEC = "pkg/runtime/testdata/jsonplaceholder.typicode.comv1.0.0_openapiv3.0.1_spec.yml"
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
echo ${BLA:-42}
BLA=$(( ${BLA:-42} + 1 ))
echo Resetting System Under Test...
    """,
)

## Add headers to some of the requests

def add_special_headers(ctx):
    """Shows how to modify an HTTP request before it is sent"""
    #before_request: lambda (req: CallRequestRaw.Input): pass

    # headers = dict([(kv.key,kv.values) for kv in req.headers])
    # headers['x-special'] = ['values!']
    # req.headers = headers
    # return req

    # req = req.http_request()
    # if req == None:
    #     print("`req` isn't an HTTP request")
    #     return
    # my_header = "X-Special"
    # assert that(dict([(pair.key, pair.values) for pair in req.headers])).does_not_contain_key(my_header)
    # req.headers.append(my_header, ["value!"])
    # print("Added some headers!")

    req = ctx.request
    if type(req) != "http_request":
        print("`ctx.request` isn't an HTTP request! It's a {}", type(req))
        return

    my_header = "X-Special"
    assert that(dict([(pair.key, pair.values) for pair in req.headers])).does_not_contain_key(my_header)
    req.headers.append(my_header, "value!")
    print("Added an extra header: {my_header}", my_header = my_header)

monkey.check(
    name = "adds_special_headers",
    before_request = add_special_headers,
    tags = ["special_headers"],
)
# just a check like others
#   no special treatment
# except for un-frozen ctx.request

# maybe place potential after_request + before_response in code

# no methods, maybe some lazy doing

## Ensure some general property

def ensure_lowish_response_time(ms):
    def responds_in_a_timely_manner(ctx):
        assert that(ctx.response).is_of_type("http_response")
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
    if type(ctx.request) != "http_request":
        return

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
```

### Issues?

Report bugs [on the project page](https://github.com/FuzzyMonkeyCo/monkey/issues) or [contact us](mailto:ook@fuzzymonkey.co).


## License

See [LICENSE](./LICENSE)
