#!/bin/bash

set -e
set -x

# Source common stuff.
scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"

# Exit in case the gonative directory exists already.
if [ -d "$CACHE/gonative" ]; then
	echo "gonative installed already, skipping..."
	exit 0
fi

# ---> Print Go version

goVersion=$(go version | awk '{ print $3 }')
echo "GO VERSION: $goVersion"

# ---> Install gonative executable

GONATIVE_SRC="$WORKSPACE/src/github.com/inconshreveable/gonative"
git clone https://github.com/inconshreveable/gonative "$GONATIVE_SRC"
make -C "$GONATIVE_SRC"
GONATIVE_EXE="$GONATIVE_SRC/gonative"

# ---> Download Go 1.4.3 to bootstrap the compiler for gonative

pkgUrl='https://storage.googleapis.com/golang/go1.4.3.linux-amd64.tar.gz'
pkgPath="$HOME/go1.4.3.tag.gz"
pkgDst="$HOME/go1.4.3"
curl -o "$pkgPath" "$pkgUrl"
[ ! -d "$pkgDst" ] && mkdir -p "$pkgDst"
tar -C "$pkgDst" -xzf "$pkgPath"
export GOROOT_BOOTSTRAP="$pkgDst/go"

# ---> Build gonative
mkdir -p "$CACHE/gonative" && cd "$CACHE/gonative"
set +e

"$GONATIVE_EXE" build \
	-platforms="windows_amd64 darwin_amd64 linux_amd64" \
	-version="${goVersion#go}"

if [ "$?" -ne 0 ]; then
	rm -Rf "$CACHE/gonative"
	exit 1
fi
