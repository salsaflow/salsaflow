package notes

import (
	// Stdlib
	"sort"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
	"github.com/salsaflow/salsaflow/version"
)

func GenerateReleaseNotes(v *version.Version, stories []common.Story) *common.ReleaseNotes {
	// Sort the stories. The following steps retain the order, so we can sort here.
	// We copy the slice so that we don't change the argument. Slices are references.
	sortedStories := make([]common.Story, len(stories))
	copy(sortedStories, stories)
	sort.Sort(common.Stories(sortedStories))

	// Group stories by their type into sections.
	// sectionMap: story.Type() -> *releaseNotesSection
	sectionMap := make(map[string]*common.ReleaseNotesSection)

	for _, story := range sortedStories {
		t := story.Type()
		s, ok := sectionMap[t]
		// In case the section is there, append.
		if ok {
			s.Stories = append(s.Stories, story)
			continue
		}
		// Otherwise create a new section.
		sectionMap[t] = &common.ReleaseNotesSection{
			StoryType: t,
			Stories:   []common.Story{story},
		}
	}

	// Collect sections into a slice.
	sections := make([]*common.ReleaseNotesSection, 0, len(sectionMap))
	for _, v := range sectionMap {
		sections = append(sections, v)
	}

	// Sort the sections alphabetically.
	sort.Sort(common.ReleaseNotesSections(sections))

	// Return the new internal representation.
	return &common.ReleaseNotes{
		Version:  v,
		Sections: sections,
	}
}
