# [testman](https://github.com/CoveredCI/testman) ~ CoveredCI's minion [![TravisCI build status](https://travis-ci.org/CoveredCI/testman.svg?branch=master)](https://travis-ci.org/CoveredCI/testman/builds)

[CoveredCI](https://coveredci.com) is an automated JSON API testing service based on QuickCheck.

[testman](https://github.com/CoveredCI/testman) is the official open source client that executes the tests CoveredCI generates.

Example `.coveredci.yml` file:

~~~yaml
version: '0'

documentation:
  kind: openapi_v2
  file: priv/openapi2v1.json
  host: localhost
  port: '{{ env "CONTAINER_PORT" }}'
  # port: 6773


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
  # - curl --fail -X DELETE http://localhost:$CONTAINER_PORT/api/1/items
  - "[[ 204 = $(curl --silent --output /dev/null --write-out '%{http_code}' -X DELETE http://$host:$CONTAINER_PORT/api/1/items) ]]"

stop:
  - docker stop $CONTAINER_ID
~~~
