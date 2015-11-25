# SalsaFlow #

|||||||
| ---------- |:----------:| ---------- |:----------:| ---------- |:----------:|
| Buil status | [![Circle CI](https://circleci.com/gh/salsaflow/salsaflow/tree/develop.svg?style=svg)](https://circleci.com/gh/salsaflow/salsaflow/tree/develop) | GoDoc | [![GoDoc](https://godoc.org/github.com/salsaflow/salsaflow?status.png)](http://godoc.org/github.com/salsaflow/salsaflow) | Gitter IM | [![Join the chat at https://gitter.im/salsaflow/salsaflow](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/salsaflow/salsaflow) |

## Overview ##

SalsaFlow is your ultimate Trunk Based Development (TBD) CLI utility.

Actually, I don't know about you, but we use it here at [Salsita](https://www.salsitasoft.com/).

## Installation ##

SalsaFlow is written in Go. Compiling has never been that fast. No more [sword fighting](http://xkcd.com/303/) in the corridor, sorry...

### Installing from a Package ###

The pre-built binaries are attached to GitHub releases for this repository.

See the [latest](https://github.com/salsaflow/salsaflow/releases/latest) release for yourself!

So, to use the pre-built binaries,

1. download the relevant archive, then
2. copy the content to a directory in your `PATH`.
   Make sure that all the files are in the same directory.

#### Upgrading SalsaFlow ####

In case you are using the pre-built binaries and you want to upgrade
you SalsaFlow to the most recent version, you case use `salsaflow pkg upgrade`.
In fetches the artifacts attached to the latest GitHub release and replaces
the current executables.

In case you need to run `pkg upgrade` as root, you may need to use
`-config` flag to tell SalsaFlow here your global configuration file is.
It is better, though, to place SalsaFlow executables in a directory that is
writable by your usual user and just add that directory into `PATH`.

More about configuration is mentioned later.

### Installing from Sources ###

1. Install [Go](https://golang.org/dl/) (used Go 1.3.3, but any Go 1.x should do the trick).
2. Set up a Go [workspace](https://golang.org/doc/code.html#Workspaces).
3. Add the `bin` directory of your workspace to `PATH`.
4. Run `go get -d github.com/salsaflow/salsaflow`.
   This will get the sources and put them into the workspace.
   You can as well just go and use `git clone` directly...
5. Run `go get github.com/tools/godep` to install `godep`, which handles vendoring.
6. Run `make godep-install` in the project directory,
   which puts the resulting binaries into the `bin` directory of the workspace.
7. Run `salsaflow` to make sure everything went well.

### Other System Requirements ###

To use SalsaFlow, you will also need

* `git` version `1.9.x` or newer in your `PATH`

Modules may also require some additional packages to be installed.

## Usage ##

Well, the best thing you can do is to just run `salsaflow -h` and read.
More in-depth SalsaFlow principes are explained on the [wiki](https://github.com/salsaflow/salsaflow/wiki).

The complete list of SalsaFlow commands follows (links pointing to the `develop` docs):

* [cherry-pick](https://github.com/salsaflow/salsaflow/blob/develop/commands/cherrypick/README.md)
* [pkg install](https://github.com/salsaflow/salsaflow/blob/develop/commands/pkg/install/README.md)
* [pkg upgrade](https://github.com/salsaflow/salsaflow/blob/develop/commands/pkg/upgrade/README.md)
* [release changes](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/changes/README.md)
* [release deploy](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/deploy/README.md)
* [release notes](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/notes/README.md)
* [release stage](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/stage/README.md)
* [release start](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/start/README.md)
* [repo bootstrap](https://github.com/salsaflow/salsaflow/blob/develop/commands/repo/bootstrap/README.md)
* [repo init](https://github.com/salsaflow/salsaflow/blob/develop/commands/repo/init/README.md)
* [repo prune](https://github.com/salsaflow/salsaflow/blob/develop/commands/repo/prune/README.md)
* [review post](https://github.com/salsaflow/salsaflow/blob/develop/commands/review/post/README.md)
* [story changes](https://github.com/salsaflow/salsaflow/blob/develop/commands/story/changes/README.md)
* [story open](https://github.com/salsaflow/salsaflow/blob/develop/commands/story/open/README.md)
* [story start](https://github.com/salsaflow/salsaflow/blob/develop/commands/story/start/README.md)
* [version](https://github.com/salsaflow/salsaflow/blob/develop/commands/version/README.md)
* [version bump](https://github.com/salsaflow/salsaflow/blob/develop/commands/version/bump/README.md)

SalsaFlow can only be used when you are within a project repository (except the
`pkg` subcommands, these can be used anywhere).

The repository is automagically initialised when you run any SalsaFlow command there,
but you can also trigger the process by running `repo init`. SalsaFlow uses a couple of git hooks,
which are installed during the initialisation process.

You probably want to read the following section about SalsaFlow configuration
before doing anything serious since SalsaFlow will anyway refuse to do anything
useful until it is configured properly.

## Configuration ##

There are two places where SalsaFlow configuration is being kept:

1. The global, user-wide configuration is written into `$HOME/.salsaflow.json`.
   This file mostly contains the data that cannot be committed,
   i.e. access tokens and such.
2. The local, project-specific configuration is expected to be placed
   into `.salsaflow` directory in the repository root. This directory
   contains the local configuration file, `config.json`, as well as
   some project-specific custom scripts that are to be supplied
   by the user and committed into `.salsaflow/scripts`. More on custom
   scripts later.

### Global Configuration ###

The global, user-specific configuration file resides in `$HOME/.salsaflow.json`.
It stores module-specific configuration as a map. The exact format obviously depends
on what modules are being used.

You can use `-config` flag with any command to specify the path to the
global configuration file manually. This is handy when you need to run `pkg
upgrade` as root using `sudo`. In that case `$HOME` is not pointing to the home
directory of your usual user and SalsaFlow will fail to find the right
configuration file unless told where to look for it using `-config`.

### Local Configuration ###

SalsaFlow looks for the local cofiguration file in `$REPO_ROOT/.salsaflow/config.json`.
The structure is similar to the global configuration file except the fact that
it also includes the list of active modules for particular module kinds.

Too see a full example, just check the SalsaFlow
[config](https://github.com/salsaflow/salsaflow/blob/develop/.salsaflow/config.json) for this project.

#### Scripts ####

SalsaFlow occasionally needs to perform an action that depends on the project type,
e.g. to increment the version number when handling releases. These custom actions
must be configured by placing certain custom scripts into `.salsaflow/scripts`
directory in the repository.

The following scripts must be supplied:

* `get_version` - Print the current project version to stdout and exit.
* `set_version` - Taking the new version string as the only argument, this script is expected to
  set the project version string to the specified value. Make sure all new files are always
  staged (`git add`), otherwise they won't get committed by SalsaFlow.

Now, to make the whole scripting thing cross-platform, it is possible to supply
multiple script files for every script name and run different scripts on different platforms.
So, the filename schema for the scripts that are to be placed into the `scripts`
directory is actually `<script-name>_<platform>.<runner>` where

* `<script-name>` is the name as mentioned above, e.g. `get_version`.
* `<platform>` can be any valid value for Go's `runtime.GOOS`, e.g. `windows`, `linux`,
  `darwin` and so on. You can also use `unix` to run the script on all Unixy systems.
* `<runner>` is the file extension that defines what interpreter to use to run the script.
  Currently it can be `bash` (Bash), `js` (Node.js), `bat` (cmd.exe) or `ps1` (PowerShell.exe).
  Naturally, only some combinations make sense, e.g. you cannot run PowerShell on Mac OS X,
  so a script called `get_version_darwin.ps1` would never be executed.

Check some [examples](https://github.com/salsaflow/skeleton-golang) to understand better how the whole thing works.

#### Project Bootstrapping ####

To get up to speed quickly, `repo bootstrap` command can be used to generated the initial
configuration. The user is prompted for all necessary data, no need to edit
config files manually.

`repo bootstrap` can be also told to use certain GitHub repository to bootstrap the local
configuration directory. When this bootstrapping skeleton is supplied, `scripts` directory of the given
repository is simply poured into the local configuration directory. This can be easily used to
share custom scripts for certain project type so that the scripts are implemented once and then
just copied around. You can check the [repository](https://github.com/salsaflow/skeleton-golang)
that was used to bootstrap SalsaFlow itself.

## Modules ##

SalsaFlow interacts with various services to carry out requested actions.
The only supported VCS is [Git](git-scm.com), so that part is hard-coded in SalsaFlow,
but other serviced are configurable in the local configuration file, namely:

* the issue tracking module,
* the code review module, and
* the release notes module (optional).

`repo bootstrap` lists the available modules during repository bootstrapping.
The values actually listed depend on what modules are compiled into SalsaFlow.
You don't really need to understand much about modules unless you feel like
implement a new module for SalsaFlow. The user is prompted for all necessary
data when bootstrapping the project, which is a one-time action, and then the
active modules are simply used by SalsaFlow transparently.

## Original Authors ##

* [tchap](https://github.com/tchap) (for [Salsita](https://github.com/salsita))
* [realyze](https://github.com/realyze) (for [Salsita](https://github.com/salsita))

## License ##

`MIT`, see the `LICENSE` file.
