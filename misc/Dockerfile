FROM alpine
MAINTAINER Pierre Fenoll <pierrefenoll@gmail.com>

WORKDIR /app
COPY monkey-Linux-x86_64 /usr/local/bin/monkey

RUN set -x \
 && apk update && apk upgrade \
 && apk add git bash

ENTRYPOINT ["/usr/local/bin/monkey"]
