package bootstrapCmd

import (
	"io"
	"text/template"
)

const LocalConfigTemplate = `# This field keeps the timestamp of when 'repo bootstrap' was executed.
# This timestamp is being used by the pre-push hook to check only the commits
# that happened after so that it is easier to migrate existing projects to SF.
# You can modify the timestamp manually in case there is a need.
salsaflow_enabled_timestamp: "{{.EnabledTimestamp}}"

#  Unless stated otherwise, the keys that are commented out
#  are required, so please uncomment them and fill in the values.

#-- ISSUE TRACKER
issue_tracker: "{{.IssueTrackerKey}}"
{{.IssueTrackerConfigTemplate}}

#-- CODE REVIEW TOOL
code_review_tool: "{{.CodeReviewToolKey}}"
{{.CodeReviewToolConfigTemplate}}

{{if .ReleaseNotesManagerKey}}#-- RELEASE NOTES MANAGER
release_notes: "{{.ReleaseNotesManagerKey}}"
{{.ReleaseNotesManagerConfigTemplate}}{{end}}
`

type LocalContext struct {
	EnabledTimestamp                  Time
	IssueTrackerKey                   string
	IssueTrackerConfigTemplate        string
	CodeReviewToolKey                 string
	CodeReviewToolConfigTemplate      string
	ReleaseNotesManagerKey            string
	ReleaseNotesManagerConfigTemplate string
}

func WriteLocalConfigTemplate(dst io.Writer, ctx *LocalContext) error {
	return template.Must(template.New("LocalConfig").Parse(LocalConfigTemplate)).Execute(dst, ctx)
}
