package metastore

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
	panic("Not implemented")
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
