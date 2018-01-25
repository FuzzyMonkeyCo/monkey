#!/bin/bash -u

# goolint: farther than golint
# go fmt first

print() {
    printf '\e[1;1m%s\e[0m\n' "$*"
}
:() {
    code=$?
    [[ $code -ne 0 ]] && print "$*"
    ((errors+=code))
}

if ! which ag >/dev/null 2>&1; then
    print Skipping: silver searcher unavailable
    exit 0
fi

errors=0

! ag 'return\s+}\s+return\s+$'
: first return can be dropped

! ag '^\s+fmt\.[^\n]+\s+log\.Print'
: log first

! ag ', err = [^;\n]+\s+if err '
: if can be inlined

! ag '^\s+err :?= [^;\n]+\s+if err '
: if can be inlined

! ag '^\s+fmt.Errorf'
: unused value

exit $errors
