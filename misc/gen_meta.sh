#!/bin/sh -eu

default_tag=0.0.0
CURRENT_TAG=${TRAVIS_TAG:-$default_tag}
echo "$CURRENT_TAG" | grep -E '^[0-9]+\.[0-9]+.[0-9]+$' >/dev/null
GIT_DESCRIBE=$(git describe --abbrev --dirty --always --tags)
out=meta.go

echo package main >$out

if [ "$CURRENT_TAG" = $default_tag ]; then
    printf 'const apiFuzzNew  = "%s"\n' 'http://fuzz.dev.fuzzymonkey.co/1/new' >>$out
    printf 'const apiFuzzNext = "%s"\n' 'http://fuzz.dev.fuzzymonkey.co/1/next' >>$out
    printf 'const apiAuthURL  = "%s"\n' 'http://hoth.dev.fuzzymonkey.co/1/token' >>$out
else
	# FIXME: use HTTPS
    printf 'const apiFuzzNew  = "%s"\n' 'http://fuzz.fuzzymonkey.co/1/new' >>$out
    printf 'const apiFuzzNext = "%s"\n' 'http://fuzz.fuzzymonkey.co/1/next' >>$out
    printf 'const apiAuthURL  = "%s"\n' 'http://hoth.fuzzymonkey.co/1/token' >>$out
fi

printf 'const binVersion = "%s"\n' "$CURRENT_TAG" >>$out
printf 'const binDescribe = "%s"' "$GIT_DESCRIBE" >>$out

gofmt -w $out
