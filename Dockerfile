# syntax=docker.io/docker/dockerfile:1.2@sha256:e2a8561e419ab1ba6b2fe6cbdf49fd92b95912df1cf7d313c3e2230a333fdbcc
# locked docker/dockerfile:1.2 ^ @ 2021/03/14 on linux/amd64

# Use --build-arg PREBUILT=1 with default target to fetch binaries from GitHub releases
ARG PREBUILT

# locked goreleaser/goreleaser:latest @ 2021/03/14 on linux/amd64
FROM --platform=$BUILDPLATFORM docker.io/goreleaser/goreleaser@sha256:fa75344740e66e5bb55ad46426eb8e6c8dedbd3dcfa15ec1c41897b143214ae2 AS go-releaser

FROM go-releaser AS base
WORKDIR /w
ENV CGO_ENABLED=0
COPY go.??? .
RUN \
  --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/var/cache/apk ln -vs /var/cache/apk /etc/apk/cache && \
    set -ux \
 && apk add the_silver_searcher \
 && ag --version \
 && H=$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum) \
 && go mod download \
 && [[ "$H" = "$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum)" ]]
COPY . .


## CI checks

FROM base AS ci-check--lint
RUN \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
 && H=$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum) \
 && make lint \
 && [[ "$H" = "$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum)" ]]

FROM base AS ci-check--mod
RUN \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
 && H=$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum) \
 && go mod tidy \
 && go mod verify \
 && [[ "$H" = "$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum)" ]]

FROM base AS ci-check--test
RUN \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
 && H=$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum) \
 && go test -tags fakefs -count 10 ./... \
 && [[ "$H" = "$(find -type f -not -path './.git/*' | sort | tar cf - -T- | sha256sum)" ]]


## Build all platforms/OS

FROM base AS monkey-build
RUN \
  --mount=type=cache,target=/go/pkg/mod \
    set -ux \
 && grep -F . Tagfile \
 && CURRENT_TAG=$(cat Tagfile) goreleaser release --snapshot --parallelism=$(nproc)


## Goreleaser's dist/ for GitHub release

FROM scratch AS goreleaser-dist
COPY --from=monkey-build /w/dist/checksums.sha256.txt /
COPY --from=monkey-build /w/dist/monkey-*.tar.gz /
COPY --from=monkey-build /w/dist/monkey-*.zip /


## Binaries for each OS

FROM monkey-build AS monkey-build-darwin
RUN set -ux \
 && tar zxvf ./dist/monkey-Darwin-x86_64.tar.gz -C .
FROM scratch AS binaries-darwin
COPY --from=monkey-build-darwin /w/monkey /

FROM monkey-build AS monkey-build-linux
RUN set -ux \
 && tar zxvf ./dist/monkey-Linux-x86_64.tar.gz -C .
FROM scratch AS binaries-linux
COPY --from=monkey-build-linux /w/monkey /

FROM monkey-build AS monkey-build-windows
RUN set -ux \
 && tar zxvf ./dist/monkey-Windows-x86_64.tar.gz -C .
FROM scratch AS binaries-windows
COPY --from=monkey-build-windows /w/monkey.exe /

FROM alpine AS monkey-prebuilt
WORKDIR /w
RUN \
  --mount=type=cache,target=/var/cache/apk ln -vs /var/cache/apk /etc/apk/cache && \
    set -ux \
 && apk update \
 && apk add curl ca-certificates
COPY Tagfile .
ARG TARGETOS
ARG TARGETARCH
RUN set -ux \
 && TAG=$(cat Tagfile) \
 && ARCHIVE=UNHANDLED \
 && if [ $TARGETOS = darwin  ] && [ $TARGETARCH = amd64 ]; then ARCHIVE=monkey-Darwin-x86_64.tar.gz; fi \
 && if [ $TARGETOS = linux   ] && [ $TARGETARCH = 386 ]; then ARCHIVE=monkey-Linux-i386.tar.gz; fi \
 && if [ $TARGETOS = linux   ] && [ $TARGETARCH = amd64 ]; then ARCHIVE=monkey-Linux-x86_64.tar.gz; fi \
 && if [ $TARGETOS = windows ] && [ $TARGETARCH = 386 ]; then ARCHIVE=monkey-Windows-i386.zip; fi \
 && if [ $TARGETOS = windows ] && [ $TARGETARCH = amd64 ]; then ARCHIVE=monkey-Windows-x86_64.zip; fi \
 && curl -fsSL -o $ARCHIVE https://github.com/FuzzyMonkeyCo/monkey/releases/download/$TAG/$ARCHIVE \
 && curl -fsSL -o checksums.sha256.txt https://github.com/FuzzyMonkeyCo/monkey/releases/download/$TAG/checksums.sha256.txt \
 && grep $ARCHIVE checksums.sha256.txt >only \
 && sha256sum -s -c only \
 && tar zxvf $ARCHIVE -C .
FROM scratch AS binaries-darwin1
COPY --from=monkey-prebuilt /w/monkey /
FROM scratch AS binaries-linux1
COPY --from=monkey-prebuilt /w/monkey /
FROM scratch AS binaries-windows1
COPY --from=monkey-prebuilt /w/monkey.exe /

FROM binaries-${TARGETOS}${PREBUILT} AS binaries


## Default target

FROM binaries
