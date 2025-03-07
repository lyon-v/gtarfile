package tarfile

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/ulikunitz/xz" // 引入第三方 xz 包
)

// Stream represents a stream of tar blocks.
type Stream struct {
	file io.ReadWriteCloser
}

// newStream creates a new Stream for tar block streaming.
func newStream(name, mode, comptype string, fileobj io.ReadWriteSeeker, bufsize, compresslevel int) (*Stream, error) {
	var f io.ReadWriteCloser
	if fileobj != nil {
		switch comptype {
		case "tar":
			f = &fileWrapper{rws: fileobj}
		case "gz":
			if mode == "r" {
				gz, err := gzip.NewReader(fileobj)
				if err != nil {
					return nil, err
				}
				f = &readWriteCloser{r: gz, w: fileobj}
			} else { // 写模式
				gz, err := gzip.NewWriterLevel(fileobj, compresslevel)
				if err != nil {
					return nil, err
				}
				f = &writeCloser{w: gz, c: wrapCloser(fileobj)}
			}
		case "bz2":
			if mode == "r" {
				f = &readWriteCloser{r: bzip2.NewReader(fileobj), w: fileobj}
			} else {
				return nil, NewCompressionError("bz2 streaming write not implemented in stdlib")
			}
		case "xz":
			if mode == "r" {
				xzReader, err := xz.NewReader(fileobj)
				if err != nil {
					return nil, err
				}
				f = &readWriteCloser{r: xzReader, w: fileobj}
			} else {
				xzWriter, err := xz.NewWriter(fileobj)
				if err != nil {
					return nil, err
				}
				f = &writeCloser{w: xzWriter, c: wrapCloser(fileobj)}
			}
		default:
			return nil, NewCompressionError("unknown compression type " + comptype)
		}
	} else {
		file, err := os.OpenFile(name, osMode(mode+"b"), 0666)
		if err != nil {
			return nil, err
		}
		switch comptype {
		case "tar":
			f = file
		case "gz":
			if mode == "r" {
				gz, err := gzip.NewReader(file)
				if err != nil {
					file.Close()
					return nil, err
				}
				f = &readWriteCloser{r: gz, w: file}
			} else {
				gz, err := gzip.NewWriterLevel(file, compresslevel)
				if err != nil {
					file.Close()
					return nil, err
				}
				f = &writeCloser{w: gz, c: file} // os.File 实现了 io.Closer 和 io.Seeker
			}
		case "bz2":
			if mode == "r" {
				f = &readWriteCloser{r: bzip2.NewReader(file), w: file}
			} else {
				file.Close()
				return nil, NewCompressionError("bz2 streaming write not implemented in stdlib")
			}
		case "xz":
			if mode == "r" {
				xzReader, err := xz.NewReader(file)
				if err != nil {
					file.Close()
					return nil, err
				}
				f = &readWriteCloser{r: xzReader, w: file}
			} else {
				xzWriter, err := xz.NewWriter(file)
				if err != nil {
					file.Close()
					return nil, err
				}
				f = &writeCloser{w: xzWriter, c: file}
			}
		default:
			file.Close()
			return nil, NewCompressionError("unknown compression type " + comptype)
		}
	}
	return &Stream{file: f}, nil
}

// Read implements io.Reader.
func (s *Stream) Read(p []byte) (int, error) {
	return s.file.Read(p)
}

// Write implements io.Writer.
func (s *Stream) Write(p []byte) (int, error) {
	return s.file.Write(p)
}

// Seek implements io.Seeker.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := s.file.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fmt.Errorf("stream does not support seeking")
}

// Close implements io.Closer.
func (s *Stream) Close() error {
	return s.file.Close()
}

// readWriteCloser adapts a Reader and Closer to ReadWriteCloser.
type readWriteCloser struct {
	r io.Reader
	w io.ReadWriteSeeker
}

func (rwc *readWriteCloser) Read(p []byte) (int, error)  { return rwc.r.Read(p) }
func (rwc *readWriteCloser) Write(p []byte) (int, error) { return 0, fmt.Errorf("write not supported") }
func (rwc *readWriteCloser) Close() error {
	if closer, ok := rwc.r.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
func (rwc *readWriteCloser) Seek(offset int64, whence int) (int64, error) {
	return rwc.w.Seek(offset, whence)
}

// writeCloser adapts a Writer and Closer to ReadWriteCloser.
type writeCloser struct {
	w io.Writer
	c io.Closer
}

func (wc *writeCloser) Read(p []byte) (int, error)  { return 0, fmt.Errorf("read not supported") }
func (wc *writeCloser) Write(p []byte) (int, error) { return wc.w.Write(p) }
func (wc *writeCloser) Close() error                { return wc.c.Close() }
func (wc *writeCloser) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := wc.c.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fmt.Errorf("seek not supported")
}

// fileWrapper adapts ReadWriteSeeker to ReadWriteCloser.
type fileWrapper struct {
	rws io.ReadWriteSeeker
}

func (fw *fileWrapper) Read(p []byte) (int, error)  { return fw.rws.Read(p) }
func (fw *fileWrapper) Write(p []byte) (int, error) { return fw.rws.Write(p) }
func (fw *fileWrapper) Seek(offset int64, whence int) (int64, error) {
	return fw.rws.Seek(offset, whence)
}
func (fw *fileWrapper) Close() error { return nil } // No-op for fileobj

// wrapCloser 判断给定的 ReadWriteSeeker 是否实现了 Closer，如果没有，则使用 fileWrapper 包装。
func wrapCloser(rws io.ReadWriteSeeker) io.Closer {
	if c, ok := rws.(io.Closer); ok {
		return c
	}
	return &fileWrapper{rws: rws}
}
