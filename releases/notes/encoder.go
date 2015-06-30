package notes

import (
	// Stdlib
	"io"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
)

type Encoding string

const (
	EncodingHtml     Encoding = "html"
	EncodingJson     Encoding = "json"
	EncodingMarkdown Encoding = "markdown"
	EncodingYaml     Encoding = "yaml"
)

func AvailableEncodings() []string {
	return []string{
		string(EncodingHtml),
		string(EncodingJson),
		string(EncodingMarkdown),
		string(EncodingYaml),
	}
}

type EncodeOptions struct {
	// Try to pretty-print.
	// What this means depends on the encoder.
	Pretty bool
}

type Encoder interface {
	Encode(*common.ReleaseNotes, *EncodeOptions) error
}

type encoderConstructor func(io.Writer) Encoder

var encoderConstructors = map[Encoding]encoderConstructor{
	EncodingHtml:     newHtmlEncoder,
	EncodingJson:     newJsonEncoder,
	EncodingMarkdown: newMarkdownEncoder,
	EncodingYaml:     newYamlEncoder,
}

func NewEncoder(encoding Encoding, writer io.Writer) (Encoder, error) {
	constructor, ok := encoderConstructors[encoding]
	if !ok {
		return nil, &ErrUnknownEncoding{encoding}
	}
	return constructor(writer), nil
}
