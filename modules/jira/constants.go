package jira

// States --------------------------------------------------------------------

const (
	stateApprovedId      = "10000"
	stateInDevelopmentId = "3"
)

var startableStateIds = []string{
	stateApprovedId,
}

var inDevelopmentStateIds = []string{
	stateInDevelopmentId,
}

// Transitions -----------------------------------------------------------------

const (
	transitionStartId = "21"
)
