# [monkey](https://github.com/FuzzyMonkeyCo/monkey) ~ FuzzyMonkeyCo's minion [![TravisCI build status](https://travis-ci.org/FuzzyMonkeyCo/monkey.svg?branch=master)](https://travis-ci.org/FuzzyMonkeyCo/monkey/builds) [![Goreport card](https://goreportcard.com/badge/github.com/FuzzyMonkeyCo/monkey)](https://goreportcard.com/report/github.com/FuzzyMonkeyCo/monkey) [![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey?ref=badge_shield)

[FuzzyMonkey](https://fuzzymonkey.co) is an automated API testing service that behaves as your users would and minimizes sequences of calls that lead to a violation of your software's properties.

[monkey](https://github.com/FuzzyMonkeyCo/monkey) is the official open source client that executes the tests FuzzyMonkey generates.

[![asciicast](https://asciinema.org/a/171571.png)](https://asciinema.org/a/171571?autoplay=1)

```
monkey  M.m.p   feedb065        go1.13.5        amd64   linux

Usage:
  monkey [-vvv] init [--with-magic]
  monkey [-vvv] login [--user=USER]
  monkey [-vvv] fuzz [--intensity=N] [--shrink=ID] [--seed=SEED] [--tag=KV]...
                     [--only=REGEX]... [--except=REGEX]...
                     [--calls-with-input=SCHEMA]... [--calls-without-input=SCHEMA]...
                     [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  monkey [-vvv] lint [--show-spec]
  monkey [-vvv] schema [--validate-against=REF]
  monkey [-vvv] exec (repl | start | reset | stop)
  monkey [-vvv] env [VAR ...]
  monkey logs [--previous=N]
  monkey [-vvv] update
  monkey version | --version
  monkey help    | --help    | -h

Options:
  -v, -vv, -vvv                  Debug verbosity level
  version                        Show the version string
  update                         Ensures monkey is the latest version
  --intensity=N                  The higher the more complex the tests [default: 10]
  --seed=SEED                    Use specific parameters for the RNG
  --shrink=ID                    Which failed test to minimize
  --tag=KV                       Labels that can help classification (format: key=value)
  --only=REGEX                   Only test matching calls
  --except=REGEX                 Do not test these calls
  --calls-with-input=SCHEMA      Test calls which can take schema PTR as input
  --calls-without-output=SCHEMA  Test calls which never output schema PTR
  --user=USER                    Authenticate on fuzzymonkey.co as USER
  --validate-against=REF         Schema $ref to validate STDIN against
  --with-magic                   Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  monkey update
  monkey exec reset
  monkey fuzz --only /pets --calls-without-input=NewPet
  echo '"kitty"' | monkey schema --validate-against=#/components/schemas/PetKind
```

### Getting started

Quick install:

```shell
curl -#fL https://git.io/FuzzyMonkey | BINDIR=/usr/local/bin sh
```

or the equivalent:

```shell
curl -#fL https://raw.githubusercontent.com/FuzzyMonkeyCo/monkey/master/.godownloader.sh | BINDIR=/usr/local/bin sh
```

Or simply install the [latest release from here](https://github.com/FuzzyMonkeyCo/monkey/releases/latest).

### Configuration

`monkey` uses Starlark as its configuration language: a Python-like non Turing-complete language developped at Google for the Bazel build system.
From [https://github.com/bazelbuild/starlark](https://github.com/bazelbuild/starlark):
> Starlark (formerly known as Skylark) is a language intended for use as a configuration language. It was designed for the Bazel build system, but may be useful for other projects as well. [...]
>
> Starlark is a dialect of Python. Like Python, it is a dynamically typed language with high-level data types, first-class functions with lexical scope, and garbage collection. Independent Starlark threads execute in parallel, so Starlark workloads scale well on parallel machines. Starlark is a small and simple language with a familiar and highly readable syntax. You can use it as an expressive notation for structured data, defining functions to eliminate repetition, or you can use it to add scripting capabilities to an existing application.
> [...]
> Design Principles
> * Deterministic evaluation. Executing the same code twice will give the same results.
> * Hermetic execution. Execution cannot access the file system, network, system clock. It is safe to execute untrusted code.
> * Parallel evaluation. Modules can be loaded in parallel. To guarantee a thread-safe execution, shared data becomes immutable.
> * Simplicity. We try to limit the number of concepts needed to understand the code. Users should be able to quickly read and write code, even if they are not expert. The language should avoid pitfalls as much as possible.
> * Focus on tooling. We recognize that the source code will be read, analyzed, modified, by both humans and tools.
> * Python-like. Python is a widely used language. Keeping the language similar to Python can reduce the learning curve and make the semantics more obvious to users.

#### Example `fuzzymonkey.star` file


```python
OpenAPIv3(
  name = "my model on localhost",
  # Note: references to schemas in `file` are resolved relative to file's location.
  file = "openapi3.json",
  host = "http://localhost:6773",
  header_authorization = "Bearer " + "MY_DEVTIME_TOKEN",

  ExecReset = "curl --fail -X DELETE http://localhost:6773/api/1/items",
)
```

#### A more involved `fuzzymonkey.star`

```python
OpenAPIv3(
  name = "my other model running with Docker",
  file = "dist/openapi3v2.json",
  # `host` does not go through shell execution
  host_resolver = 'echo "https://localhost:${CONTAINER_PORT}"',
##host = Eval("echo "https://localhost:${CONTAINER_PORT}"), TODO

  # Note: commands are executed in shells sharing the same environment variables,
  # with `set -e` and `set -o pipefail` flags on.

  # The following get executed once per test
  #   so have these commands complete as fast as possible.
  # Also, make sure that each test starts from a clean slate
  #   otherwise results will be unreliable.
  ExecStart = """
  CONTAINER_ID=$(docker run --rm -d -p 8080 cake_sample_master)
  cmd="$(docker port $CONTAINER_ID 6773/tcp)"
  CONTAINER_PORT=$(python -c "_, port = '$cmd'.split(':'); print(port)")
  host=localhost
  until $(curl -# --output /dev/null --silent --fail --head http://$host:$CONTAINER_PORT/api/1/items); do
    printf .
    sleep 5
  done
  """,

  ExecReset = """
  [[ 204 = $(curl -# --output /dev/null --write-out '%{http_code}' -X DELETE http://$host:$CONTAINER_PORT/api/1/items) ]]
  """,

  ExecStop = "docker stop --time 5 $CONTAINER_ID",
)
```

### Issues?

Report bugs [on the project page](https://github.com/FuzzyMonkeyCo/monkey/issues) or [contact us](mailto:ook@fuzzymonkey.co).


## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey?ref=badge_large)
