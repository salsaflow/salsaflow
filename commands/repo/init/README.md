# `repo init` #

Initialise the local repository for SalsaFlow.

## Usage ##

```
salsaflow repo init
```

## Description ##

This command initialises the local repository for SalsaFlow. In case you need
to generate initial configuration, `repo bootstrap` is your friend.

Anyway, this command makes sure the repository is initialised for SalsaFlow,
e.g. that the core branches are created and that the local git hooks are set up.
`repo init` also checks that all required dependencies are installed so that
other SalsaFlow commands work flawlessly.

### Steps ###

This command goes through the following steps:

1. In case the repository is already initialised, exit.
   SalsaFlow stores a flag in local git config to quickly check this.
2. Make sure that the git hooks are installed (install them if not).
   This is done by running the hook executables with special flags.
   Only the SF git hooks know these flags.
3. Check the git version being used, SF requires 1.9.0+.
4. Perform other registered checks, e.g. the Review Board module
   checks that RBTools 0.6.x are installed.
5. Mark the repository as initialised.
