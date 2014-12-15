/*
Deploy a release into production.

  salsaflow release deploy [-release=VERSION]

Description

Deploy the selected release into production.

The release to be deployed can be selected in the following ways:

1. The -release command line flag selects VERSION as the release
   to be deployed. That's it.
2. When no release is specified manually, SalsaFlow prompts the user to select
   one of the releases that were staged since the previous deployment.
   In other words, it lists the tags that were created since the current stable
   branch position.

There are some checks in place so that you cannot simply deploy any release.
Well, actually, the -release flag cancels all checks, so it should be used
with caution, but for the selection mode of release deploy, SalsaFlow will
only list the releases that are suitable for being deployed. This usually means
that all associated stories were accepted. But it's not as simple as that.
SalsaFlow will never list a release that happened after a release that is not
releasable so that it is not possible to deploy stories that were not accepted.

Steps

This command goes through the following steps:

  1. Make sure the stable branch exists and is up to date.
  2. Select the release to be deployed. Prompt the user in case the release
     is not specified explicitly.
  3. Reset the stable branch to point to the tag associated with
     the given release.
  4. Push the stable branch to the remote repository. This is expected to trigger
     deployment (or any post-release step that makes sense for the project).
*/
package deployCmd
