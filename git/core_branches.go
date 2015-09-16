package git

import (
	// Stdlib
	"bufio"
	"fmt"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/errs"
)

func IsCoreBranch(branch string) (bool, error) {
	coreBranches, err := coreBranchSet()
	if err != nil {
		return false, err
	}

	_, isCore := coreBranches[branch]
	return isCore, nil
}

// CoreBranchHashes returns a map containing all core branches with their hashes,
// i.e. map[branchName]commitHash.
func CoreBranchHashes() (map[string]string, error) {
	coreBranches, err := coreBranchSet()
	if err != nil {
		return nil, err
	}

	// Get all heads known to git.
	//
	// The output consists of lines with the following format:
	//
	// <hash> <ref>
	//
	// e.g.
	//
	// 07f9c47a3005d3545991d4175cd9866c02dbdd85 refs/heads/master
	//
	output, err := Run("show-ref", "--heads")
	if err != nil {
		return nil, err
	}

	// Parse the output.
	var (
		task    = "Parse git show-ref output"
		hashes  = make(map[string]string, len(coreBranches))
		scanner = bufio.NewScanner(output)
	)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			return nil, errs.NewError(
				task, fmt.Errorf("failed to parse output line: %v", line))
		}
		hash, ref := parts[0], parts[1]
		// Drop the refs/heads/ prefix.
		branch := ref[len("refs/heads/"):]
		// In case this is a core branch, add it to the resulting map.
		if _, ok := coreBranches[branch]; ok {
			hashes[branch] = hash
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return hashes, nil
}

func coreBranchSet() (map[string]struct{}, error) {
	// Load config.
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	// Return the core branches set.
	return map[string]struct{}{
		config.TrunkBranchName:   struct{}{},
		config.ReleaseBranchName: struct{}{},
		config.StagingBranchName: struct{}{},
		config.StableBranchName:  struct{}{},
	}, nil
}
