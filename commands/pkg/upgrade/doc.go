/*
Upgrade SalsaFlow to the latest release.

Description

This command takes care of installing the latest SalsaFlow release.

The pre-built binaries are fetched from GitHub. They are expected to be appended
as release assets to the GitHub release specified by the given version.
Once the binaries are downloaded and unpacked, the current SalsaFlow binaries
are simply replaced by the new ones.

The repository that the assets are fetched from can be specified using
the available command line flags. By default it is `salsaflow/salsaflow`.

Release Assets

To make a GitHub release compatible with pkg, it is necessary to
append a few zip archives to the release. These archives are expected to
contain the pre-built binaries of SalsaFlow.

The binaries that are to be packed into the archive can be found in the
bin directory of your Go workspace after running make.

It is necessary to create packages for all supported platforms and architectures.
To make it possible for pkg to choose the right archive, the archive must
be named in the following way:

  salsaflow-<version>-<platform>-<architecture>.zip

For example it can be

  salsaflow-0.4.0-darwin-amd64.zip
*/
package upgradeCmd
