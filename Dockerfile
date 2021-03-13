# syntax=docker/dockerfile:1.2


## Build all platforms/os

#FIXME: lock goreleaser image
FROM --platform=$BUILDPLATFORM goreleaser/goreleaser AS monkey-build
COPY . .
RUN \
  --mount=target=/root/.cache,type=cache \
  --mount=target=/go/pkg/mod,type=cache \
    set -x \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
#FIXME: write CURRENT_TAG before CI publishes tag
 && CURRENT_TAG=0.29.0 goreleaser release --rm-dist --snapshot --skip-publish


## Export binaries for each os

FROM monkey-build AS monkey-build-darwin
RUN set -x \
 && tar zxvf ./dist/monkey-Darwin-x86_64.tar.gz -C .

FROM scratch AS binaries-darwin
COPY --from=monkey-build-darwin /go/monkey /

FROM monkey-build AS monkey-build-linux
RUN set -x \
 && tar zxvf ./dist/monkey-Linux-x86_64.tar.gz -C .
FROM scratch AS binaries-linux
COPY --from=monkey-build-linux /go/monkey /

FROM monkey-build AS monkey-build-windows
RUN set -x \
 && tar zxvf ./dist/monkey-Windows-x86_64.tar.gz -C .
FROM scratch AS binaries-windows
COPY --from=monkey-build-windows /go/monkey.exe /

FROM binaries-$TARGETOS AS binaries


## Default

FROM binaries
