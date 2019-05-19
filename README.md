# [monkey](https://github.com/FuzzyMonkeyCo/monkey) ~ FuzzyMonkeyCo's minion [![TravisCI build status](https://travis-ci.org/FuzzyMonkeyCo/monkey.svg?branch=master)](https://travis-ci.org/FuzzyMonkeyCo/monkey/builds) [![Goreport card](https://goreportcard.com/badge/github.com/FuzzyMonkeyCo/monkey)](https://goreportcard.com/report/github.com/FuzzyMonkeyCo/monkey) [![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey?ref=badge_shield)

[FuzzyMonkey](https://fuzzymonkey.co) is an automated JSON API testing service based on QuickCheck.

[monkey](https://github.com/FuzzyMonkeyCo/monkey) is the official open source client that executes the tests FuzzyMonkey generates.

[![asciicast](https://asciinema.org/a/171571.png)](https://asciinema.org/a/171571?autoplay=1)

```
monkey	v0.0.0	0.19.1-165-g11b29c8-dirty	go1.12.5

Usage:
  monkey [-vvv] init [--with-magic]
  monkey [-vvv] env [VAR ...]
  monkey [-vvv] login [--user=USER]
  monkey [-vvv] fuzz [--tests=N] [--seed=SEED] [--tag=TAG]...
                     [--only=REGEX]... [--except=REGEX]...
                     [--calls-with-input=SCHEMA]... [--calls-without-input=SCHEMA]...
                     [--calls-with-output=SCHEMA]... [--calls-without-output=SCHEMA]...
  monkey [-vvv] shrink --test=ID [--seed=SEED] [--tag=TAG]...
  monkey [-vvv] lint [--show-spec] [--hide-config]
  monkey [-vvv] schema [--validate-against=REF]
  monkey [-vvv] exec (start | reset | stop)
  monkey [-vvv] -h | --help
  monkey [-vvv]      --update
  monkey [-vvv] -V | --version

Options:
  -v, -vv, -vvv                  Debug verbosity level
  -h, --help                     Show this screen
  -U, --update                   Ensures monkey is current
  -V, --version                  Show version
  --hide-config                  Do not show YAML configuration while linting
  --seed=SEED                    Use specific parameters for the RNG
  --validate-against=REF         Schema $ref to validate STDIN against
  --tag=TAG                      Labels that can help classification
  --test=ID                      Which test to shrink
  --tests=N                      Number of tests to run [default: 100]
  --only=REGEX                   Only test matching calls
  --except=REGEX                 Do not test these calls
  --calls-with-input=SCHEMA      Test calls which can take schema PTR as input
  --calls-without-output=SCHEMA  Test calls which never output schema PTR
  --user=USER                    Authenticate on fuzzymonkey.co as USER
  --with-magic                   Auto fill in schemas from random API calls

Try:
     export FUZZYMONKEY_API_KEY=42
  monkey --update
  monkey fuzz --only /pets --calls-without-input=NewPet --tests=0
  echo '"kitty"' | monkey schema --validate-against=#/components/schemas/PetKind
```

### Getting started

Quick install:

```shell
sh <(curl -#fSL http://goo.gl/3d7tPe)
```

or the equivalent:

```shell
sh <(curl -#fSL https://raw.githubusercontent.com/FuzzyMonkeyCo/monkey/master/misc/latest.sh)
```

### Example `.fuzzymonkey.yml` file:

```yaml
version: 0
documentation:
  kind: openapi_v2
  file: priv/openapi2v1.yaml
  host: localhost
  port: 6773
reset:
  - curl --fail -X DELETE http://localhost:6773/api/1/items
```

### A more involved `.fuzzymonkey.yml`

```yaml
version: 0

documentation:
  kind: openapi_v2
  file: priv/openapi2v1.json
  host: localhost
  port: '{{ env "CONTAINER_PORT" }}'


start:
  - CONTAINER_ID=$(docker run --rm -d -p 6773 cake_sample_master)
  - cmd="$(docker port $CONTAINER_ID 6773/tcp)"
  - CONTAINER_PORT=$(python -c "_, port = '$cmd'.split(':'); print(port)")
  - host=localhost
  - |
    until $(curl -# --output /dev/null --silent --fail --head http://$host:$CONTAINER_PORT/api/1/items); do
        printf .
        sleep 5
    done

reset:
  - "[[ 204 = $(curl -# --output /dev/null --write-out '%{http_code}' -X DELETE http://$host:$CONTAINER_PORT/api/1/items) ]]"

stop:
  - docker stop --time 5 $CONTAINER_ID
```

### Issues?

Report bugs [on the project page](https://github.com/FuzzyMonkeyCo/monkey/issues) or [contact us](mailto:ook@fuzzymonkey.co).
