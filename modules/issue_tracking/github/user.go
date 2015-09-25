package github

import (
	// Vendor
	"github.com/google/go-github/github"
)

type user struct {
	me *github.User
}

func (u *user) Id() string {
	return *u.me.Login
}
