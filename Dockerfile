# syntax=docker.io/docker/dockerfile:1.2@sha256:e2a8561e419ab1ba6b2fe6cbdf49fd92b95912df1cf7d313c3e2230a333fdbcc
# locked docker/dockerfile:1.2 ^ @ 2021/03/14 on linux/amd64

# locked goreleaser/goreleaser:latest @ 2021/03/14 on linux/amd64
FROM --platform=$BUILDPLATFORM docker.io/goreleaser/goreleaser@sha256:fa75344740e66e5bb55ad46426eb8e6c8dedbd3dcfa15ec1c41897b143214ae2 AS go-releaser

FROM go-releaser AS base
COPY go.??? .
RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
 && apk add the_silver_searcher \
 && ag --version \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
 && export CGO_ENABLED=0 \
 && go mod download
COPY . .


## CI checks

FROM base AS ci-check--lint
RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
 && export CGO_ENABLED=0 \
 && make lint \
 && git --no-pager diff && [[ $(git --no-pager diff --name-only | wc -l) = 0 ]]

FROM base AS ci-check--mod
RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
 && export CGO_ENABLED=0 \
 && go mod tidy \
 && go mod verify \
 && git --no-pager diff && [[ $(git --no-pager diff --name-only | wc -l) = 0 ]]

FROM base AS ci-check--test
RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
 && export CGO_ENABLED=0 \
 && make test.ci \
 && git --no-pager diff && [[ $(git --no-pager diff --name-only | wc -l) = 0 ]]


## Build all platforms/OS

FROM base AS monkey-build
RUN \
  --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
    # Prevents: $GOPATH/go.mod exists but should not
 && unset GOPATH \
 && grep -F . Tagfile \
 && CURRENT_TAG=$(cat Tagfile) goreleaser release --snapshot --parallelism=$(nproc)


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
