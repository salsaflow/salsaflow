package jira

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
