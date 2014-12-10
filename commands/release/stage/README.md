# `release stage` #

Stage the current release branch for acceptance.

## Usage ##

```
salsaflow release stage
```

## Description ##

Use this command to stage the current branch for acceptance. This means that
the release branch is tagged, closed and the staging branch is reset to point
to the tag. Pushing the staging branch then triggers deployment.

### Steps ###

This command goes through the following steps:

1. Fetch the remote repository to make sure the release branch exists
   and that it is up to date.
2. Make sure the release can be staged. This step depends on the issue tracker
   module, but in general the point is to make sure the assigned stories were
   reviewed and tested.
3. Tag the release branch with the release tag.
4. Delete the release branch.
5. Reset the staging branch to point to the newly created tag.
6. Update the remote repository. This is also supposed to trigger deployment.
