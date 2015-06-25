package metastore

import (
	// Stdlib
	"net/http"

	// Internal
	"github.com/salsaflow/salsaflow/metastore/client"
)

func getMetadata(hashes []string) ([]*client.CommitData, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	store, err := NewStore(config.ServerURL().String(), config.Token())
	if err != nil {
		return nil, err
	}

	metadata := make([]*client.CommitData, 0, len(hashes))
	for _, hash := range hashes {
		data, resp, err := store.GetCommitMetadata(hash)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				metadata = append(metadata, nil)
			}
			return nil, err
		}
		metadata = append(metadata, data)
	}
	return metadata, nil
}

func storeMetadata(data []*client.CommitData) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	store, err := NewStore(config.ServerURL().String(), config.Token())
	if err != nil {
		return err
	}

	_, err = store.StoreCommitMetadata(data)
	return err
}
