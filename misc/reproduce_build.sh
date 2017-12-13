#!/bin/sh -eu

GOVSN=${GOVSN:-$(go version | awk '{print $3}' | sed 's/go//')}
workdir=/go/src/github.com/FuzzyMonkeyCo/monkey

# cat <<EOF

make x
docker run --rm --interactive \
       --volume "$PWD":$workdir \
       --workdir $workdir \
       golang:"$GOVSN"-alpine3.7 \
       /bin/sh -eux -s <<EOF

# Before install
apk update
apk upgrade
apk add git curl make

# Install
make deps

# Build
make clean && rm -r vendor/
DST=repro make x
chown -R $(id -u):$(id -g) .

# Check
sha256sum -cw repro/*.sha256.txt

EOF
