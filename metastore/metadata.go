package metastore

import (
	"github.com/salsaflow/salsaflow/git"
)

type ServiceData struct {
	// ServiceId identifies the given service.
	// It can be "jira", "github" and so on.
	ServiceId string

	// Metadata keeps the metadata itself.
	// This is specific to each particular service.
	Metadata map[string]interface{}
}

type CommitData struct {
	Commit         *git.Commit
	IssueTracker   *ServiceData
	CodeReviewTool *ServiceData
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

		tracker, err := dataToService(commit.SHA, meta.IssueTracker)
		if err != nil {
			return nil, err
		}
		reviewTool := dataToService(commit.SHA, meta.CodeReviewTool)
		if err != nil {
			return nil, err
		}

		commitData = append(commitData, &CommitData{
			Commit:         commit,
			IssueTracker:   &tracker,
			CodeReviewTool: &reviewTool,
		})
	}

	// Return CommitData.
	return metaCommits, nil
}

func StoreMetadataForCommits(data []*CommitData) error {
	// Convert []*CommitData into []*client.CommitData.
	rawData := make([]client.CommitData, 0, len(data))
	for _, commit := range data {
		trackerData := serviceToData(commit.IssueTracker)
		reviewToolData := serviceToData(commit.CodeReviewTool)
		rawData = append(rawData, &client.CommitData{
			SHA:            commit.SHA,
			IssueTracker:   trackerData,
			CodeReviewTool: reviewToolData,
		})
	}

	// Store the metadata.
	return storeMetadata(rawData)
}

func dataToService(sha string, data map[string]interface{}) (*ServiceData, error) {
	var srv ServiceData

	// Get the service ID.
	v, ok := data["service_id"]
	if !ok {
		return nil, &ErrInvalidData{sha, data}
	}
	id, ok := v.(string)
	if !ok {
		return nil, &ErrInvalidData{sha, data}
	}

	// We can delete the service_id key here.
	// There is no other reference to this map really.
	delete(data, "service_id")
	srv.Metadata = data
	return &srv, nil
}

func serviceToData(srv *ServiceData) (data map[string]interface{}) {
	// We need to create a map that contains all the metadata
	// as well as srv.ServiceId as "service_id".
	raw := make(map[string]interface{}, 1+len(srv.Metadata))
	raw["service_id"] = srv.ServiceId
	for k, v := range srv.Metadata {
		raw[k] = v
	}
	return raw
}
