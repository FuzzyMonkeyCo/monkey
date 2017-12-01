#!/bin/sh

set -o errexit
set -o nounset

# Installs the latest version of https://github.com/CoveredCI/testman for your system

slug=CoveredCI/testman

target_path=/usr/local/bin
case :$PATH: in
    *:$target_path:*) ;;
    *)
        echo $target_path' is not in your PATH'
        exit 2
        ;;
esac

echo Looking for latest tag of $slug
latest_tag_url=$(curl --silent --location --output /dev/null --write-out '%{url_effective}' https://github.com/$slug/releases/latest)
latest_tag=$(basename "$latest_tag_url")
echo "Latest tag: $latest_tag"

fatal() {
    echo "$@"
    echo "Please report this at https://github.com/$slug/issues or hi@coveredci.co"
    exit 2
}

exe=
case "$(uname -s)" in  # https://stackoverflow.com/a/27776822/1418165
    Linux)  exe=testman-linux_amd64 ;;
    Darwin) exe=testman-darwin_amd64 ;;
    CYGWIN*|MINGW32*|MSYS*) exe=testman-windows_amd64.exe ;;
    *) fatal "Unsupported architecture '$(uname -s)': $(uname -a)" ;;
esac

echo "Downloading $exe v$latest_tag"
tmp="$(mktemp)"
curl -# --location --output "$tmp" "https://github.com/$slug/releases/download/$latest_tag/$exe"
chmod +x "$tmp"
mv --verbose "$tmp" $target_path/testman

if ! which testman >/dev/null 2>&1; then
    fatal "$exe does not appear to be in $target_path"
fi
if ! testman --version | grep "$latest_tag" >/dev/null 2>&1; then
    fatal "This is not the expected version: $(testman --version || true)"
fi
echo Successful installation!
