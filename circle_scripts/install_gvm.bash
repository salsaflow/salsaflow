#!/bin/bash

set -x

#--- Make sure we are running on CircleCI

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"
source "$scripts/common_gvm"

#--- Install gvm

gvm version >/dev/null
code="$?"
set -e
if [ "$code" -ne 0 ]; then
	bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)
fi

#--- Install and use go 1.4.3

set +e
source "$scripts/common_gvm"
set -e

gvm install "$GO"
