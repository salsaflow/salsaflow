/*
Post review requests for the specified revisions.

  salsaflow review post [-fixes=RRID] [-reviewer=REVIEWER] [-open] [REVISION]

  salsaflow review post -parent=BRANCH [-fixes=RRID] [-no_fetch]
                       [-no_rebase] [-no_merge] [-merge_no_ff]
                       [-ask_once] [-pick] [-reviewer=REVIEWER] [-open]

See the command help page for more details in the flags and such.

Description

Post review requests for the specified revisions. The revision range can be
specified in multiple ways:

  1. By using the `parent` flag, all revisions between `BRANCH` and the current
     branch (`HEAD`) are selected for being posted into the code review system.
  2. When not using the `parent` flag, you can specify `REVISION`. This selects
     just a single commit to be posted.
  3. When not using the `parent` flag and not even specifying `REVISION`,
     the tip of the current branch (`HEAD`) is selected.

The overall workflow is explained in more details at
https://github.com/salsaflow/salsaflow/wiki/SalsaFlow-Workflow.

Tags

SalsaFlow will not allow you to post review requests without the selected
commits containing the Story-Id tag in the commit message. No need to worry,
though. When SalsaFlow finds a commit not containing the right tag, it will
prompt you to select the story to assign the commit to and it will amend
the commit message to insert the tag.

This mechanism is never triggered when REVISION is specified explicitly.
In general it is not possible to amend any commit in the git graph, so this
option is simply disabled in this case.

Steps

For the parent mode, the command goes through the following steps:

  1. Fetch the repository unless -no_fetch is specified.
  2. Select the commits to be posted for review - PARENT.. range.
  3. Walk the commits to check the Story-Id tags. In case the tag is missing
     for any of the selected commits, start constructing the revision range on
     a temporary branch, asking the user and amending the commit messages where
     necessary.
  4. Reset the current branch to point to the temporary branch.
  5. Merge the current branch into the parent branch unless -no_merge is set.
  6. Push the parent branch unless -no_merge is set.
     In that case the current branch is pushed.
  5. Post a review request for every commit in the range.

For the revision mode, the command goes through the following steps:

  1. Make sure the selected commit is associated with a story by the Story-Id
     tag. In case REVISION is given, fail in case the tag is not there. In case
     it is HEAD that is chosen, ask the user to select the story to assign the
     commit to. Amend the commit message.
  2. Post a review request for the given commit.
*/
package postCmd
