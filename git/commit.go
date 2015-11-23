package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"
)

type Commit struct {
	SHA             string
	Source          string
	Merge           string
	Author          string
	AuthorDate      time.Time
	Committer       string
	CommitDate      time.Time
	MessageTitle    string
	Message         string
	ChangeIdTag     string
	IssueTrackerTag string
	StoryIdTag      string
}

var commitTemplate = `commit {{.SHA}} {{.Source}}{{with .Merge}}Merge: {{.}}{{"\n"}}{{end}}
Author:     {{.Author}}
AuthorDate: {{.AuthorDate}}
Commit:     {{.Committer}}
CommitDate: {{.CommitDate}}

{{.Message | indent }}
`

// Dump writes a human-friendly representation of the commit to the writer.
// The output closely resembles the output of git log.
func (commit *Commit) Dump(wr io.Writer) error {
	funcMap := template.FuncMap{
		"indent": func(content string) (string, error) {
			var out bytes.Buffer
			scanner := bufio.NewScanner(strings.NewReader(content))
			for scanner.Scan() {
				if _, err := fmt.Fprintln(&out, "    ", scanner.Text()); err != nil {
					return "", err
				}
			}
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return out.String(), nil
		},
	}

	tpl := template.Must(template.New("commit").Funcs(funcMap).Parse(commitTemplate))
	return tpl.Execute(wr, commit)
}
