#!/bin/sh -eu

default_tag=0.0.0
CURRENT_TAG=${TRAVIS_TAG:-$default_tag}
echo "$CURRENT_TAG" | grep -E '^[0-9]+\.[0-9]+.[0-9]+$' >/dev/null
GIT_DESCRIBE=$(git describe --abbrev --dirty --always --tags)
out=meta.go

echo package main >$out

if [ "$CURRENT_TAG" = $default_tag ]; then
    { printf 'const wsURL = "%s"\n' 'ws://api.dev.fuzzymonkey.co:7077/1/fuzz'
    } >>$out
else
    { printf 'const wsURL = "%s"\n' 'wss://api.fuzzymonkey.co:7077/1/fuzz'
    } >>$out
fi

{ printf 'const binVersion = "%s"\n' "$CURRENT_TAG"
  printf 'const binDescribe = "%s"' "$GIT_DESCRIBE"
} >>$out

gofmt -w $out
