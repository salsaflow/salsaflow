package metastore

import (
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

func (store *Store) GetCommitMetadata(sha string) (*client.CommitData, error) {
	data, _, err := store.client.Commits.Get(sha)
	return data, err
}

func (store *Store) PostCommitMetadata(data []*client.CommitData) error {
	_, err := store.client.Commits.Post(data)
	return err
}
