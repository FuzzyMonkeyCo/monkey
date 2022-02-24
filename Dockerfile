# syntax=docker.io/docker/dockerfile:1@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2

# Use --build-arg PREBUILT=1 with default target to fetch binaries from GitHub releases
ARG PREBUILT

FROM --platform=$BUILDPLATFORM docker.io/library/alpine@sha256:21a3deaa0d32a8057914f36584b5288d2e5ecc984380bc0118285c70fa8c9300 AS alpine
FROM --platform=$BUILDPLATFORM docker.io/nilslice/protolock@sha256:baf9bca8b7a28b945c557f36d562a34cf7ca85a63f6ba8cdadbe333e12ccea51 AS protolock
FROM --platform=$BUILDPLATFORM docker.io/library/golang@sha256:e06c83493ef6d69c95018da90f2887bf337470db074d3c648b8b648d8e3c441e AS golang
FROM --platform=$BUILDPLATFORM docker.io/goreleaser/goreleaser@sha256:202577e3d05c717171c79be926e7b8ba97aac4c7c0bb3fc0fe5a112508b2651c AS goreleaser
# On this image:
#  go env GOCACHE    => /root/.cache/go-build
#  go env GOMODCACHE => /go/pkg/mod


FROM goreleaser AS base
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
 && git --no-pager diff --exit-code
COPY . .


## CI checks

FROM base AS ci-check--lint
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && make lint \
 && git --no-pager diff --exit-code

FROM base AS ci-check--mod
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && go mod tidy \
 && go mod verify \
 && git --no-pager diff --exit-code

FROM base AS ci-check--test
ENV TESTPWDID=1
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && go test ./... \
 && go test -count 10 ./... \
 && git --no-pager diff --exit-code

FROM alpine AS ci-check--protolock-
WORKDIR /app
RUN \
  --mount=type=cache,target=/var/cache/apk ln -vs /var/cache/apk /etc/apk/cache && \
    set -ux \
 && apk add git
COPY pkg/internal/fm/proto.lock .
COPY pkg/internal/fm/*.proto .
ARG FORCE
RUN \
  --mount=from=protolock,source=/usr/bin/protolock,target=/usr/bin/protolock \
    set -ux \
 && if [ -n "${FORCE:-}" ]; then \
      /usr/bin/protolock commit --force && exit ; \
    fi \
 && git init \
 && git add -A . \
 && /usr/bin/protolock commit \
 && git --no-pager diff --exit-code
FROM scratch AS ci-check--protolock
COPY --from=ci-check--protolock- /app/proto.lock /

FROM golang AS ci-check--protoc-
WORKDIR /app
ENV GOBIN /go/bin
# https://github.com/moby/buildkit/blob/a1cfefeaeb66501a95a4d2f5858c939211f331ac/frontend/dockerfile/docs/syntax.md#example-cache-apt-packages
RUN rm -f /etc/apt/apt.conf.d/docker-clean; echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN \
  --mount=type=cache,target=/var/cache/apt --mount=type=cache,target=/var/lib/apt \
    set -ux \
 && apt update \
 && apt-get --no-install-recommends install -y protobuf-compiler
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/root/.cache/go-build \
    set -ux \
 && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1 \
 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest \
 && go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.2.0
ADD https://raw.githubusercontent.com/protocolbuffers/protobuf/2f91da585e96a7efe43505f714f03c7716a94ecb/src/google/protobuf/struct.proto /wellknown/google/protobuf/struct.proto
COPY pkg/internal/fm/*.proto .
RUN \
  --mount=type=cache,target=/var/cache/apt --mount=type=cache,target=/var/lib/apt \
    set -ux \
 && protoc \
      -I . \
      -I /wellknown \
      --go_out=.         --plugin protoc-gen-go="$GOBIN"/protoc-gen-go \
      --go-grpc_out=.    --plugin protoc-gen-go-grpc="$GOBIN"/protoc-gen-go-grpc \
      --go-vtproto_out=. --plugin protoc-gen-go-vtproto="$GOBIN"/protoc-gen-go-vtproto \
      --go-vtproto_opt=features=marshal+unmarshal+size \
      *.proto
FROM scratch AS ci-check--protoc
COPY --from=ci-check--protoc- /app/github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm/*.pb.go /

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

FROM alpine AS archmap-darwin-amd64-
RUN        echo monkey-Darwin-x86_64.tar.gz >/archmap
FROM alpine AS archmap-linux-386-
RUN        echo monkey-Linux-i386.tar.gz    >/archmap
FROM alpine AS archmap-linux-amd64-
RUN        echo monkey-Linux-x86_64.tar.gz  >/archmap
FROM alpine AS archmap-windows-386-
RUN        echo monkey-Windows-i386.zip     >/archmap
FROM alpine AS archmap-windows-amd64-
RUN        echo monkey-Windows-x86_64.zip   >/archmap

FROM archmap-$TARGETOS-$TARGETARCH-$TARGETVARIANT AS archmap


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
