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
for path in "$@" \
                'C:/Program Files/Git/usr/bin' ~/.local/bin \
                /usr/local/bin /usr/bin /bin
do
    case :"$PATH": in
        *:"$path":*)
            if ! mkdir -p "$path" >/dev/null 2>&1; then
                continue
            fi
            monkey_test="$path"/monkey
            if [ -f "$monkey_test" ] && [ -w "$monkey_test" ]; then
                target_path="$path"
                echo "Selected target path: $target_path"
                break
            fi
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
target="$target_path"/monkey

echo Looking for latest tag of $slug
latest_tag_url=$(curl -# --fail --location --output /dev/null --write-out '%{url_effective}' $page/releases/latest)
latest_tag=$(basename "$latest_tag_url")
latest_version=monkey/"$latest_tag"

if [ "$("$target" --version 2>&1 || true)" = "$latest_version" ]; then
    echo "Latest tag is already installed. You're good to go"
    exit 0
fi

exe=monkey-"$(uname -s)-$(uname -m)"
case "${exe##monkey-}" in
    CYGWIN*|MINGW32*|MSYS*) exe=monkey-Windows-"$(uname -m)".exe ;;
esac

echo "Downloading v$latest_tag of $exe"
tmp="$(mktemp)"
url="$page/releases/download/$latest_tag/$exe"
curl -# --fail --location --output "$tmp" "$url"
echo Verifying checksum...
sha="$tmp".sha256.txt
curl -# --fail --location --output "$sha" "$url".sha256.txt
tmpdir="$(dirname "$tmp")"
( cd "$tmpdir"
  mv "$tmp" "$exe"
  if   which sha256sum >/dev/null 2>&1; then
      sha256sum --check --strict "$sha"
  elif which shasum >/dev/null 2>&1; then
      shasum -a 256 -c "$sha"
  elif which openssl >/dev/null 2>&1; then
      sum="$(openssl sha256 "$exe" | cut -d= -f2)"
      grep -F "$sum" "$sha" >/dev/null 2>&1
  else
      echo Found no way of verifying checksum. Skipping!
  fi
  rm "$sha"
  chmod +x "$exe"
)
mv -v "$tmpdir/$exe" "$target"

installed_version=$("$target" --version)
if [ "$installed_version" != "$latest_version" ]; then
    fatal "This is not the expected version: $installed_version"
fi

echo Successful installation!
echo Note: make sure "$target_path" is in your PATH
