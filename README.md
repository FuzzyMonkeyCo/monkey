# [monkey](https://github.com/FuzzyMonkeyCo/monkey) ~ FuzzyMonkeyCo's minion [![TravisCI build status](https://travis-ci.org/FuzzyMonkeyCo/monkey.svg?branch=master)](https://travis-ci.org/FuzzyMonkeyCo/monkey/builds) [![Goreport card](https://goreportcard.com/badge/github.com/FuzzyMonkeyCo/monkey)](https://goreportcard.com/report/github.com/FuzzyMonkeyCo/monkey)

[FuzzyMonkey](https://fuzzymonkey.co) is an automated JSON API testing service based on QuickCheck.

[monkey](https://github.com/FuzzyMonkeyCo/monkey) is the official open source client that executes the tests FuzzyMonkey generates.

### Getting started

Quick install:

```shell
sh <(curl -fsSL http://goo.gl/3d7tPe)
```

or the equivalent:

```shell
sh <(curl -fsSL https://raw.githubusercontent.com/FuzzyMonkeyCo/monkey/master/misc/latest.sh)
```

Then run:

```shell
$ monkey fuzz
No validation errors found.
✓✓✓✓✓✓✓✓✓✓✗
✗✗✓✓✓✓✓✓✓
Ran 20 tests totalling 47 requests
A bug was detected after 11 tests then shrunk 9 times!
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
    until $(curl --output /dev/null --silent --fail --head http://$host:$CONTAINER_PORT/api/1/items); do
        printf .
        sleep 5
    done

reset:
  - "[[ 204 = $(curl --silent --output /dev/null --write-out '%{http_code}' -X DELETE http://$host:$CONTAINER_PORT/api/1/items) ]]"

stop:
  - docker stop --time 5 $CONTAINER_ID
```

### Issues?

Report bugs [on the project page](https://github.com/FuzzyMonkeyCo/monkey/issues) or [contact us](mailto:ook@fuzzymonkey.co).
