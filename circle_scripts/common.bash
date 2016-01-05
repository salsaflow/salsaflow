function ensure_variable_set() {
	value="$(eval echo \${$1})"
	if [ -z "$value" ]; then
		echo "$variable environment variable is not set" 1>&2
		exit 1
	fi
}

function ensure_directory_exists() {
	path="$1"
	mkdir -p "$path"
	if [ "$?" -ne 0 ]; then
		echo "Failed to create directory $path" 1>&2
		exit 1
	fi
}

if [ -n "$CIRCLECI" ]; then
	PROJECT_USERNAME="$CIRCLE_PROJECT_USERNAME"
	PROJECT_REPONAME="$CIRCLE_PROJECT_REPONAME"
	SOURCES="$HOME/$CIRCLE_PROJECT_REPONAME"
	ROOT="$HOME/SalsaFlow_CC"
	GOLANG_GOOS="linux"
	GOLANG_GOARCH="amd64"
fi

for variable in PROJECT_USERNAME PROJECT_REPONAME SOURCES ROOT GOLANG_GOOS GOLANG_GOARCH; do
	ensure_variable_set "$variable"
done

WORKSPACE="$ROOT/workspace"
CACHE="$ROOT/cache"
GARBAGE="$ROOT/tmp"

for directory in "$WORKSPACE" "$CACHE" "$GARBAGE"; do
	ensure_directory_exists "$directory"
done

set -e
set -x
