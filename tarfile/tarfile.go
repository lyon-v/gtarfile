package tarfile

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/ulikunitz/xz" // 引入第三方 xz 包

	"golang.org/x/sys/unix"
)

// TarFile provides an interface to tar archives.
type TarFile struct {
	Debug            int                                      // Debug level (0 to 3)
	Dereference      bool                                     // Follow symlinks if true
	IgnoreZeros      bool                                     // Skip empty/invalid blocks if true
	ErrorLevel       int                                      // Error reporting level
	Format           int                                      // Archive format (DEFAULT_FORMAT, USTAR_FORMAT, etc.)
	Encoding         string                                   // Encoding for 8-bit strings
	Errors           string                                   // Error handler for unicode conversion
	TarInfo          func() *TarInfo                          // Factory for TarInfo objects
	FileObject       func(*TarFile, *TarInfo) *ExFileObject   // Factory for file objects
	ExtractionFilter func(*TarInfo, string) (*TarInfo, error) // Filter for extraction

	Name       string             // Path to the tar file
	Mode       string             // "r", "a", "w", "x"
	fileMode   string             // Underlying file mode ("rb", "r+b", etc.)
	FileObj    io.ReadWriteSeeker // File object for reading/writing
	Stream     bool               // Treat as a stream if true
	ExtFileObj bool               // True if FileObj is externally provided
	PaxHeaders map[string]string  // PAX headers

	CopyBufSize int                  // Buffer size for copying
	Closed      bool                 // Whether the archive is closed
	Members     []*TarInfo           // List of members
	Loaded      bool                 // Whether all members are loaded
	Offset      int64                // Current position in the archive
	Inodes      map[[2]uint64]string // Cache of inodes for hard links
	FirstMember *TarInfo             // First member for iteration
}

// NewTarFile initializes a new TarFile instance.
func NewTarFile(name, mode string, fileobj io.ReadWriteSeeker, opts ...TarFileOption) (*TarFile, error) {
	modes := map[string]string{"r": "rb", "a": "r+b", "w": "wb", "x": "xb"}
	fileMode, ok := modes[mode]
	if !ok {
		return nil, fmt.Errorf("mode must be 'r', 'a', 'w' or 'x'")
	}

	tf := &TarFile{
		Debug:       0,
		Dereference: false,
		IgnoreZeros: false,
		ErrorLevel:  1,
		Format:      DEFAULT_FORMAT,
		Encoding:    ENCODING,
		Errors:      "surrogateescape",
		TarInfo:     func() *TarInfo { return NewTarInfo("") },
		FileObject:  func(tf *TarFile, ti *TarInfo) *ExFileObject { return NewExFileObject(tf, ti) },
		PaxHeaders:  make(map[string]string),
		Mode:        mode,
		fileMode:    fileMode,
	}

	// Apply options
	for _, opt := range opts {
		opt(tf)
	}

	if fileobj == nil {
		if tf.Mode == "a" && !fileExists(name) {
			tf.Mode = "w"
			tf.fileMode = "wb"
		}
		f, err := os.OpenFile(name, osMode(tf.fileMode), 0666)
		if err != nil {
			return nil, err
		}
		tf.FileObj = f
		tf.ExtFileObj = false
	} else {
		tf.FileObj = fileobj
		tf.ExtFileObj = true
		if name == "" {
			if n, ok := fileobj.(interface{ Name() string }); ok {
				name = n.Name()
			}
		}
	}
	if name != "" {
		abs, err := filepath.Abs(name)
		if err != nil {
			return nil, err
		}
		tf.Name = abs
	}

	tf.Offset = tell(tf.FileObj)
	tf.Inodes = make(map[[2]uint64]string)

	// Initialize based on mode
	var err error
	switch tf.Mode {
	case "r":
		tf.FirstMember, err = tf.Next()
		if err != nil {
			tf.Close()
			return nil, err
		}
	case "a":
		for {
			if _, err := tf.FileObj.Seek(tf.Offset, io.SeekStart); err != nil {
				tf.Close()
				return nil, err
			}
			ti, err := tf.TarInfo().FromTarFile(tf)
			if err != nil {
				if _, ok := err.(*EOFHeaderError); ok {
					if _, err := tf.FileObj.Seek(tf.Offset, io.SeekStart); err != nil {
						tf.Close()
						return nil, err
					}
					break
				}
				tf.Close()
				return nil, NewReadError(err.Error())
			}
			tf.Members = append(tf.Members, ti)
		}
	case "w", "x":
		tf.Loaded = true
		if len(tf.PaxHeaders) > 0 {
			buf, err := tf.TarInfo().CreatePaxGlobalHeader(tf.PaxHeaders)
			if err != nil {
				tf.Close()
				return nil, err
			}
			if _, err := tf.FileObj.Write(buf); err != nil {
				tf.Close()
				return nil, err
			}
			tf.Offset += int64(len(buf))
		}
	}

	return tf, nil
}

// TarFileOption defines options for NewTarFile.
type TarFileOption func(*TarFile)

// WithFormat sets the archive format.
func WithFormat(format int) TarFileOption {
	return func(tf *TarFile) { tf.Format = format }
}

// WithEncoding sets the encoding.
func WithEncoding(encoding string) TarFileOption {
	return func(tf *TarFile) { tf.Encoding = encoding }
}

// WithErrors sets the error handler.
func WithErrors(errors string) TarFileOption {
	return func(tf *TarFile) { tf.Errors = errors }
}

// WithPaxHeaders sets the PAX headers.
func WithPaxHeaders(headers map[string]string) TarFileOption {
	return func(tf *TarFile) { tf.PaxHeaders = headers }
}

// Open opens a tar archive with the specified mode and compression.
func Open(name, mode string, fileobj io.ReadWriteSeeker, bufsize int, opts ...TarFileOption) (*TarFile, error) {
	if name == "" && fileobj == nil {
		return nil, fmt.Errorf("nothing to open")
	}

	switch {
	case mode == "r" || mode == "r:*":
		for _, comptype := range []string{"tar", "gz", "bz2", "xz"} {
			f, err := openMethod(comptype, name, "r", fileobj, opts...)
			if err == nil {
				return f, nil
			}
			if fileobj != nil {
				if _, err := fileobj.Seek(0, io.SeekStart); err != nil {
					return nil, err
				}
			}
		}
		return nil, NewReadError("file could not be opened successfully")

	case strings.Contains(mode, ":"):
		filemode, comptype := splitMode(mode, ":")
		return openMethod(comptype, name, filemode, fileobj, opts...)

	case strings.Contains(mode, "|"):
		filemode, comptype := splitMode(mode, "|")
		if filemode != "r" && filemode != "w" {
			return nil, fmt.Errorf("mode must be 'r' or 'w'")
		}
		stream, err := newStream(name, filemode, comptype, fileobj, bufsize, 9)
		if err != nil {
			return nil, err
		}
		tf, err := NewTarFile(name, filemode, stream, append(opts, func(tf *TarFile) { tf.Stream = true })...)
		if err != nil {
			stream.Close()
			return nil, err
		}
		tf.ExtFileObj = false
		return tf, nil

	case mode == "a" || mode == "w" || mode == "x":
		return NewTarFile(name, mode, fileobj, opts...)
	}

	return nil, fmt.Errorf("undiscernible mode")
}

func splitMode(mode, sep string) (string, string) {
	parts := strings.SplitN(mode, sep, 2)
	filemode := parts[0]
	if filemode == "" {
		filemode = "r"
	}
	comptype := parts[1]
	if comptype == "" {
		comptype = "tar"
	}
	return filemode, comptype
}

func openMethod(comptype, name, mode string, fileobj io.ReadWriteSeeker, opts ...TarFileOption) (*TarFile, error) {
	switch comptype {
	case "tar":
		return NewTarFile(name, mode, fileobj, opts...)
	case "gz":
		var f io.ReadWriteSeeker
		if fileobj != nil {
			gz, err := gzip.NewReader(fileobj)
			if err != nil {
				return nil, err
			}
			f = &readWriteSeeker{gz, fileobj}
		} else {
			f, _ = os.Open(name) // Simplified, needs proper gzip handling
		}
		return NewTarFile(name, mode, f, opts...)
	case "bz2":
		f := bzip2.NewReader(fileobj)
		return NewTarFile(name, mode, &readWriteSeeker{f, fileobj}, opts...)
	case "xz":
		f, err := xz.NewReader(fileobj)
		if err != nil {
			return nil, err
		}
		return NewTarFile(name, mode, &readWriteSeeker{f, fileobj}, opts...)
	default:
		return nil, NewCompressionError(fmt.Sprintf("unknown compression type %q", comptype))
	}
}

// readWriteSeeker adapts a Reader to ReadWriteSeeker (simplified).
type readWriteSeeker struct {
	r io.Reader
	w io.ReadWriteSeeker
}

func (rws *readWriteSeeker) Read(p []byte) (int, error)  { return rws.r.Read(p) }
func (rws *readWriteSeeker) Write(p []byte) (int, error) { return 0, fmt.Errorf("write not supported") }
func (rws *readWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	return rws.w.Seek(offset, whence)
}

// Close closes the TarFile.
func (tf *TarFile) Close() error {
	if tf.Closed {
		return nil
	}
	tf.Closed = true
	defer func() {
		if !tf.ExtFileObj {
			if f, ok := tf.FileObj.(*os.File); ok {
				f.Close()
			}
		}
	}()

	if tf.Mode == "a" || tf.Mode == "w" || tf.Mode == "x" {
		_, err := tf.FileObj.Write(make([]byte, BLOCKSIZE*2)) // Two zero blocks
		if err != nil {
			return err
		}
		tf.Offset += BLOCKSIZE * 2
		_, remainder := divmod(tf.Offset, RECORDSIZE)
		if remainder > 0 {
			_, err = tf.FileObj.Write(make([]byte, RECORDSIZE-remainder))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetMember returns a TarInfo object for the named member.
func (tf *TarFile) GetMember(name string) (*TarInfo, error) {
	tf.check("")
	ti := tf.getMember(strings.TrimSuffix(name, "/"))
	if ti == nil {
		return nil, fmt.Errorf("filename %q not found", name)
	}
	return ti, nil
}

// GetMembers returns all members as a list of TarInfo objects.
func (tf *TarFile) GetMembers() ([]*TarInfo, error) {
	tf.check("")
	if !tf.Loaded {
		tf.load()
	}
	return tf.Members, nil
}

// GetNames returns the names of all members.
func (tf *TarFile) GetNames() ([]string, error) {
	members, err := tf.GetMembers()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(members))
	for i, m := range members {
		names[i] = m.Name
	}
	return names, nil
}

// GetTarInfo creates a TarInfo object from a file.
func (tf *TarFile) GetTarInfo(name, arcname string, fileobj *os.File) (*TarInfo, error) {
	tf.check("awx")
	if fileobj != nil {
		name = fileobj.Name()
	}
	if arcname == "" {
		arcname = name
	}
	arcname = strings.ReplaceAll(arcname, string(os.PathSeparator), "/")
	arcname = strings.TrimPrefix(arcname, "/")

	ti := tf.TarInfo()
	var stat syscall.Stat_t
	if fileobj == nil {
		if tf.Dereference {
			err := syscall.Stat(name, &stat)
			if err != nil {
				return nil, err
			}
		} else {
			err := syscall.Lstat(name, &stat)
			if err != nil {
				return nil, err
			}
		}
	} else {
		err := syscall.Fstat(int(fileobj.Fd()), &stat)
		if err != nil {
			return nil, err
		}
	}

	linkname := ""
	inode := [2]uint64{stat.Ino, stat.Dev} // 改为 uint64
	switch {
	case stat.Mode&syscall.S_IFMT == syscall.S_IFREG:
		if !tf.Dereference && stat.Nlink > 1 && tf.Inodes[inode] != "" && arcname != tf.Inodes[inode] {
			ti.Type = LNKTYPE
			linkname = tf.Inodes[inode]
		} else {
			ti.Type = REGTYPE
			if stat.Ino != 0 {
				tf.Inodes[inode] = arcname
			}
		}
	case stat.Mode&syscall.S_IFMT == syscall.S_IFDIR:
		ti.Type = DIRTYPE
	case stat.Mode&syscall.S_IFMT == syscall.S_IFIFO:
		ti.Type = FIFOTYPE
	case stat.Mode&syscall.S_IFMT == syscall.S_IFLNK:
		ti.Type = SYMTYPE
		l, err := os.Readlink(name)
		if err != nil {
			return nil, err
		}
		linkname = l
	case stat.Mode&syscall.S_IFMT == syscall.S_IFCHR:
		ti.Type = CHRTYPE
	case stat.Mode&syscall.S_IFMT == syscall.S_IFBLK:
		ti.Type = BLKTYPE
	default:
		return nil, nil
	}

	ti.Name = arcname
	ti.Mode = int64(stat.Mode & 07777)
	ti.UID = int(stat.Uid)
	ti.GID = int(stat.Gid)
	if ti.Type == REGTYPE {
		ti.Size = stat.Size
	} else {
		ti.Size = 0
	}
	ti.Mtime = time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)
	ti.Linkname = linkname
	// TODO: Set uname and gname using system calls if available
	if ti.Type == CHRTYPE || ti.Type == BLKTYPE {
		ti.DevMajor = int(unix.Major(uint64(stat.Rdev)))
		ti.DevMinor = int(unix.Minor(uint64(stat.Rdev)))
	}
	return ti, nil
}

// Add adds a file to the archive.
func (tf *TarFile) Add(name, arcname string, recursive bool, filter func(*TarInfo) (*TarInfo, error)) error {
	tf.check("awx")
	if arcname == "" {
		arcname = name
	}
	if tf.Name != "" && filepath.Clean(name) == tf.Name {
		tf.dbg(2, fmt.Sprintf("tarfile: Skipped %q", name))
		return nil
	}
	tf.dbg(1, name)

	ti, err := tf.GetTarInfo(name, arcname, nil)
	if err != nil {
		return err
	}
	if ti == nil {
		tf.dbg(1, fmt.Sprintf("tarfile: Unsupported type %q", name))
		return nil
	}

	if filter != nil {
		ti, err = filter(ti)
		if err != nil {
			return err
		}
		if ti == nil {
			tf.dbg(2, fmt.Sprintf("tarfile: Excluded %q", name))
			return nil
		}
	}

	if ti.IsReg() {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		return tf.AddFile(ti, f)
	} else if ti.IsDir() {
		if err := tf.AddFile(ti, nil); err != nil {
			return err
		}
		if recursive {
			files, err := os.ReadDir(name)
			if err != nil {
				return err
			}
			sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
			for _, fi := range files {
				err := tf.Add(filepath.Join(name, fi.Name()), filepath.Join(arcname, fi.Name()), recursive, filter)
				if err != nil {
					return err
				}
			}
		}
	} else {
		return tf.AddFile(ti, nil)
	}
	return nil
}

// AddFile adds a TarInfo object to the archive.
func (tf *TarFile) AddFile(tarinfo *TarInfo, fileobj io.Reader) error {
	tf.check("awx")
	if fileobj == nil && tarinfo.IsReg() && tarinfo.Size != 0 {
		return fmt.Errorf("fileobj not provided for non zero-size regular file")
	}

	ti := tarinfo // Shallow copy in Go (struct is copied)
	buf, err := ti.ToBuf(tf.Format, tf.Encoding, tf.Errors)
	if err != nil {
		return err
	}
	if _, err := tf.FileObj.Write(buf); err != nil {
		return err
	}
	tf.Offset += int64(len(buf))

	if fileobj != nil {
		if _, err := io.CopyN(tf.FileObj, fileobj, ti.Size); err != nil {
			return err
		}
		blocks, remainder := divmod(ti.Size, BLOCKSIZE)
		if remainder > 0 {
			_, err := tf.FileObj.Write(make([]byte, BLOCKSIZE-remainder))
			if err != nil {
				return err
			}
			blocks++
		}
		tf.Offset += blocks * BLOCKSIZE
	}

	tf.Members = append(tf.Members, ti)
	return nil
}

// Next returns the next member of the archive.
func (tf *TarFile) Next() (*TarInfo, error) {
	tf.check("ra")
	if tf.FirstMember != nil {
		m := tf.FirstMember
		tf.FirstMember = nil
		return m, nil
	}

	if tf.Offset != tell(tf.FileObj) {
		if tf.Offset == 0 {
			return nil, nil
		}
		if _, err := tf.FileObj.Seek(tf.Offset-1, io.SeekStart); err != nil {
			return nil, err
		}
		b := make([]byte, 1)
		if _, err := tf.FileObj.Read(b); err != nil {
			return nil, NewReadError("unexpected end of data")
		}
	}

	var tarinfo *TarInfo
	for {
		ti, err := tf.TarInfo().FromTarFile(tf)
		if err != nil {
			switch e := err.(type) {
			case *EOFHeaderError:
				if tf.IgnoreZeros {
					tf.dbg(2, fmt.Sprintf("0x%X: %s", tf.Offset, e))
					tf.Offset += BLOCKSIZE
					continue
				}
			case *InvalidHeaderError:
				if tf.IgnoreZeros {
					tf.dbg(2, fmt.Sprintf("0x%X: %s", tf.Offset, e))
					tf.Offset += BLOCKSIZE
					continue
				}
				if tf.Offset == 0 {
					return nil, NewReadError(e.Error())
				}
			case *EmptyHeaderError:
				if tf.Offset == 0 {
					return nil, NewReadError("empty file")
				}
			case *TruncatedHeaderError:
				if tf.Offset == 0 {
					return nil, NewReadError(e.Error())
				}
			case *SubsequentHeaderError:
				return nil, NewReadError(e.Error())
			default:
				return nil, err
			}
		}
		tarinfo = ti
		break
	}

	if tarinfo != nil && !tf.Stream {
		tf.Members = append(tf.Members, tarinfo)
	} else {
		tf.Loaded = true
	}
	return tarinfo, nil
}

// Helper methods

func (tf *TarFile) getMember(name string) *TarInfo {
	members, _ := tf.GetMembers()
	for i := len(members) - 1; i >= 0; i-- {
		m := members[i]
		if name == m.Name {
			return m
		}
	}
	return nil
}

func (tf *TarFile) load() {
	if !tf.Stream {
		for {
			ti, err := tf.Next()
			if err != nil {
				break // 或根据错误类型处理
			}
			if ti == nil {
				break
			}
		}
		tf.Loaded = true
	}
}

func (tf *TarFile) check(mode string) error {
	if tf.Closed {
		return fmt.Errorf("TarFile is closed")
	}
	if mode != "" && !strings.Contains(mode, tf.Mode) {
		return fmt.Errorf("bad operation for mode %q", tf.Mode)
	}
	return nil
}

func (tf *TarFile) dbg(level int, msg string) {
	if level <= tf.Debug {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
}

// Utility functions

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func osMode(mode string) int {
	switch mode {
	case "rb":
		return os.O_RDONLY
	case "r+b":
		return os.O_RDWR
	case "wb":
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case "xb":
		return os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	return 0
}

func tell(r io.Seeker) int64 {
	pos, _ := r.Seek(0, io.SeekCurrent)
	return pos
}
