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
	"os"
)

// Implement flag.Value interface for Command.
type helpValue Command

func (hv *helpValue) String() string {
	return "false"
}

func (hv *helpValue) Set(v string) error {
	((*Command)(hv)).Usage()
	os.Exit(0)
	return nil
}

// FIXME: Not sure how this works, is it even necessary?
func (hv *helpValue) IsBoolFlag() bool {
	return true
}

// Helper action that just wraps a call to Usage.
func helpAction(exitCode int) func(*Command, []string) {
	return func(cmd *Command, args []string) {
		cmd.Usage()
		os.Exit(exitCode)
	}
}
