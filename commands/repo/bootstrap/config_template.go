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
{{ if eq .IssueTrackerKey "pivotal_tracker" }}pivotal_tracker:
#  project_id: 123456
#
#  THE FOLLOWING SECTION IS OPTIONAL.
#  The values visible there are the default choices.
#  In case the defaults are fine, just delete the section,
#  otherwise uncomment what you need to change.
#
#  labels:
#    point_me: "point_me"
#    no_review: "no review"
#    reviewed: "reviewed"
#    verified: "qa+"
#    skip_check_labels:
#      - "wontfix"
#      - "dupe"
{{ else if eq .IssueTrackerKey "jira" }}jira:
#  server_url: "https://example.com/jira"
#  project_key: "EX"
{{ end }}
#-- CODE REVIEW TOOL
code_review_tool: "{{.CodeReviewToolKey}}"
{{ if eq .CodeReviewToolKey "review_board" }}review_board:
#  server_url: "https://review.example.com"{{ end }}

{{with .ReleaseNotesManagerKey}}#-- RELEASE NOTES MANAGER
release_notes: "{{.}}"{{end}}
`

type LocalContext struct {
	EnabledTimestamp       Time
	IssueTrackerKey        string
	CodeReviewToolKey      string
	ReleaseNotesManagerKey string
}

func WriteLocalConfigTemplate(dst io.Writer, ctx *LocalContext) error {
	return template.Must(template.New("LocalConfig").Parse(LocalConfigTemplate)).Execute(dst, ctx)
}
