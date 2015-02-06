package prompt

import "strings"

const CommitTitleMaxLength = 50

func Shorten(line string, maxLen int) string {
	// In case the line is short enough, we are done.
	if len(line) <= maxLen {
		return line
	}

	// Incorporate the trailing " ..."
	truncated := line[:maxLen-4]

	// Drop the last word in case it was cut in half.
	if line[maxLen-4] != ' ' {
		if i := strings.LastIndex(truncated, " "); i != -1 {
			truncated = truncated[:i]
		}
	}

	// Return the truncated line with " ..." appended.
	return truncated + " ..."
}

func ShortenCommitTitle(title string) string {
	return Shorten(title, CommitTitleMaxLength)
}
