#!/bin/bash

set -e
set -x

#--- Make sure we are running on CircleCI

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"

#--- Set up the environment

export GOROOT="$CACHE/gonative/go"
export PATH="$CACHE/gonative/go/bin:$PATH"

#--- Install dependencies

go get github.com/mitchellh/gox

#--- Prepare the Go workspace and move the sources into it

dst="$WORKSPACE/src/github.com/$CIRCLE_PROJECT_USERNAME"
mkdir -p "$dst"
ln -s "$HOME/$CIRCLE_PROJECT_REPONAME" "$dst/$CIRCLE_PROJECT_REPONAME"

export GOPATH="$WORKSPACE:$(godep path):$GOPATH"

#--- Build the project

sources="$dst/$CIRCLE_PROJECT_REPONAME"
cd "$sources"

pkgs="$(cat <<-EOF
github.com/salsaflow/salsaflow
github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
github.com/salsaflow/salsaflow/bin/hooks/salsaflow-post-checkout
github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push
EOF
)"

echo "$pkgs" | xargs gox -verbose -osarch="windows/amd64 linux/amd64 darwin/amd64"
