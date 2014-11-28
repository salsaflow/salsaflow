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

// The root object that starts the chain of subcommands.
type App struct {
	*Command

	Name    string
	Version string
}

// App constructor. Unfortunately App could not be written in a way that would
// allow to simply use a struct literal.
func NewApp(name string) *App {
	app := &App{
		Name: name,
		Command: &Command{
			Action: helpAction(1),
		},
	}
	app.Command.helpTemplate = AppHelpTemplate
	app.Command.helpTemplateData = app

	app.Command.Flags.Var((*helpValue)(app.Command), "h", "print help and exit")

	return app
}

// Default App help template.
var AppHelpTemplate = `APPLICATION:
  {{.Name}}{{with .Short}} - {{.}}{{end}}

{{with .UsageLine}}USAGE:
  {{.}}{{end}}

{{with .Version}}VERSION:
  {{.}}{{end}}

OPTIONS:
{{.DefaultFlagsString}}
{{with .Long}}DESCRIPTION:{{.}}{{end}}

{{with .Subcmds}}SUBCOMMANDS:
  {{range .}}{{.Name}}{{with .Short}}{{ "\t" }} {{.}}{{end}}
  {{end}}
{{end}}`
