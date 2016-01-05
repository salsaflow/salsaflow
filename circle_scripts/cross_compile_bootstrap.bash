#!/bin/bash

#--- Source common variables

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common.bash"

#--- Exit in case gonative cache exists already

gonativeCache="$CACHE/gonative"

if [ -d "$gonativeCache" ]; then
	echo "gonative cached, skipping..."
	exit 0
fi

#--- Print Go version

#goVersion=$(go version | awk '{ print $3 }')
#goVersion=${goVersion#go}
goVersion='1.4.3'

echo "Go version: $goVersion"

#--- Install gonative executable

gonativeSrc="$WORKSPACE/src/github.com/inconshreveable/gonative"
git clone 'https://github.com/inconshreveable/gonative' "$gonativeSrc"
make -C "$gonativeSrc"

gonativeExe="$gonativeSrc/gonative"

#--- Download Go 1.4.3 to bootstrap the compiler for gonative

#pkgUrl="https://storage.googleapis.com/golang/go1.4.3.${GOLANG_GOOS}-${GOLANG_GOARCH}.tar.gz"
#pkgPath="$GARBAGE/go1.4.3.tag.gz"
#pkgDst="$GARBAGE/go1.4.3"
#
#curl -o "$pkgPath" "$pkgUrl"
#
#[ ! -d "$pkgDst" ] && mkdir -p "$pkgDst"
#tar -C "$pkgDst" -xzf "$pkgPath"
#
#export GOROOT_BOOTSTRAP="$pkgDst/go"

#--- Build gonative

mkdir -p "$gonativeCache" && cd "$gonativeCache"
set +e

"$gonativeExe" build \
	-platforms="windows_amd64 darwin_amd64 linux_amd64" \
	-version="$goVersion"

if [ "$?" -ne 0 ]; then
	rm -Rf "$gonativeCache"
	exit 1
fi
