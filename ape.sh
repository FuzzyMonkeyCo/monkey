#!/bin/bash -eu

i=0
if ls *.cov >/dev/null 2>&1; then
    biggest=$(ls *.cov | sort -Vr | head -n1)
    i=$(basename $biggest .cov)
    ((i+=1))
fi

dir="$(dirname "$0")"
err=/tmp/.monkey_$RANDOM.code

MONKEY_CODEFILE=$err MONKEY_ARGS="$@" "$dir"/monkey.test \
           -test.coverprofile="$dir"/$i.cov \
           -test.run=^TestCov$

code=$(cat $err)
rm $err
exit $code
