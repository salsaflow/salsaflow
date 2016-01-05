#!/bin/bash

#--- Source common variables

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common.bash"

#--- Install dependencies

go get github.com/tools/godep
go get github.com/mitchellh/gox

#--- Set up the environment

gonativeCache="$CACHE/gonative"

export PATH="$gonativeCache/go/bin:$PATH"

export GOROOT="$gonativeCache/go"
export GOPATH="$WORKSPACE:$(godep path)"

#--- Prepare the Go workspace and move the sources into it

base="$WORKSPACE/src/github.com/$PROJECT_USERNAME"
workspaceSources="$base/$PROJECT_REPONAME"

mkdir -p "$base" && ln -s "$SOURCES" "$workspaceSources"

#--- Build the project

cd "$workspaceSources"

pkgs="$(cat <<-EOF
github.com/salsaflow/salsaflow
github.com/salsaflow/salsaflow/bin/hooks/salsaflow-commit-msg
github.com/salsaflow/salsaflow/bin/hooks/salsaflow-post-checkout
github.com/salsaflow/salsaflow/bin/hooks/salsaflow-pre-push
EOF
)"

echo "$pkgs" | xargs gox -osarch="windows/amd64 linux/amd64 darwin/amd64"
