#!/bin/sh -eu

CURRENT_TAG=${TRAVIS_TAG:-0.0.0}
echo $CURRENT_TAG | grep -E '^[0-9]+\.[0-9]+.[0-9]+$' >/dev/null
GIT_DESCRIBE=$(git describe --abbrev --dirty --always --tags)
out=meta.go

echo package main >$out
printf 'const binVersion = "%s"\n' $CURRENT_TAG >>$out
printf 'const binDescribe = "%s"' $GIT_DESCRIBE >>$out
gofmt -w $out
