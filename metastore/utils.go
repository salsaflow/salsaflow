package metastore

import (
	// Stdlib
	"fmt"

	// Internal
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/metastore/client"
)

type Resource struct {
	// ServiceId identifies the given service.
	// It can be "jira", "github" and so on.
	ServiceId string

	// Metadata keeps the metadata itself.
	// This is specific to each particular service.
	Metadata map[string]interface{}
}

type CommitData struct {
	Commit        *git.Commit
	Story         *Resource
	ReviewRequest *Resource
}

func FetchMetadataForCommits(commits []*git.Commit) (data []*CommitData, err error) {
	// Fetch raw data.
	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hashes = append(hashes, commit.SHA)
	}

	metadata, err := getMetadata(hashes)
	if err != nil {
		return nil, err
	}

	// Convert to CommitData.
	commitData := make([]*CommitData, 0, len(commits))
	for i := range commits {
		var (
			commit = commits[i]
			meta   = metadata[i]
		)

		story, err := dataToService(commit.SHA, meta.Story)
		if err != nil {
			return nil, err
		}
		reviewRequest, err := dataToService(commit.SHA, meta.ReviewRequest)
		if err != nil {
			return nil, err
		}

		commitData = append(commitData, &CommitData{
			Commit:        commit,
			Story:         story,
			ReviewRequest: reviewRequest,
		})
	}

	// Return CommitData.
	return commitData, nil
}

func StoreMetadataForCommits(data []*CommitData) error {
	// Convert []*CommitData into []*client.CommitData.
	rawData := make([]*client.CommitData, 0, len(data))
	for _, commit := range data {
		trackerData := serviceToData(commit.Story)
		reviewToolData := serviceToData(commit.ReviewRequest)
		rawData = append(rawData, &client.CommitData{
			SHA:           commit.Commit.SHA,
			Story:         trackerData,
			ReviewRequest: reviewToolData,
		})
	}

	// Store the metadata.
	return storeMetadata(rawData)
}

func dataToService(sha string, data map[string]interface{}) (*Resource, error) {
	var res Resource

	// Get the service ID.
	v, ok := data["service_id"]
	if !ok {
		return nil, &ErrFieldNotSet{sha, "service_id"}
	}
	id, ok := v.(string)
	if !ok {
		return nil, &ErrInvalidFieldType{sha, "service_id"}
	}
	res.ServiceId = id

	// We can delete the service_id key here.
	// There is no other reference to this map really.
	delete(data, "service_id")
	res.Metadata = data
	return &res, nil
}

func serviceToData(res *Resource) (data map[string]interface{}) {
	// We need to create a map that contains all the metadata
	// as well as res.ServiceId as "service_id".
	raw := make(map[string]interface{}, 1+len(res.Metadata))
	raw["service_id"] = res.ServiceId
	for k, v := range res.Metadata {
		raw[k] = v
	}
	return raw
}

type ErrFieldNotSet struct {
	sha   string
	field string
}

func (err *ErrFieldNotSet) Error() string {
	return fmt.Sprintf("commit %v: metadata field '%v': not set", err.sha, err.field)
}

type ErrInvalidFieldType struct {
	sha   string
	field string
}

func (err *ErrInvalidFieldType) Error() string {
	return fmt.Sprintf("commit %v: metadata field '%v': invalid type", err.sha, err.field)
}
