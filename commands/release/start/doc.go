/*
Create a new release branch on top of the trunk branch.

  salsaflow release start [-no_fetch] [-next_trunk_version=NEXT]

Description

This command shall be used to start a new release branch according to the workflow.

Steps

This command goes through the following steps:

  1. Fetch the remote repository.
  2. Make sure the trunk branch is up to date.
  3. Make sure the release branch does not exist.
  4. The user is prompted to confirm the release. This step largely depends on
     the issue tracker module that is being used. Various checks can be carried
     out here to make sure the release can be started.
  5. Create the release branch on top of the trunk branch.
  6. Set and commit the trunk version string. This means that the version string
     is set for the release that is going to be forked off the trunk branch next.
     The new trunk version string is by default generated by auto-incrementing
     the current one (resetting the patch number to 0 and incrementing the minor by 1).
  7. Mark the release as started in the issue tracker. This again depends on
     the module that is being used.
  8. All modified branches are pushed.
*/
package startCmd