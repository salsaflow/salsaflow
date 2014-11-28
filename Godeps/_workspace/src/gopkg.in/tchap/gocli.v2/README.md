# About gocli #

[![Build
Status](https://drone.io/github.com/tchap/gocli/status.png)](https://drone.io/github.com/tchap/gocli/latest)

gocli is yet another package to aid with parsing command line arguments.

Unlike many other libraries, it focuses mainly on support of subcommands.
Define as many subcommands as you want, they are handled by using FlagSets
recursively. Simple yet powerful enough for many scenarios.

The help output format is inspired among others by codegangsta's cli library.

# Status #

The API has been tagged with `v1.0.0`, please use `gopkg.in` to lock your dependencies.

# Usage #

```go
import "gopkg.in/tchap/gocli.v2"
```

# Documentation #

[GoDoc](http://godoc.org/github.com/tchap/gocli)

# Example #

See `app_test.go`.

# License #

`MIT`, see the `LICENSE` file.
