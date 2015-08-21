#!/bin/bash

set -e
set -x

# Make sure we are running on CircleCI.
if  [ -z "$CIRCLECI" ]; then
	echo "This script can only be executed on CircleCI."
	exit 1
fi

PREFIX="$HOME/cache"

# Exit in case the gonative directory exists already.
if [ -d "$PREFIX/gonative" ]; then
	echo "gonative installed already, skipping..."
	exit 0
fi

# ---> Install the tools
go get github.com/inconshreveable/gonative

# ---> Clone Go sources
goVersion=$(go version | awk '{ print $3 }')

if [ ! -d "$PREFIX/go" ]; then
	[ ! -d "$PREFIX" ] && mkdir -p "$PREFIX"
	cd "$PREFIX" && git clone https://go.googlesource.com/go && cd go && git checkout "$goVersion"
else
	cd "$PREFIX/go" && git pull && git checkout "$goVersion"
fi

# ---> Build gonative
mkdir -p "$PREFIX/gonative" && cd "$PREFIX/gonative"
set +e
GOROOT="$PREFIX/go" gonative build -src="$PREFIX/go" -platforms="windows_amd64 darwin_amd64 linux_amd64" -version="${goVersion#go}"
if [ "$?" -ne 0 ]; then
	rm -Rf "$PREFIX/gonative"
	exit 1
fi
