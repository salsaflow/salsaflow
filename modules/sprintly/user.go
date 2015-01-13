package sprintly

import (
	// Stdlib
	"strconv"

	// Other
	"github.com/salsita/go-sprintly/sprintly"
)

type user struct {
	*sprintly.User
}

func (u *user) Id() string {
	return strconv.Itoa(u.User.Id)
}
