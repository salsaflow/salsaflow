#!/bin/bash

set -e
set -x

# Source common stuff.
scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"
source "$scripts/common_gvm"

PREFIX="$HOME/cache"

# Exit in case the gonative directory exists already.
if [ -d "$PREFIX/gonative" ]; then
	echo "gonative installed already, skipping..."
	exit 0
fi

# ---> Print Go version

goVersion=$(go version | awk '{ print $3 }')
echo "GO VERSION: $goVersion"

# ---> Install gonative

GONATIVE_SRC="$WORKSPACE/src/github.com/inconshreveable/gonative"

# Clone the source.
git clone https://github.com/inconshreveable/gonative "$GONATIVE_SRC"
cd "$GONATIVE_SRC"

# Restore the dependencies. We try 3 times since it somehow fails occasionally.
ok=0
set +e
for i in $(seq 3); do
	echo "TRYING TO RESTORE GONATIVE DEPS (RUN $i)"
	GOPATH="$GONATIVE_SRC/Godeps/_workspace" godep restore -v
	if [ "$?" -eq 0 ]; then
		ok=1
		break
	fi
done
if [ $ok -ne 1 ]; then
	echo "FAILED TO RESTORE GONATIVE DEPS" 2>&1
	exit 1
fi
set -e

# Install gonative, at last.
GOPATH="$WORKSPACE" godep go install
export PATH="$WORKSPACE/bin:$PATH"

# ---> Clone Go sources

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
