package tarfile

import "io"

// ExFileObject provides a file-like interface to a tar member.
type ExFileObject struct {
	tf     *TarFile
	ti     *TarInfo
	offset int64
	pos    int64
}

// NewExFileObject creates a new ExFileObject.
func NewExFileObject(tf *TarFile, ti *TarInfo) *ExFileObject {
	return &ExFileObject{
		tf:     tf,
		ti:     ti,
		offset: ti.OffsetData,
		pos:    0,
	}
}

// Read reads up to len(p) bytes from the tar member.
func (ef *ExFileObject) Read(p []byte) (int, error) {
	if ef.pos >= ef.ti.Size {
		return 0, io.EOF
	}
	if _, err := ef.tf.FileObj.Seek(ef.offset+ef.pos, io.SeekStart); err != nil {
		return 0, err
	}
	n, err := ef.tf.FileObj.Read(p)
	if err != nil && err != io.EOF {
		return n, err
	}
	if int64(n) > ef.ti.Size-ef.pos {
		n = int(ef.ti.Size - ef.pos)
	}
	ef.pos += int64(n)
	return n, err
}
