package bootstrapCmd

import (
	"io"
	"text/template"
)

const LocalConfigTemplate = `issue_tracker: "{{.IssueTrackerKey}}"
{{ if eq .IssueTrackerKey "jira" }}jira:
#  server_url: "https://example.com/jira"
#  project_key: "EX"{{ end }}
code_review_tool: "{{.CodeReviewToolKey}}"
{{ if eq .CodeReviewToolKey "review_board" }}review_board:
#  server_url: "https://review.example.com"{{ end }}
`

type LocalContext struct {
	IssueTrackerKey   string
	CodeReviewToolKey string
}

func BumpLocalConfig(dst io.Writer, ctx *LocalContext) error {
	return template.Must(template.New("LocalConfig").Parse(LocalConfigTemplate)).Execute(dst, ctx)
}
