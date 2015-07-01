package notes

import (
	// Stdlib
	"sort"
	"strings"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
)

// The following types basically mirror their counterparts from the common package.
// The main difference is that the types are annotated for encoders and also
// we don't use common.Story, but our own type, which makes it possible to serialize
// easily since the struct contains fields, not methods.

type releaseNotes struct {
	Version  string                 `json:"version"  yaml:"version"`
	Sections []*releaseNotesSection `json:"sections" yaml:"sections"`
}

type releaseNotesSection struct {
	StoryType string   `json:"story_type" yaml"story_type"`
	Stories   []*story `json:"stories"    yaml:"stories"`
}

type story struct {
	Id    string `json:"id"    yaml:"id"`
	Title string `json:"title" yaml:"title"`
	URL   string `json:"url"   yaml:"url"`
}

// Implement sort.Interface to sort story sections alphabetically.
type releaseNotesSections []*releaseNotesSection

func (sections releaseNotesSections) Len() int {
	return len(sections)
}

func (sections releaseNotesSections) Less(i, j int) bool {
	return sections[i].StoryType < sections[j].StoryType
}

func (sections releaseNotesSections) Swap(i, j int) {
	sections[i], sections[j] = sections[j], sections[i]
}

// toInternalRepresentation converts the given release notes to an internal
// representation that is easily serializable any typical encoder.
func toInternalRepresentation(notes *common.ReleaseNotes) *releaseNotes {
	// Generate sections.
	var sections []*releaseNotesSection
	for _, section := range notes.Sections {
		// Sort stories. What this means is specified to each issue tracker.
		sort.Sort(common.Stories(section.Stories))

		// Generate the story section.
		var stories []*story
		for _, s := range section.Stories {
			stories = append(stories, &story{
				Id:    s.ReadableId(),
				Title: s.Title(),
				URL:   s.URL(),
			})
		}
		sections = append(sections, &releaseNotesSection{
			StoryType: strings.Title(section.StoryType),
			Stories:   stories,
		})
	}

	// Sort sections alphabetically by story type.
	sort.Sort(releaseNotesSections(sections))

	// Return the new internal representation.
	return &releaseNotes{
		Version:  notes.Version.BaseString(),
		Sections: sections,
	}
}
