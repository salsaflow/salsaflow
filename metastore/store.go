package metastore

import (
	// Stdlib
	"net/http"

	// Internal
	"github.com/salsaflow/salsaflow/metastore/client"
)

type Store struct {
	client *client.Client
}

func NewStore(baseURL, token string) (*Store, error) {
	api, err := client.New(baseURL, token)
	if err != nil {
		return nil, err
	}

	return &Store{api}, nil
}

func (store *Store) GetCommitMetadata(sha string) (*client.CommitData, *http.Response, error) {
	return store.client.Commits.Get(sha)
}

func (store *Store) StoreCommitMetadata(data []*client.CommitData) (*http.Response, error) {
	return store.client.Commits.Post(data)
}
