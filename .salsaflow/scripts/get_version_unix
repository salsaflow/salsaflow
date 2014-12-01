#!/bin/bash

# Try to get the version string.
version="$(grep 'const Version' app/metadata/version.go | cut -d' ' -f4 | tr -d '"')"

# Make sure the string is not empty.
[ -z "$version" ] && exit 1

# Print the version string to stdout.
echo $version
