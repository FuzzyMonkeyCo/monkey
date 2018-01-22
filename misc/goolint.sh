#!/bin/bash -u

# goolint: farther than golint

print() {
    printf '\e[1;1m%s\e[0m\n' "$*"
}
nfo() {
    code=$?
    [[ $code -ne 0 ]] && print "$*"
    ((errors+=code))
}

if ! which ag >/dev/null 2>&1; then
    print Skipping: ag is unavailable
    exit 0
fi

errors=0

! ag 'return\s+}\s+return'
nfo first return can be dropped

! ag '^\s+fmt\.[^\n]+\n\s+log\.Print'
nfo 'log first then fmt'

! ag ', err = [^;\n]+$\n\s+if err != nil'
nfo if can be inlined

! ag '^\s+err\s*=[^\n]+\s+if err '
nfo if can be inlined

exit $errors
