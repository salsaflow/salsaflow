#!/bin/bash

set -e
set -x

#--- Make sure we are running on CircleCI

if  [ -z "$CIRCLECI" ]; then
	echo "This script can only be executed on CircleCI."
	exit 1
fi

#--- Set up the environment

export GOROOT="$HOME/cache/gonative/go"
export PATH="$HOME/cache/gonative/go/bin:$PATH"

#--- Install dependencies

go get github.com/tools/godep
go get github.com/mitchellh/gox

#--- Prepare the Go workspace and move the sources into it

workspace="$HOME/workspace"
mkdir -p "$workspace"

dst="$workspace/src/github.com/$CIRCLE_PROJECT_USERNAME"
mkdir -p "$dst"
ln -s "$HOME/$CIRCLE_PROJECT_REPONAME" "$dst/$CIRCLE_PROJECT_REPONAME"

export GOPATH="$workspace:$(godep path):$GOPATH"

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

echo "$pkgs" | xargs gox -osarch="windows/amd64 linux/amd64 darwin/amd64"
