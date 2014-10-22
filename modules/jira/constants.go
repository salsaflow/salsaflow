package jira

var startableStateIds = []string{
	stateApprovedId,
}

// States --------------------------------------------------------------------

const (
	stateApprovedId   = "10000"
	stateCodingId     = "3"
	stateCodingDoneId = "10003"
)

// Transitions -----------------------------------------------------------------

const (
	transitionStartId = "21"
)
