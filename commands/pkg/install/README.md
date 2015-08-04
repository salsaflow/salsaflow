# `pkg install` #

Install the specified SalsaFlow release.

## Usage ##

```
pkg install [-github_owner=OWNER]
            [-github_repo=REPO]
            [-dst=DST]
            <version>
```

## Description ##

This command can be used to download and install the specified SalsaFlow release.
The pre-built packages are fetched from GitHub. They are expected to be appended
as release assets to the GitHub release specified by the given version.

The repository that the assets are fetched from can be specified using
the available command line flags. By default it is `salsaflow/salsaflow`.

By default the current executables of SalsaFlow are replaced by
the executables being installed, but -dst can be used to specify a custom
target directory that the downloaded executables are moved to.

## Release Assets ##

To make a GitHub release compatible with `pkg`, it is necessary to
append a few zip archives to the release. These archives are expected to
contain the pre-built binaries of SalsaFlow.

The binaries that are to be packed into the archive can be found in the
`bin` directory of your Go workspace after running `make`.

It is necessary to create packages for all supported platforms and architectures.
To make it possible for `pkg` to choose the right archive, the archive must
be named in the following way:

```
salsaflow-<version>-<platform>-<architecture>.zip
```

For example it can be

```
salsaflow-0.4.0-darwin-amd64.zip
```
