package jira

// States --------------------------------------------------------------------

const (
	stateApprovedId   = "10000"
	stateCodingId     = "3"
	stateCodingDoneId = "10003"
)

var startableStateIds = []string{
	stateApprovedId,
}

var inProgressStateIds = []string{
	stateCodingId,
	stateCodingDoneId,
}

// Transitions -----------------------------------------------------------------

const (
	transitionStartId = "21"
)
