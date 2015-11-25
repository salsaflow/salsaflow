/*
Cherry-pick commits into selected target branch.

  cherry-pick [-fetch] [-target=TARGET] COMMIT...

Description

Use git cherry-pick to copy the specified commits into the target branch,
which is the release branch by default. So this command is useful when
you need to get some changes from trunk to the release branch. But it can
be used for any cherry-picking when a custom target branch is specified.

This command makes sure that the target branch is up to date,
but it does not fetch the repository by default. Use -fetch to
update the repository before doing the check.

Steps

The command goes through the following steps:

  1. Fetch the remote repository when -fetch is set.
  2. Make sure the target branch is up to date.
  3. Parse the commit list to get commit hashes.
  4. Checkout the target branch.
  5. Run git cherry-pick with the given list of hashes.
  6. Checkout the original branch.
*/
package startCmd
