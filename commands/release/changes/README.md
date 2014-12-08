# `release changes` #

List the changes associated with the currently running release.

## Usage ##

```
salsaflow release changes [-porcelain] [-to_cherrypick]
```

## Description ##

This command can be used to list commits associated with the stories that were
added to the currently running release. The commits being listed are grouped by
`Change-Id` and then `Story-Id` tags.

The `to_cherrypick` flag can be used to list the changes that should be cherry-picked
into the release branch before the release is staged (the release branch is closed).

The `porcelain` flag makes the output more script-friendly.

### Steps ###

The command goes through the following steps:

1. Make sure the release branch exists.
2. Get the release version string (the version string stored in the release branch).
3. Fetch the associated stories from the issue tracker.
4. Collect the story commits and group them by `Change-Id` and `Story-Id`.
5. Show the list to the user.
