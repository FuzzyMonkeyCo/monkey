#!/bin/bash -eu

# Generate README.md

set -o errtrace
set -o pipefail

[[ 1 -ne $(git status --porcelain -- README.md | grep -Ec '^.[^ ]') ]] || exit 0

beg_usage=$(grep -n '```' -- README.md | head -n1 | cut -d: -f1)
end_usage=$(grep -n '```' -- README.md | head -n2 | tail -n1 | cut -d: -f1)
cat <(head -n "$beg_usage" README.md) <(./monkey -h) <(tail -n +"$end_usage" README.md) >_ && mv _ README.md

beg_example=$(grep -n '```' -- README.md | tail -n2 | head -n1 | cut -d: -f1)
end_example=$(grep -n '```' -- README.md | tail -n2 | tail -n1 | cut -d: -f1)
cat <(head -n "$beg_example" README.md) <(cat fuzzymonkey.star) <(tail -n +"$end_example" README.md) >_ && mv _ README.md
