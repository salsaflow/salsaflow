package jira

import (
	// Internal
	"github.com/salsaflow/salsaflow/version"

	// Vendor
	"github.com/salsita/go-jira/v2/jira"
)

func isLabeled(issue *jira.Issue, label string) bool {
	for _, l := range issue.Fields.Labels {
		if l == label {
			return true
		}
	}
	return false
}

func isReleaseLabel(label string) bool {
	if label[0] != 'v' {
		return false
	}

	_, err := version.Parse(label[1:])
	return err == nil
}
