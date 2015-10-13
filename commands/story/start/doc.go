/*
Start working on a new story.

  story start [-base=BASE] [-no_branch] [-push]

Description

This command shall be used when the developer wants to start working on a new story.

Unless -no_branch is specified, the user is asked to insert
the branch name to be used for the branch holding the story commits.
The branch of the given name is created on top of the trunk branch
and checked out. A custom base branch can be set by using -base.
The story branch is then pushed in case `-push` is specified.

Steps

The command goes through the following steps:

  1. Fetch startable stories from the issue tracker.
  2. Prompt the user to select a story.
  3. Prompt the user to insert the story branch slug unless -no_branch is set.
  4. In case a non-empty branch slug is inserted,
     create the specified story branch on top of trunk.
     The remote repository is fetched to make sure the base branch is up to date
     before the story branch is created.
  5. Add the user among the story owners.
  6. Start the story in the issue tracker.
  7. Push the story branch in case -push is set.
*/
package startCmd
