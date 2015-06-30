package notes

import (
	// Stdlib
	"bytes"
	"encoding/json"
	"io"

	// Internal
	"github.com/salsaflow/salsaflow/modules/common"
)

type jsonEncoder struct {
	writer io.Writer
}

func newJsonEncoder(writer io.Writer) Encoder {
	return &jsonEncoder{writer}
}

func (encoder *jsonEncoder) Encode(nts *common.ReleaseNotes, opts *EncodeOptions) error {
	notes := toInternalRepresentation(nts)

	var (
		raw []byte
		err error
	)
	if opts != nil && opts.Pretty {
		raw, err = json.MarshalIndent(notes, "", "  ")
	} else {
		raw, err = json.Marshal(notes)
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(encoder.writer, bytes.NewReader(raw))
	return err
}
