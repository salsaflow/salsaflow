package pivotaltracker

import (
	// Stdlib
	"strconv"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/modules/common"

	// Other
	"gopkg.in/salsita/go-pivotaltracker.v0/v5/pivotal"
)

type story struct {
	s *pivotal.Story
}

func (story *story) GetId() string {
	return strconv.Itoa(story.s.Id)
}

func (story *story) GetAssignees() []common.User {
	var users []common.User
	for _, id := range story.s.OwnerIds {
		users = append(users, userId(id))
	}
	return users
}
