package notes

import (
	// Stdlib
	"bytes"
	"io"
	"text/template"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
)

const htmlTemplate = `
<h2>Release Notes</h2>

<p>These are the release notes for version {{.Version}} as collected from the issue tracker.</p>

<p>The following sections contain the release notes grouped by story type.</p>
{{range .Sections}}
<h3>{{.StoryType}}</h3>
<ul>{{range .Stories}}
  <li>[<a href="{{.URL}}">{{.Id}}</a>] - {{.Title}}</li>{{end}}
</ul>
{{end}}
`

type htmlEncoder struct {
	writer io.Writer
}

func newHtmlEncoder(writer io.Writer) Encoder {
	return &htmlEncoder{writer}
}

func (encoder *htmlEncoder) Encode(nts *common.ReleaseNotes, opts *EncodeOptions) error {
	notes := toInternalRepresentation(nts)

	var output bytes.Buffer
	t := template.Must(template.New("release notes").Parse(htmlTemplate))
	if err := t.Execute(&output, notes); err != nil {
		return err
	}

	_, err := io.Copy(encoder.writer, &output)
	return err
}
