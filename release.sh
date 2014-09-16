#!/bin/bash

# This script can be used to build the Go part of SalsaFlow and upload it
# to the specified GitHub release. It must be invoked from the top level
# directory and it is expecting the repo to be cloned inside of a Go
# workspace.

set -e

if [ "$#" -ne 1 ]; then
	echo "Usage: $1 VERSION"
	exit 2
fi

(cd git-trunk && make install)

version="$1"
pkg="salsaflow.$version.darwin-amd64-osx10.9"

trap cleanup EXIT

cleanup() {
	rm -R ${pkg}*
}

if [ ! -e "$pkg" ]; then
	mkdir "$pkg"
	cp ../../../../bin/git-trunk "$pkg/git-trunk"
	cp ../../../../bin/commit-msg "$pkg/git-trunk-hook-commit-msg"
fi

if [ ! -e "$pkg.zip" ]; then
	zip -r "$pkg.zip" "$pkg"
fi

ghr --username salsita --repository SalsaFlow --draft "v$version" "$pkg.zip"
