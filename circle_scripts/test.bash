#!/bin/bash

set -e
set -x

# Source common stuff
scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"

# Run tests
export GOPATH="$HOME/workspace:$(godep path):$GOPATH"
make test
