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
	stateIdComplete         = "10404"
	stateIdReleased         = "10106"
	stateIdClosed           = "6"
)

var startableStateIds = []string{
	stateIdApproved,
}

var inDevelopmentStateIds = []string{
	stateIdBeingImplemented,
	stateIdImplemented,
}

// To pass the staging check, all the issues associated with the given release
// must be in one of the following states. However, only the issues what are
// Tested are in the end moved to the Staged state, the rest if left as it is.
var stageableStateIds = []string{
	stateIdTested,
	stateIdStaged,
	stateIdAccepted,
	stateIdReleased,
	stateIdDone,
	stateIdComplete,
	stateIdClosed,
}

var acceptedStateIds = []string{
	stateIdAccepted,
	stateIdReleased,
	stateIdDone,
	stateIdComplete,
	stateIdClosed,
}

// Transitions -----------------------------------------------------------------

const (
	transitionIdStartImplementing = "321"
	transitionIdStage             = "371"
	transitionIdUnstage           = "461"
	transitionIdRelease           = "91"
)
