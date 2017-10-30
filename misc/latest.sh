#!/bin/sh -e

# Installs the latest version of https://github.com/CoveredCI/testman for your system

slug=CoveredCI/testman

target_path=/usr/local/bin
case :$PATH: in
    *:$target_path:*) ;;
    *)
        echo $target_path' is not in your $PATH'
        exit 2
        ;;
esac

echo Looking for latest tag of $slug
latest_tag_url=$(curl --silent --location --output /dev/null --write-out '%{url_effective}' https://github.com/$slug/releases/latest)
latest_tag=$(basename $latest_tag_url)
echo Latest tag: $latest_tag

exe=''
case $(uname) in
    *Linux) exe=testman-linux_amd64 ;;
    *Darwin) exe=testman-darwin_amd64 ;;
    *)
        echo "Unsupported architecture $(uname)"
        echo "Please open an issue at https://github.com/$slug/issues"
        exit 2
        ;;
esac

echo Downloading v$latest_tag of $exe
tmp=/tmp/testman
curl --silent --location --output $tmp https://github.com/$slug/releases/download/$latest_tag/$exe
chmod +x $tmp
mv --verbose $tmp $target_path/

testman --version
echo Successful installation!
