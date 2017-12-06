# [testman](https://github.com/CoveredCI/testman) ~ CoveredCI's minion [![TravisCI build status](https://travis-ci.org/CoveredCI/testman.svg?branch=master)](https://travis-ci.org/CoveredCI/testman/builds)

[CoveredCI](https://coveredci.com) is an automated JSON API testing service based on QuickCheck.

[testman](https://github.com/CoveredCI/testman) is the official open source client that executes the tests CoveredCI generates.

### Quick install

```shell
sh <(curl -sfSL https://goo.gl/Wb1JeU)
```

or the equivalent:

```shell
sh <(curl -sfSL https://raw.githubusercontent.com/CoveredCI/testman/master/misc/latest.sh)
```

### Example `.coveredci.yml` file:

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

### A more involved `.coveredci.yml`

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

Report bugs [on the project page](https://github.com/coveredCI/testman/issues) or [contact us](mailto:hi@coveredci.co).
