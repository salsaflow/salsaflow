# `version bump` #

Bump the project version string.

## Usage ##

```
salsaflow version bump [-commit] VERSION
```

## Description ##

Bump the project version string by invoking the `set_version` custom script.

In case `-commit` is set, the version string is committed into the current branch.

### Steps ###

This command goes through the following steps:

1. Invoke the `set_version` script, passing the new version string as the only argument.
2. In case `-commit` is set, commit the version string into the current branch.
