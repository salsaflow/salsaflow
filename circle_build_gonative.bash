#!/bin/bash

set -e
set -x

PREFIX="$HOME/cache"

# Exit in case the gonative directory exists already.
if [ -d "$PREFIX/gonative" ]; then
	echo "gonative installed already, skipping..."
	exit 0
fi

# ---> Install the tools
go get github.com/inconshreveable/gonative

# ---> Clone Go sources
if [ ! -d "$PREFIX/go" ]; then
	[ ! -d "$PREFIX" ] && mkdir -p "$PREFIX"
	cd "$PREFIX" && hg clone -u release https://code.google.com/p/go
else
	cd "$PREFIX/go" && hg pull -u
fi

# ---> Build gonative
mkdir -p "$PREFIX/gonative" && cd "$PREFIX/gonative"
set +e
GOROOT="$PREFIX/go" gonative -src="$PREFIX/go" -platforms="windows_amd64 darwin_amd64 linux_amd64" -version="1.3.3"
if [ "$?" -ne 0 ]; then
	rm -R "$PREFIX/gonative"
	exit 1
fi
