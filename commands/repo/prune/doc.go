/*
Delete Git branches that are no longer needed.

  salsaflow repo prune -state=STATE

Description

Delete Git branches that are no longer needed.

All story branches are checked and the branches are only contain commits
associated with stories that are in the selected state or further
are offered to be deleted. Both local and remote branches are affected.

This commands counts on the fact that all branches starting with story/
are forked off trunk. In case this is not met, weird things can happen.

Steps

This command goes through the following steps:

  1. Collect all story branches.
  2. Drop the branches that contain a commit from a story
     that does not comply with the state requirements.
  3. Prompt the user for confirmation for every branch remaining.
  4. Delete the chosen branches, both local and remote.
*/
package pruneCmd
