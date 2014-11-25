/*
   The MIT License (MIT)

   Copyright (c) 2013 Ondřej Kupka

   Permission is hereby granted, free of charge, to any person obtaining a copy of
   this software and associated documentation files (the "Software"), to deal in
   the Software without restriction, including without limitation the rights to
   use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
   the Software, and to permit persons to whom the Software is furnished to do so,
   subject to the following conditions:

   The above copyright notice and this permission notice shall be included in all
   copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
   FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
   COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
   IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package gocli

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

// Command represents a node in the chain of subcommands. Even App uses it,
// it is embedded there.
type Command struct {
	// Fitting to be set in the Command struct literal.
	UsageLine string
	Short     string
	Long      string
	Action    func(cmd *Command, args []string)

	// Flag set for this subcommand.
	Flags flag.FlagSet

	// All subcommands registered. Although it can be accessed directly,
	// it is not supposed to be used like that. Use MustRegisterSubcmd().
	// This is exported only for the templates to be able to access it.
	Subcmds []*Command

	// Internal field set according to the command type.
	helpTemplate     string
	helpTemplateData interface{}
}

// Get name from the usage line. The name is the first word on the usage line.
func (cmd *Command) Name() string {
	name := strings.TrimSpace(cmd.UsageLine)
	if i := strings.Index(name, " "); i >= 0 {
		name = name[:i]
	}
	return name
}

// Default Command help template.
var CommandHelpTemplate = `COMMAND:
  {{.Name}} - {{.Short}}

USAGE:
  {{.UsageLine}}

OPTIONS:
{{.DefaultFlagsString}}
{{with .Long}}DESCRIPTION:{{.}}{{end}}
{{with .Subcmds}}SUBCOMMANDS:
  {{range .}}{{.Name}}{{with .Short}}{{ "\t" }} - {{.}}{{end}}
  {{end}}
{{end}}`

// Print help and exit.
func (cmd *Command) Usage() {
	cmd.UsageLine = strings.TrimSpace(cmd.UsageLine)

	tw := tabwriter.NewWriter(os.Stderr, 0, 8, 2, '\t', 0)
	defer tw.Flush()

	t := template.Must(template.New("usage").Parse(cmd.helpTemplate))
	t.Execute(tw, cmd.helpTemplateData)
}

// Returns the same as flag.FlagSet.PrintDefaults, but as a string instead
// of printing it directly into the output stream.
func (cmd *Command) DefaultFlagsString() string {
	var b bytes.Buffer
	cmd.Flags.SetOutput(&b)
	cmd.Flags.PrintDefaults()
	return b.String()
}

// Run the command with supplied arguments. This is called recursively if a
// subcommand is detected, just a prefix is cut off from the arguments.
func (cmd *Command) Run(args []string) {
	err := cmd.Flags.Parse(args)
	if err != nil {
		cmd.Usage()
		os.Exit(1)
	}
	subArgs := cmd.Flags.Args()

	if len(subArgs) == 0 {
		cmd.Action(cmd, subArgs)
		return
	}

	name := subArgs[0]
	for _, subcmd := range cmd.Subcmds {
		if subcmd.Name() == name {
			subcmd.Run(subArgs[1:])
			return
		}
	}

	cmd.Action(cmd, subArgs)
}

// Register a new subcommand with the command. Panic if something is wrong.
func (cmd *Command) MustRegisterSubcommand(subcmd *Command) {
	// Require some fields to be non-empty.
	switch {
	case subcmd.Short == "":
		panic("Short not set")
	case subcmd.UsageLine == "":
		panic("UsageLine not set")
	}

	// Fill in the unexported fields.
	subcmd.helpTemplate = CommandHelpTemplate
	subcmd.helpTemplateData = subcmd

	// Print help if there is no action defined.
	if subcmd.Action == nil {
		subcmd.Action = helpAction(1)
	}

	// Define the help flag.
	subcmd.Flags.Var((*helpValue)(subcmd), "h", "print help and exit")

	// Easy if there are no subcommands defined yet.
	if cmd.Subcmds == nil {
		cmd.Subcmds = []*Command{subcmd}
		return
	}

	// Check for subcommand name collisions.
	for _, c := range cmd.Subcmds {
		if c.Name() == subcmd.Name() {
			panic(fmt.Sprintf("Subcommand %s already defined", subcmd.Name))
		}
	}

	cmd.Subcmds = append(cmd.Subcmds, subcmd)
}
