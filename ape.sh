#!/bin/bash -eu

i=0
if ls *.cov >/dev/null 2>&1; then
    biggest=$(ls *.cov | sort -Vr | head -n1)
    i=$(basename $biggest .cov)
    ((i+=1))
fi

MONKEY_ARGS="$@" ./monkey.test -test.coverprofile=$i.cov -test.run=^TestCov$
