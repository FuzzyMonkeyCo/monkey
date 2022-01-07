# [monkey](https://github.com/FuzzyMonkeyCo/monkey) ~ FuzzyMonkeyCo's minion [![Goreport card](https://goreportcard.com/badge/github.com/FuzzyMonkeyCo/monkey)](https://goreportcard.com/report/github.com/FuzzyMonkeyCo/monkey)

[FuzzyMonkey](https://fuzzymonkey.co) is an automated API testing service that behaves as your users would and minimizes sequences of calls that lead to a violation of your software's properties.

[monkey](https://github.com/FuzzyMonkeyCo/monkey) is the official open source client that executes the tests FuzzyMonkey generates.

[![asciicast](https://asciinema.org/a/171571.png)](https://asciinema.org/a/171571?autoplay=1)

```
monkey M.m.p go1.17.5 linux amd64

Usage:
  monkey [-vvv] fuzz [--intensity=N] [--seed=SEED] [--label=KV]...
                     [--tags=TAGS | --exclude-tags=TAGS]
                     [--no-shrinking]
                     [--progress=PROGRESS]
                     [--time-budget-overall=DURATION]
                     [--only=REGEX]... [--except=REGEX]...
                     [--calls-with-input=SCHEMA]... [--calls-without-input=SCHEMA]...
                     [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  monkey [-vvv] lint [--show-spec]
  monkey [-vvv] fmt [-w]
  monkey [-vvv] schema [--validate-against=REF]
  monkey [-vvv] exec (repl | start | reset | stop)
  monkey [-vvv] env [VAR ...]
  monkey        logs [--previous=N]
  monkey        pastseed
  monkey [-vvv] update
  monkey        version | --version
  monkey        help    | --help    | -h

Options:
  -v, -vv, -vvv                   Debug verbosity level
  version                         Show the version string
  update                          Ensures monkey is the latest version
  --intensity=N                   The higher the more complex the tests [default: 10]
  --time-budget-overall=DURATION  Stop testing after DURATION (e.g. '30s' or '5h')
  --seed=SEED                     Use specific parameters for the Random Number Generator
  --label=KV                      Labels that can help classification (format: key=value)
  --tags=TAGS                     Only run Check.s whose tags match at least one of these (comma separated)
  --progress=PROGRESS             dots, bar, ci (defaults: dots)
  --only=REGEX                    Only test matching calls
  --except=REGEX                  Do not test these calls
  --calls-with-input=SCHEMA       Test calls which can take schema PTR as input
  --calls-without-output=SCHEMA   Test calls which never output schema PTR
  --validate-against=REF          Schema $ref to validate STDIN against

Try:
     export FUZZYMONKEY_API_KEY=fm_42
  monkey update
  monkey exec reset
  monkey fuzz --only /pets --calls-without-input=NewPet --seed=$(monkey pastseed)
  echo '"kitty"' | monkey schema --validate-against=#/components/schemas/PetKind
```

### Getting started

**Recommended way:** using [the GitHub Action](https://github.com/FuzzyMonkeyCo/action-monkey).

Quick install:
```shell
curl -#fL https://git.io/FuzzyMonkey | BINDIR=/usr/local/bin sh

# or the equivalent:
curl -#fL https://raw.githubusercontent.com/FuzzyMonkeyCo/monkey/master/.godownloader.sh | BINDIR=/usr/local/bin sh
```

With Docker:
```shell
DOCKER_BUILDKIT=1 docker build -o=. --platform=local git://github.com/FuzzyMonkeyCo/monkey

# or the faster:
DOCKER_BUILDKIT=1 docker build -o=. --platform=local --build-arg PREBUILT=1 git://github.com/FuzzyMonkeyCo/monkey
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
    # Note: references to schemas in `file` are resolved relative to file's location.
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
    after_response = lambda ctx: assert.that(ctx.response.elapsed_ns).is_at_most(500e6),
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
    after_response = stateful_model_of_posts,
)

## Encapsulation: ensure each Check owns its own ctx.state.

def encapsulation_1_of_2(ctx):
    """Show that state is not shared with encapsulation_2_of_2"""
    assert.that(ctx.state).is_empty()

Check(
    name = "encapsulation_1_of_2",
    after_response = encapsulation_1_of_2,
    tags = ["encapsulation"],
)

Check(
    name = "encapsulation_2_of_2",
    after_response = lambda ctx: None,
    state = {"data": 42},
    tags = ["encapsulation"],
)

## A test that always fails

Check(
    name = "always_fails",
    after_response = lambda ctx: assert.that(None).is_not_none(),
    tags = ["failing"],
)
```

### Issues?

Report bugs [on the project page](https://github.com/FuzzyMonkeyCo/monkey/issues) or [contact us](mailto:ook@fuzzymonkey.co).


## License

See [LICENSE](./LICENSE)
