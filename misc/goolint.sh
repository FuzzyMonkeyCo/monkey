#!/bin/sh -u

# goolint: farther than golint
# go fmt first

[ -f go.mod ] || exit 0

E() {
    code=$?
    [ $code -ne 0 ] && echo "  $*"
    errors=$(( errors + code ))
}

if ! command -v ag >/dev/null 2>&1; then
    echo Skipping: silver searcher unavailable
    exit 0
fi

g() {
    ag --ignore '*.pb.go' --ignore 'migrations/bindata.go' "$@"
}

errors=0

! g 'return\s+}\s+return\s+}'
E first return can be dropped

! g '^\s+fmt\.[^\n]+\s+log\.Print'
E log first

! g ', err = [^;\n]+\s+if err '
E if can be inlined

! g '^\s+err :?= [^;\n]+\s+if err '
E if can be inlined

! g '^\s+fmt\.Errorf'
E unused value

! g --ignore '*.pb.go' 'fmt\.Errorf."[^%\n]*"'
E prefer errors.New

! g '([^ ]+) != nil\s+{\s+for [^=]+ := range \1'
E unnecessary nil check around range

# ! g 'make\(\[\][^],]+, [^0]'
# E you meant capacity

! g '\.Println\((\"[^"]+ \")+'
E unnecessary trailing space

! g 'log.Printf\([^,]+\\n.\,'
E superfluous newline

! g '[^A-Za-z0-9_\]]byte\("\\?[^"]"\)|'"[^A-Za-z0-9_\\]]byte\\('\\\\?[^']'\\)"
E that\'s just single quotes with extra steps

exit $errors
