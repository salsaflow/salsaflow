package common

// FilterStories is the filter function implemented for []common.Story type.
func FilterStories(stories []Story, filter func(Story) bool) []Story {
	ss := make([]Story, 0, len(stories))
	for _, story := range stories {
		if filter(story) {
			ss = append(ss, story)
		}
	}
	return ss
}
