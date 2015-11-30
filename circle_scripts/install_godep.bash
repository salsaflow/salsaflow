#!/bin/bash

set -e
set -x

#--- Make sure we are running on CircleCI

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"
source "$scripts/common_gvm"

#--- Install godep

go get github.com/tools/godep
