package metadata

import (
	"net/url"

	"github.com/salsaflow/salsaflow/metadata/client"
)

type Store struct {
	client *client.Client
}

func NewStore(baseURL *url.URL, token string) (*Store, error) {
	api, err := client.New(baseURL, token)
	if err != nil {
		return nil, err
	}

	return &Store{api}, nil
}
