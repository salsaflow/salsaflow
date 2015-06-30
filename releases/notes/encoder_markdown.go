package notes

import (
	// Stdlib
	"bytes"
	"io"
	"text/template"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
)

const markdownTemplate = `
## Release Notes ##

These are the release notes for version {{.Version}}.

The following sections contain the release notes grouped by story type.
{{range .Sections}}
### {{.StoryType}} ###
{{range .Stories}}
* [[{{.Id}}]({{.URL}})] - {{.Title}}{{end}}
{{end}}
`

type markdownEncoder struct {
	writer io.Writer
}

func newMarkdownEncoder(writer io.Writer) Encoder {
	return &markdownEncoder{writer}
}

func (encoder *markdownEncoder) Encode(nts *common.ReleaseNotes, opts *EncodeOptions) error {
	notes := toInternalRepresentation(nts)

	var output bytes.Buffer
	t := template.Must(template.New("release notes").Parse(markdownTemplate))
	if err := t.Execute(&output, notes); err != nil {
		return err
	}

	_, err := io.Copy(encoder.writer, &output)
	return err
}
