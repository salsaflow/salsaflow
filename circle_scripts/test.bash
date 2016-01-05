#!/bin/bash

#--- Source common variables

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common.bash"

#--- Run tests

export GOPATH="$WORKSPACE:$(godep path)"
make test
