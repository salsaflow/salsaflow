# SalsaFlow #

|||||
| ---------- |:----------:| ---------- |:----------:|
| Buil status | [![Circle CI](https://circleci.com/gh/salsaflow/salsaflow/tree/develop.svg?style=svg)](https://circleci.com/gh/salsaflow/salsaflow/tree/develop) | GoDoc | [![GoDoc](https://godoc.org/github.com/salsaflow/salsaflow?status.png)](http://godoc.org/github.com/salsaflow/salsaflow) |

## Overview ##

SalsaFlow is your ultimate Trunk Based Development (TBD) CLI utility.

Actually, I don't know about you, but we use it here at [Salsita](https://www.salsitasoft.com/).

## Installation ##

SalsaFlow is written in Go, which somehow implies how to install it.

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

## Usage ##

Well, the best thing you can do is to just run `salsaflow -h` and read.

Every SalsaFlow command is shortly explained when you run `salsaflow <command> -h`.
A more comprehensive documentation is available on the [wiki](https://github.com/salsaflow/salsaflow/wiki).

You probably want to read the following section before doing anything serious
with SalsaFlow since SalsaFlow will refuse to do anything meaningful until
it is configured properly.

## Configuration ##

There are two config files that are being used to configure SalsaFlow.
One is for the user-wide configuration, the other one is for the project-specific stuff.

### The Global Configuration File ###

The global, user-specific configuration file resides in `$HOME/.salsaflow.yml`.

The only universally required global config is:

```yaml
github:
  token: "<github-token>"
```

### The Local Configuration File ###

The local, project-specific configuration file is expected to be placed
in the repository root. It should be called `salsaflow.yml`.

The only universally required local config is:

```yaml
issue_tracker: "<module-id>"
code_review_tool: "<module-id>"
scripts:
  get_version: "<path-to-script>"
  set_version: "<path-to-script>"
```

#### Modules ####

As you must have noticed, _modules_ are mentioned above.
SalsaFlow is modular where possible, and the configuration files contain
sections where the configuration for these modules is specified.

A module must be first activated:

```yaml
issue_tracker: "jira"
code_review_tool: "review_board"
```

This is very close to dependency injection or something, you just tell SalsaFlow
what implementation to use for the given module (interface).

Then, when necessary, the module-specific config goes to the section
that is called after the module name, for example:

```yaml
jira:
  server_url: "https://jira.example.com"
  project_key: "SF"
review_board:
  server_url: "https://review.example.com"
```

The configuration for all available modules is described in more details later.

#### Scripts ####

SalsaFlow occasionally needs to perform an action that depends on the project type,
e.g. to increment the version number when handling releases. These custom actions
are configures in the `scripts` section of the local configuration file.

Every values in this section must be a path to the relevant script relative from
the repository root.

##### Get Project Version #####

The script located at `scripts.get_version` is expected to print the current
version string to stdout.

##### Set Project Version #####

The script located at `scripts.set_version` script is expected to bump
the version string to the values that is passed to it as the first and only argument.
SalsaFlow will handle the committing, just make sure the modified files are staged.

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

## Original Authors ##

* [tchap](https://github.com/tchap) (for [Salsita](https://github.com/salsita))
* [realyze](https://github.com/realyze) (for [Salsita](https://github.com/salsita))

## License ##

`MIT`, see the `LICENSE` file.
