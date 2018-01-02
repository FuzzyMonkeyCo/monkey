#!/bin/sh

set -o errexit
set -o nounset

# Installs the latest version of https://github.com/CoveredCI/testman for your system

slug=CoveredCI/testman

fatal() {
    echo "$@"
    echo "Please report this at https://github.com/$slug/issues or hi@coveredci.co"
    exit 2
}

# Note: ~/.local/bin is for TravisCI.com
target_path=
for path in /usr/local/bin /usr/bin ~/.local/bin nil; do
    case :$PATH: in
        :nil:) fatal "Could not find a suitable target path among $PATH" ;;
        *:$path:*)
            mkdir -p $path >/dev/null 2>&1 || true
            if touch $path/testman >/dev/null 2>&1; then
                target_path=$path
                echo "Selected target path: $target_path"
                break
            else
                rm $path/testman >/dev/null 2>&1 || true
            fi;;
        *) ;;
    esac
done

echo Looking for latest tag of $slug
latest_tag_url=$(curl --silent --location --output /dev/null --write-out '%{url_effective}' https://github.com/$slug/releases/latest)
latest_tag=$(basename "$latest_tag_url")
echo "Latest tag: $latest_tag"

exe="testman-$(uname -s)-$(uname -m)"
case "$exe" in
    CYGWIN*|MINGW32*|MSYS*) exe="$exe".exe ;;
esac

echo "Downloading $exe v$latest_tag"
tmp="$(mktemp)"
curl -# --location --output "$tmp" "https://github.com/$slug/releases/download/$latest_tag/$exe"
chmod +x "$tmp"
mv "$tmp" $target_path/testman

if ! which testman >/dev/null 2>&1; then
    fatal "$exe does not appear to be in $target_path"
fi
if ! testman --version | grep "$latest_tag" >/dev/null 2>&1; then
    fatal "This is not the expected version: $(testman --version || true)"
fi
echo Successful installation!
