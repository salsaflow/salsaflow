# `story start` #

Start working on a new story.

## Usage ##

```
story start [-no_branch]
```

## Description ##

This command shall be used when the developer wants to start working on a new story.

### Steps ###

The command goes through the following steps:

1. Fetch startable stories from the issue tracker.
2. Prompt the user to select a story.
3. Prompt the user to insert the story branch slug unless `no_branch` is set.
4. In case a non-empty branch slug is inserted,
   create the specified story branch on top of trunk.
   The remote repository is fetched to make sure trunk is up to date
   before the story branch is created.
5. Add the user among the story owners.
6. Start the story in the issue tracker.
