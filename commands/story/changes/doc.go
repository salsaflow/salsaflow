/*
List story commits grouped by Change-Id.

  salsaflow story changes STORY_ID

Description

This command can be used to list Git changes associated with the given Story-Id tag.
The listed commits are grouped by the Change-Id tag and every commit is associated with
a source branch that tells what branch the commit belongs to. This makes it very easy
to see where particular changes are located.
*/
package changesCmd
