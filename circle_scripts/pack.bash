#!/bin/bash

set -e
set -x

# Source common stuff
scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common"

# Pack
SRC="$HOME/$CIRCLE_PROJECT_REPONAME"
DST="$HOME/build/dist"

SALSAFLOW_VERSION="$(echo -n `"$SRC/salsaflow_linux_amd64" -version`)"
VERSION="$SALSAFLOW_VERSION+circleci$CIRCLE_BUILD_NUM"

for os in linux windows darwin; do
	if [ "$os" == "windows" ]; then
		exe_suffix=".exe"
	else
		exe_suffix=""
	fi
	os_suffix="${os}_amd64"

	base="salsaflow-$VERSION-${os_suffix}"
	mkdir -p "$DST/$base"

	cp "$SRC/salsaflow_${os_suffix}${exe_suffix}" \
	   "$DST/$base/salsaflow${exe_suffix}"

	for hook in commit-msg pre-push post-checkout; do
		cp "$SRC/salsaflow-${hook}_${os_suffix}${exe_suffix}" \
		   "$DST/$base/salsaflow-${hook}${exe_suffix}"
	done

	(cd "$DST" && zip -r "${base}.zip" "$base/" && cp "${base}.zip" "$CIRCLE_ARTIFACTS/")
done
