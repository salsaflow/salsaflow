package pkg

import (
	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
)

func newGitHubClient() (*github.Client, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	transport := &oauth.Transport{
		Token: &oauth.Token{AccessToken: config.GitHubToken()},
	}
	return github.NewClient(transport.Client()), nil
}
