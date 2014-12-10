# SalsaFlow #

|||||
| ---------- |:----------:| ---------- |:----------:|
| Buil status | [![Circle CI](https://circleci.com/gh/salsaflow/salsaflow/tree/develop.svg?style=svg)](https://circleci.com/gh/salsaflow/salsaflow/tree/develop) | GoDoc | [![GoDoc](https://godoc.org/github.com/salsaflow/salsaflow?status.png)](http://godoc.org/github.com/salsaflow/salsaflow) |

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
More in-depth SalsaFlow principes are explained in the [wiki](https://github.com/salsaflow/salsaflow/wiki).

The complete list of SalsaFlow commands follows (links pointing to the `develop` docs):

* [pkg install](https://github.com/salsaflow/salsaflow/blob/develop/commands/pkg/install/README.md)
* [pkg upgrade](https://github.com/salsaflow/salsaflow/blob/develop/commands/pkg/upgrade/README.md)
* [release changes](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/changes/README.md)
* [release deploy](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/deploy/README.md)
* [release stage](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/stage/README.md)
* [release start](https://github.com/salsaflow/salsaflow/blob/develop/commands/release/start/README.md)
* [repo bootstrap](https://github.com/salsaflow/salsaflow/blob/develop/commands/repo/bootstrap/README.md)
* [repo init](https://github.com/salsaflow/salsaflow/blob/develop/commands/repo/init/README.md)
* [story changes](https://github.com/salsaflow/salsaflow/blob/develop/commands/story/changes/README.md)
* [story open](https://github.com/salsaflow/salsaflow/blob/develop/commands/story/open/README.md)
* [story start](https://github.com/salsaflow/salsaflow/blob/develop/commands/story/start/README.md)
* [version](https://github.com/salsaflow/salsaflow/blob/develop/commands/version/README.md)

SalsaFlow can only be used when you are within a project repository.
The repository is automagically initialised when you run any SalsaFlow command there.
SalsaFlow uses a couple of git hooks, which are installed as a part
of the initialisation process.

You probably want to read the following section about SalsaFlow configuration
before doing anything serious since SalsaFlow will anyway refuse to do anything
useful until it is configured properly.

## Configuration ##

There are two places where SalsaFlow configuration is being kept:

1. The local, project-specific configuration is expected to be placed
   into `.salsaflow` directory in the repository root. This directory
   contains the local configuration file, `config.yml`, as well as
   some project-specific custom scripts that are to be supplied
   by the user and committed into `.salsaflow/scripts`. More on custom
   scripts later.
2. The global, user-wide configuration is to be placed into `$HOME/.salsaflow.yml`.
   This file mostly contains the data that cannot be committed,
   i.e. access tokens and such.

### Global Configuration ###

The global, user-specific configuration file resides in `$HOME/.salsaflow.yml`.
The format depends solely on the modules that are active. No worries, modules
are explained later.

### Local Configuration ###

SalsaFlow looks for the local cofiguration file in `.salsaflow/config.yml`.
The only universally required local configuration keys are

```yaml
issue_tracker: "<module-id>"
code_review_tool: "<module-id>"
```

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

To get up to speed quickly, `repo bootstrap` command can be used to prepare the local
configuration file so that the user only fills in the missing values that are clearly marked.

`repo bootstrap` can be also told to use certain GitHub repository to bootstrap the local
configuration directory. When this bootstrapping skeleton is supplied, the contents of the given
repository are simply poured into the local configuration directory. This can be easily used to
share custom scripts for certain project type so that the scripts are implemented once and then
just copied around. You can check the [repository](https://github.com/salsaflow/skeleton-golang)
that was used to bootstrap SalsaFlow itself.

#### Modules ####

SalsaFlow is modular where possible, and the configuration files contain
sections where the configuration for these modules is specified.

A module must be first activated:

```yaml
issue_tracker: "jira"
code_review_tool: "review_board"
```

This is very close to dependency injection. There are a few module types and it
must be specified what implementation to use for the given module type (interface).

Then, when necessary, the module-specific config goes to the section
that is names after the module name, for example:

```yaml
jira:
  server_url: "https://jira.example.com"
  project_key: "SF"
review_board:
  server_url: "https://review.example.com"
```

The configuration for all available modules is described in more details later.

## Modules ##

SalsaFlow interacts with various services to carry out requested actions.
The only supported VCS is [Git](git-scm.com), so that part is hard-coded in SalsaFlow,
but other serviced are configurable in the local configuration file, namely:

* The issue tracker module must be specified under the `issue_tracker` key.
  Allowed values are: `jira`.
* The code review module must be specified under the `code_review_tool` key.
  Allowed values are: `review_board`.

### Supported Issue Trackers ###

#### JIRA ####

To activate this module, put the following config into the **local** configuration file:

```yaml
issue_tracker: "jira"
jira:
  server_url: "https://jira.example.com"
  project_key: "SF"
```

where

* `server_url` is the URL that can be used to access JIRA, and
* `project_key` is the JIRA project key that the repository is associated with.

The **global** configuration file must then contain the following additional config:

```yaml
jira:
  credentials:
    - server_prefix: jira.example.com
      username: "username"
      password: "secret"
    - server_prefix: jira.another-example.com
      username: "another-username"
      password: "another-secret"
```

where

* `server_prefix` is being used to bind credentials to JIRA instance.
   The URL scheme is not being used for matching, hence `jira.example.com`.
   You can specify multiple records, the longest match wins.
* `username` is the JIRA username to be used for the given JIRA instance, and
* `password` is the JIRA password to be used for the given JIRA instance.

As apparent from the example, there can be multiple server records in the file.

### Supported Code Review Tools ###

#### Review Board ####

To activate this module, put the following config into the **local** configuration file:

```yaml
code_review_tool: "review_board"
review_board:
  server_url: "https://review.example.com"
```

where

* `server_url` is the URL that can be used to access Review Board.

Please make sure that `RBTools` package version `0.6.x` is installed.
This module relies on the `rbt` command heavily.

## Original Authors ##

* [tchap](https://github.com/tchap) (for [Salsita](https://github.com/salsita))
* [realyze](https://github.com/realyze) (for [Salsita](https://github.com/salsita))

## License ##

`MIT`, see the `LICENSE` file.
