package metastore

import (
	"github.com/salsaflow/salsaflow/metastore/client"
)

func GetCommitMetadata(sha string) (*client.CommitData, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	store, err := NewStore(config.ServerURL().String(), config.Token())
	if err != nil {
		return nil, err
	}

	return store.GetCommitMetadata(sha)
}

func StoreCommitMetadata(data []*client.CommitData) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	store, err := NewStore(config.ServerURL().String(), config.Token())
	if err != nil {
		return err
	}

	return store.StoreCommitMetadata(data)
}
