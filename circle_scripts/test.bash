#!/bin/bash

set -e
set -x

if  [ -z "$CIRCLECI" ]; then
	echo "This script can only be executed on CircleCI."
	exit 1
fi

export GOPATH="$HOME/workspace:$(godep path):$GOPATH"

make test
