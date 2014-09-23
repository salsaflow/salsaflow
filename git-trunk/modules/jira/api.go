package jira

import (
	// Stdlib
	"net/http"

	// Internal
	"github.com/salsita/SalsaFlow/git-trunk/modules/jira/client"
)

type BasicAuthRoundTripper struct {
	next http.RoundTripper
}

func (rt *BasicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(config.Username(), config.Password())
	return rt.next.RoundTrip(req)
}

func newClient() *client.Client {
	return client.New(config.BaseURL(), &http.Client{
		Transport: &BasicAuthRoundTripper{http.DefaultTransport},
	})
}

func fetchMyself() (*client.User, error) {
	myself, _, err := newClient().Myself.Get()
	return myself, err
}
