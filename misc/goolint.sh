#!/bin/sh -u

# goolint: farther than golint
# go fmt first

E() {
    code=$?
    [ $code -ne 0 ] && echo "  $*"
    errors=$(( errors + code ))
}

if ! command -v ag >/dev/null 2>&1; then
    echo Skipping: silver searcher unavailable
    exit 0
fi

errors=0

! ag 'return\s+}\s+return\s+}'
E first return can be dropped

! ag '^\s+fmt\.[^\n]+\s+log\.Print'
E log first

! ag ', err = [^;\n]+\s+if err '
E if can be inlined

! ag '^\s+err :?= [^;\n]+\s+if err '
E if can be inlined

! ag '^\s+fmt\.Errorf'
E unused value

! ag --ignore '*.pb.go' 'fmt\.Errorf."[^%\n]*"'
E prefer errors.New

! ag '([^ ]+) != nil\s+{\s+for [^=]+ := range \1'
E unnecessary nil check around range

exit $errors
