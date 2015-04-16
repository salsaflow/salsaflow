package github

import (
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func NewClient(token string) *github.Client {
	httpClient := oauth2.NewClient(oauth2.NoContext, &tokenSource{token})
	return github.NewClient(httpClient)
}

type tokenSource struct {
	token string
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: ts.token}, nil
}
