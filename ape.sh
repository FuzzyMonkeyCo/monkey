#!/bin/bash -eu

set -o pipefail

dir="$(dirname "$0")"
err=/tmp/.monkey_$RANDOM.code

i=0
if ls "$dir"/*.cov >/dev/null 2>&1; then
    biggest=$(find "$dir" -name '*.cov' | sort -Vr | head -n1)
    i=$(basename "$biggest" .cov)
    ((i+=1))
fi

MONKEY_CODEFILE=$err MONKEY_ARGS="$*" "$dir"/monkey.test \
               -test.coverprofile="$dir"/$i.cov \
               -test.run=^TestCov$

code=$(cat $err)
rm $err
# shellcheck disable=SC2086
exit $code
