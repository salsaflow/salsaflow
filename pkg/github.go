package pkg

import (
	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
)

func newGitHubClient(token string) (*github.Client, error) {
	transport := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}
	return github.NewClient(transport.Client()), nil
}
