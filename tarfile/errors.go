package tarfile

type TarError struct {
	msg string
}

func (e *TarError) Error() string { return e.msg }

type ExtractError struct{ TarError }
type ReadError struct{ TarError }
type CompressionError struct{ TarError }
type StreamError struct{ TarError }
type HeaderError struct{ TarError }
type EmptyHeaderError struct{ HeaderError }
type TruncatedHeaderError struct{ HeaderError }
type EOFHeaderError struct{ HeaderError }
type InvalidHeaderError struct{ HeaderError }
type SubsequentHeaderError struct{ HeaderError }

func NewTarError(msg string) error {
	return &TarError{msg: msg}
}

func NewExtractError(msg string) error {
	return &ExtractError{TarError{msg: msg}}
}

func NewReadError(msg string) error {
	return &ReadError{TarError{msg: msg}}
}

func NewCompressionError(msg string) error {
	return &CompressionError{TarError{msg: msg}}
}

func NewStreamError(msg string) error {
	return &StreamError{TarError{msg: msg}}
}

func NewEmptyHeaderError(msg string) error {
	return &EmptyHeaderError{HeaderError{TarError{msg: msg}}}
}

func NewTruncatedHeaderError(msg string) error {
	return &TruncatedHeaderError{HeaderError{TarError{msg: msg}}}
}

func NewEOFHeaderError(msg string) error {
	return &EOFHeaderError{HeaderError{TarError{msg: msg}}}
}

func NewInvalidHeaderError(msg string) error {
	return &InvalidHeaderError{HeaderError{TarError{msg: msg}}}
}

func NewSubsequentHeaderError(msg string) error {
	return &SubsequentHeaderError{HeaderError{TarError{msg: msg}}}
}
