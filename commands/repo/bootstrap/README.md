# `repo bootstrap` #

Bootstrap the repository for SalsaFlow.

```
salsaflow repo bootstrap -skeleton=SKELETON [-skeleton_only]

salsaflow repo bootstrap -no_skeleton
```

## Description ##

This command should be used to set up the local configuration directory
for SalsaFlow (the directory that is then committed into the repository).

The user is prompted for all necessary data.

The -skeleton flag can be used to specify the repository to be used
for custom scripts. It expects a string of `$OWNER/$REPO` and then uses
the repository located at `github.com/$OWNER/$REPO`. It clones the repository
and copies `scripts` directory into the local configuration directory.

In case no skeleton is to be used to bootstrap the repository,
-no_skeleton must be specified explicitly.

In case the repository is bootstrapped, but the skeleton is missing,
it can be added by specifying `-skeleton=SKELETON -skeleton_only`.
That will skip the configuration file generation step.

### Steps ###

This command goes through the following steps:

1. In case -skeleton_only is not specified, prompt the user
   for all necessary config and save it into the local config directory.
2. In case -skeleton is specified, clone the given skeleton repository
   into the local cache, which is located in the current user's home directory.
   The repository is just pulled in case it already exists. Once the content is available,
   it is copied into the local metadata directory, .salsaflow.
