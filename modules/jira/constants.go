package jira

// States --------------------------------------------------------------------

const (
	stateApprovedId = "10000"
	stateInCodingId = "3"
	stateInReviewId = "10003"
)

var startableStateIds = []string{
	stateApprovedId,
}

var inProgressStateIds = []string{
	stateInCodingId,
	stateInReviewId,
}

// Transitions -----------------------------------------------------------------

const (
	transitionStartId = "21"
)
