# syntax=docker.io/docker/dockerfile:1.2@sha256:e2a8561e419ab1ba6b2fe6cbdf49fd92b95912df1cf7d313c3e2230a333fdbcc
# locked docker/dockerfile:1.2 ^ @ 2021/03/14 on linux/amd64

# Use --build-arg PREBUILT=1 with default target to fetch binaries from GitHub releases
ARG PREBUILT

ARG OS=Linux
ARG ARCH=x86_64

# locked goreleaser/goreleaser:latest @ 2021/03/14 on linux/amd64
FROM --platform=$BUILDPLATFORM docker.io/goreleaser/goreleaser@sha256:fa75344740e66e5bb55ad46426eb8e6c8dedbd3dcfa15ec1c41897b143214ae2 AS go-releaser
# On this image:
#  go env GOCACHE    => /root/.cache/go-build
#  go env GOMODCACHE => /go/pkg/mod


FROM go-releaser AS base
WORKDIR /w
ENV CGO_ENABLED=0
COPY go.??? .
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/var/cache/apk ln -vs /var/cache/apk /etc/apk/cache && \
    set -ux \
 && apk add the_silver_searcher \
 && ag --version \
 && apk add git \
 && git version \
 && git init \
 && git add -A . \
 && go mod download \
 && git --no-pager diff && [[ 0 -eq $(git --no-pager diff --name-only | wc -l) ]]
COPY . .


## CI checks

FROM base AS ci-check--lint
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && make lint \
 && git --no-pager diff && [[ 0 -eq $(git --no-pager diff --name-only | wc -l) ]]

FROM base AS ci-check--mod
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && go mod tidy \
 && go mod verify \
 && git --no-pager diff && [[ 0 -eq $(git --no-pager diff --name-only | wc -l) ]]

FROM base AS ci-check--test
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && go test -tags fakefs -count 10 ./... \
 && git --no-pager diff && [[ 0 -eq $(git --no-pager diff --name-only | wc -l) ]]


## Build all platforms/OS

FROM base AS monkey-build
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && grep -F . Tagfile \
 && CURRENT_TAG=$(cat Tagfile) goreleaser release --snapshot


## Goreleaser's dist/ for GitHub release

FROM scratch AS goreleaser-dist-many
COPY --from=monkey-build /w/dist/checksums.sha256.txt /
COPY --from=monkey-build /w/dist/monkey-*.tar.gz /
COPY --from=monkey-build /w/dist/monkey-*.zip /
FROM scratch AS goreleaser-dist
COPY --from=goreleaser-dist-many / /


## Binaries for each OS

FROM alpine AS archmap-builder
ARG OS
ARG ARCH
RUN \
    set -ux \
 && ext=tar.gz \
 # More at https://stackoverflow.com/a/27776822/1418165
 && case "$OS" in \
      Linux) os=Linux ;; \
      Darwin) os=Darwin ;; \
      CYGWIN*|MINGW32*|MSYS*|MINGW*) os=Windows ; ext=zip ;; \
      *) echo "Unhandled OS '$OS'" && exit 1 ;; \
    esac \
 && case "$ARCH" in \
      i386) arch=i386 ;; \
      x86_64) arch=x86_64 ;; \
      *) echo "Unhandled ARCH '$ARCH'" && exit 1 ;; \
    esac \
 && echo monkey-$os-$arch.$ext >/archmap

FROM scratch AS archmap
COPY --from=archmap-builder /archmap /


FROM monkey-build AS zxf
RUN \
    --mount=from=archmap,source=/archmap,target=/archmap \
    set -ux \
 && tar zxvf ./dist/$(cat /archmap) -C .
FROM scratch AS binaries-
COPY --from=zxf /w/monkey* /

FROM alpine AS monkey-prebuilt
WORKDIR /w
RUN \
  --mount=type=cache,target=/var/cache/apk ln -vs /var/cache/apk /etc/apk/cache && \
    set -ux \
 && apk update \
 && apk add curl ca-certificates
RUN \
    --mount=source=Tagfile,target=Tagfile \
    --mount=from=archmap,source=/archmap,target=/archmap \
    set -ux \
 && TAG=$(cat Tagfile) \
 && ARCHIVE=$(cat /archmap) \
 && curl -fsSL -o $ARCHIVE https://github.com/FuzzyMonkeyCo/monkey/releases/download/$TAG/$ARCHIVE \
 && curl -fsSL -o checksums.sha256.txt https://github.com/FuzzyMonkeyCo/monkey/releases/download/$TAG/checksums.sha256.txt \
 && grep $ARCHIVE checksums.sha256.txt >only \
 && sha256sum -s -c only \
 && tar zxvf $ARCHIVE -C . \
 && rm $ARCHIVE
FROM scratch AS binaries-1
COPY --from=monkey-prebuilt /w/monkey* /

FROM binaries-$PREBUILT AS binaries


## Default target

FROM binaries
