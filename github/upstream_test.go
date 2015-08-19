package github

import (
	"fmt"
)

type testingData struct {
	upstreamURL    string
	expectedOwner  string
	expectedRepo   string
	expectingError bool
}

var _ = Describe("parsing a GitHub remote upstream URL", func() {

	data := []testingData{
		//  regular URL, HTTPS scheme, .git suffix
		{
			"https://github.com/owner/repo.git",
			"owner",
			"repo",
			false,
		},
		// regular URL, SSH scheme, .git suffix
		{
			"ssh://git@github.com/owner/repo.git",
			"owner",
			"repo",
			false,
		},
		// regular URL, HTTPS scheme
		{
			"https://github.com/owner/repo",
			"owner",
			"repo",
			false,
		},
		// regular URL, SSH scheme
		{
			"ssh://git@github.com/owner/repo",
			"owner",
			"repo",
			false,
		},
		// regular URL, error - missing URL scheme
		{
			"github.com/owner/repo",
			"",
			"",
			true,
		},
		// regular URL, error - incomplete URL path
		{
			"github.com/owner/",
			"",
			"",
			true,
		},
		// SSH address, .git suffix
		{
			"git@github.com:owner/repo.git",
			"owner",
			"repo",
			false,
		},
		// SSH address
		{
			"git@github.com:owner/repo",
			"owner",
			"repo",
			false,
		},
		// SSH address, custom host (can be specified in .git/config)
		{
			"git@github-custom:owner/repo.git",
			"owner",
			"repo",
			false,
		},
		// SSH address, error - incomplete URL path
		{
			"git@github.com:owner/",
			"",
			"",
			true,
		},
	}

	for _, td := range data {
		func(d testingData) {

			Context(fmt.Sprintf("%+v", d), func() {

				It("should return expected results", func() {

					owner, repo, err := parseUpstreamURL(d.upstreamURL)

					Expect(owner).To(Equal(d.expectedOwner))
					Expect(repo).To(Equal(d.expectedRepo))

					if d.expectingError {
						Expect(err).ToNot(BeNil())
					} else {
						Expect(err).To(BeNil())
					}
				})
			})
		}(td)
	}

})
