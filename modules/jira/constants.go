package jira

// Issue Types -----------------------------------------------------------------

const (
	issueTypeIdBug              = "1"
	issueTypeIdTechnicalTask    = "10401"
	issueTypeIdTechnicalSubTask = "10002"
	issueTypeIdUserStory        = "10500"
)

var codingIssueTypeIds = []string{
	issueTypeIdBug,
	issueTypeIdTechnicalTask,
	issueTypeIdTechnicalSubTask,
	issueTypeIdUserStory,
}

// States --------------------------------------------------------------------

const (
	stateIdApproved         = "10000"
	stateIdBeingImplemented = "10400"
	stateIdImplemented      = "10401"
)

var startableStateIds = []string{
	stateIdApproved,
}

var inDevelopmentStateIds = []string{
	stateIdBeingImplemented,
	stateIdImplemented,
}

// Transitions -----------------------------------------------------------------

const (
	transitionIdStartImplementing = "321"
)
