package jira

import (
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
