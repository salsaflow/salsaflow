package jira

// Issue Types -----------------------------------------------------------------

const (
	issueTypeIdBug           = "1"
	issueTypeIdCodingTask    = "10508"
	issueTypeIdCodingSubTask = "10501"
	issueTypeIdUserStory     = "10500"
)

var codingIssueTypeIds = []string{
	issueTypeIdBug,
	issueTypeIdCodingTask,
	issueTypeIdCodingSubTask,
	issueTypeIdUserStory,
}

// States --------------------------------------------------------------------

const (
	stateIdApproved         = "10000"
	stateIdBeingImplemented = "10400"
	stateIdImplemented      = "10401"
	stateIdTested           = "10103"
	stateIdStaged           = "10105"
	stateIdAccepted         = "10005"
	stateIdDone             = "10108"
)

var startableStateIds = []string{
	stateIdApproved,
}

var inDevelopmentStateIds = []string{
	stateIdBeingImplemented,
	stateIdImplemented,
}

var stageableStateIds = []string{
	stateIdTested,
	stateIdStaged,
	stateIdAccepted,
	stateIdDone,
}

// Transitions -----------------------------------------------------------------

const (
	transitionIdStartImplementing = "321"
	transitionIdStage             = "371"
	transitionIdUnstage           = "461"
)
