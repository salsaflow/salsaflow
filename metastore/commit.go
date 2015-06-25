package metastore

import (
	"github.com/salsaflow/salsaflow/git"
	"github.com/salsaflow/salsaflow/metastore/client"
)

type Commit struct {
	Commit *git.Commit
	Meta   *client.CommitData
}

func GetMetadataForCommits(commits []*git.Commit) ([]*Commit, error) {
	hashes := make([]string, 0, len(commits))
	for _, commit := range commits {
		hashes = append(hashes, commit.SHA)
	}

	metadata, err := GetCommitMetadata(hashes)
	if err != nil {
		return nil, err
	}

	metaCommits := make([]*Commit, 0, len(commits))
	for i := range commits {
		metaCommits = append(metaCommits, &Commit{
			Commit: commits[i],
			Meta:   metadata[i],
		})
	}
	return metaCommits, nil
}
