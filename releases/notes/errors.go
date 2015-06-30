package notes

type ErrUnknownEncoding struct {
	encoding Encoding
}

func (err *ErrUnknownEncoding) Error() string {
	return "unknown encoding: " + string(err.encoding)
}
