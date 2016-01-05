#!/bin/bash

#--- Make sure we are on Circle CI

if [ -z "$CIRCLECI" ]; then
	echo "This script can only be run on Circle CI" 1>&2
	exit 1
fi

#--- Source common variables

scripts="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$scripts/common.bash"

#--- Pack

src="$SOURCES"
dst="$GARBAGE/build/dist"

salsaflowVersion="$(echo -n `"$src/salsaflow_linux_amd64" -version`)"
version="$salsaflowVersion+circleci$CIRCLE_BUILD_NUM"

for os in linux windows darwin; do
	if [ "$os" == "windows" ]; then
		exe_suffix=".exe"
	else
		exe_suffix=""
	fi
	os_suffix="${os}_amd64"

	base="salsaflow-$version-${os_suffix}"
	mkdir -p "$dst/$base"

	cp "$src/salsaflow_${os_suffix}${exe_suffix}" \
	   "$dst/$base/salsaflow${exe_suffix}"

	for hook in commit-msg pre-push post-checkout; do
		cp "$src/salsaflow-${hook}_${os_suffix}${exe_suffix}" \
		   "$dst/$base/salsaflow-${hook}${exe_suffix}"
	done

	(cd "$dst" && zip -r "${base}.zip" "$base/" && cp "${base}.zip" "$CIRCLE_ARTIFACTS/")
done
