package pivotaltracker

import (
	// Stdlib
	"strconv"

	// Other
	"github.com/salsita/go-pivotaltracker/v5/pivotal"
)

type user struct {
	me *pivotal.Me
}

func (u *user) Id() string {
	return strconv.Itoa(u.me.Id)
}

type userId int

func (id userId) Id() string {
	return strconv.Itoa(int(id))
}
