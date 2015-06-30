package notes

import (
	// Stdlib
	"bytes"
	"io"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"

	// Vendor
	"gopkg.in/yaml.v2"
)

type yamlEncoder struct {
	writer io.Writer
}

func newYamlEncoder(writer io.Writer) Encoder {
	return &yamlEncoder{writer}
}

func (encoder *yamlEncoder) Encode(nts *common.ReleaseNotes, opts *EncodeOptions) error {
	notes := toInternalRepresentation(nts)

	raw, err := yaml.Marshal(notes)
	if err != nil {
		return err
	}

	_, err = io.Copy(encoder.writer, bytes.NewReader(raw))
	return err
}
