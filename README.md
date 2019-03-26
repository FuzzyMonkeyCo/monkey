# [monkey](https://github.com/FuzzyMonkeyCo/monkey) ~ FuzzyMonkeyCo's minion [![TravisCI build status](https://travis-ci.org/FuzzyMonkeyCo/monkey.svg?branch=master)](https://travis-ci.org/FuzzyMonkeyCo/monkey/builds) [![Goreport card](https://goreportcard.com/badge/github.com/FuzzyMonkeyCo/monkey)](https://goreportcard.com/report/github.com/FuzzyMonkeyCo/monkey)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey?ref=badge_shield)

[FuzzyMonkey](https://fuzzymonkey.co) is an automated JSON API testing service based on QuickCheck.

[monkey](https://github.com/FuzzyMonkeyCo/monkey) is the official open source client that executes the tests FuzzyMonkey generates.

[![asciicast](https://asciinema.org/a/171571.png)](https://asciinema.org/a/171571?autoplay=1)

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


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2FFuzzyMonkeyCo%2Fmonkey?ref=badge_large)