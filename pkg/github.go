package pkg

import (
	"github.com/google/go-github/github"
	ghutil "github.com/salsaflow/salsaflow/github"
)

func newGitHubClient() (*github.Client, error) {
	config, err := ghutil.LoadConfig()
	if err != nil {
		return nil, err
	}

	return ghutil.NewClient(config.ApiToken()), nil
}
