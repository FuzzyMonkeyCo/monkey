#!/bin/sh

set -o errexit
set -o nounset

# Installs the latest version of https://github.com/FuzzyMonkeyCo/monkey for your system

slug=FuzzyMonkeyCo/monkey
page=https://github.com/$slug

fatal() {
    echo "$@"
    echo "Please report this at $page/issues or ook@fuzzymonkey.co"
    exit 2
}

# Note: ~/.local/bin is for TravisCI.com
# Note: C:\Program Files\Git\usr\bin is for appveyor.com
target_path=
for path in "$@" /usr/local/bin /usr/bin ~/.local/bin 'C:\Program Files\Git\usr\bin'; do
    case :"$path": in
        *:"$path":*)
            mkdir -p "$path" >/dev/null 2>&1 || true
            if touch "$path"/monkey >/dev/null 2>&1; then
                rm   "$path"/monkey >/dev/null 2>&1
                target_path="$path"
                echo "Selected target path: $target_path"
                break
            fi;;
    esac
done
if [ -z "$target_path" ]; then
    fatal "Could not find a suitable target path among $PATH"
fi

echo Looking for latest tag of $slug
latest_tag_url=$(curl --silent --location --output /dev/null --write-out '%{url_effective}' $page/releases/latest)
latest_tag=$(basename "$latest_tag_url")
latest_version=monkey/"$latest_tag"
echo Latest tag: "$latest_tag"

if [ "$(monkey --version 2>&1 || true)" = "$latest_version" ]; then
    echo "Latest tag is already installed. You're good to go"
    exit 0
fi

exe=monkey-"$(uname -s)-$(uname -m)"
case "$exe" in
    CYGWIN*|MINGW32*|MSYS*) exe=$exe.exe ;;
esac

echo "Downloading $exe v$latest_tag"
tmp="$(mktemp)"
curl -# --location --output "$tmp".sha256s.txt "$page/releases/download/$latest_tag/sha256s.txt"
curl -# --location --output "$tmp"             "$page/releases/download/$latest_tag/$exe"
tmpdir="$(dirname "$tmp")"
( cd "$tmpdir"
  mv "$tmp" "$exe"
  sha256sum --check --strict --ignore-missing "$tmp".sha256s.txt
  rm "$tmp".sha256s.txt
  chmod +x "$exe"
)
mv -v "$tmpdir/$exe" "$target_path"/monkey

installed_version=$("$target_path"/monkey --version)
if [ "$installed_version" != "$latest_version" ]; then
    fatal "This is not the expected version: $installed_version"
fi

echo Successful installation!
echo Note: make sure "$target_path" is in your PATH
