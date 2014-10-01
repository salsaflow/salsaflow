# SalsaFlow #

Your ultimate Trunk Based Development (TBD) CLI utility.

Actually, I don't know about you, but we use it here at [Salsita](https://www.salsitasoft.com/).

## Installation ##

1. Install [Go](https://golang.org/dl/) (used Go 1.3.1, but any Go 1.x should do the trick).
2. Set up a Go [workspace](https://golang.org/doc/code.html#Workspaces).
3. Run `go get github.com/salsita/salsaflow`. This will get the sources and install the executable into the workspace.
   Add `$GOPATH/bin` into `PATH` to be able to run the executable directly from the command line.
4. Run `go get github.com/salsita/salsaflow/bin/hooks/salsaflow-commit-msg`,
   then use it as the `commit-msg` [hook](http://git-scm.com/book/en/Customizing-Git-Git-Hooks) in your repo.

### Other System Requirements ###

* `git>=2.0.0` in your `PATH`

## License ##

`MIT`, see the `LICENSE` file.
