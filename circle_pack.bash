#!/bin/bash

set -e
set -x

SOURCES="$WORKSPACE/src/github.com/salsita/salsaflow"
DST="$WORKSPACE/bin/dist"

SALSAFLOW_VERSION="$(echo -n `"$SOURCES/salsaflow_linux_amd64" version`)"
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

	cp "$SOURCES/salsaflow_${os_suffix}${exe_suffix}" \
	   "$DST/$base/salsaflow${exe_suffix}"

	for hook in commit-msg pre-push; do
		cp "$SOURCES/bin/hooks/salsaflow-${hook}/salsaflow-${hook}_${os_suffix}${exe_suffix}" \
		   "$DST/$base/salsaflow-${hook}${exe_suffix}"
	done

	(cd "$DST" && zip -r "${base}.zip" "$base/" && cp "${base}.zip" "$CIRCLE_ARTIFACTS/")
done
