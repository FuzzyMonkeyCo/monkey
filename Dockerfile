# syntax=docker/dockerfile:1.2


## Build all platforms/OS

# locked goreleaser/goreleaser:latest @ 2021/03/14
FROM --platform=$BUILDPLATFORM goreleaser/goreleaser@sha256:fa75344740e66e5bb55ad46426eb8e6c8dedbd3dcfa15ec1c41897b143214ae2 AS monkey-build
COPY . .
RUN \
  --mount=target=/root/.cache,type=cache \
  --mount=target=/go/pkg/mod,type=cache \
    set -ux \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
 && grep -F . Tagfile \
 && CURRENT_TAG=$(cat Tagfile) goreleaser release --parallelism=$(nproc) --skip-publish


## Goreleaser's dist/ for GitHub release

FROM scratch AS goreleaser-dist
COPY --from=monkey-build /go/dist/checksums.sha256.txt /
COPY --from=monkey-build /go/dist/monkey-*.tar.gz /
COPY --from=monkey-build /go/dist/monkey-*.zip /


## Binaries for each OS

FROM monkey-build AS monkey-build-darwin
RUN set -ux \
 && tar zxvf ./dist/monkey-Darwin-x86_64.tar.gz -C .

FROM scratch AS binaries-darwin
COPY --from=monkey-build-darwin /go/monkey /

FROM monkey-build AS monkey-build-linux
RUN set -ux \
 && tar zxvf ./dist/monkey-Linux-x86_64.tar.gz -C .
FROM scratch AS binaries-linux
COPY --from=monkey-build-linux /go/monkey /

FROM monkey-build AS monkey-build-windows
RUN set -ux \
 && tar zxvf ./dist/monkey-Windows-x86_64.tar.gz -C .
FROM scratch AS binaries-windows
COPY --from=monkey-build-windows /go/monkey.exe /

FROM binaries-$TARGETOS AS binaries


## Default target

FROM binaries
