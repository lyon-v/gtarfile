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
	"sync"
	"syscall"
	"time"

	"github.com/ulikunitz/xz" // 引入第三方 xz 包

	"golang.org/x/sys/unix"
)

// TarFile provides an interface to tar archives.
type TarFile struct {
	// 私有字段，提供更好的封装
	debug            int                                      // Debug level (0 to 3)
	dereference      bool                                     // Follow symlinks if true
	ignoreZeros      bool                                     // Skip empty/invalid blocks if true
	errorLevel       int                                      // Error reporting level
	format           int                                      // Archive format (DEFAULT_FORMAT, USTAR_FORMAT, etc.)
	encoding         string                                   // Encoding for 8-bit strings
	errors           string                                   // Error handler for unicode conversion
	tarInfo          func() *TarInfo                          // Factory for TarInfo objects
	fileObject       func(*TarFile, *TarInfo) *ExFileObject   // Factory for file objects
	extractionFilter func(*TarInfo, string) (*TarInfo, error) // Filter for extraction

	name       string             // Path to the tar file
	mode       string             // "r", "a", "w", "x"
	fileMode   string             // Underlying file mode ("rb", "r+b", etc.)
	fileObj    io.ReadWriteSeeker // File object for reading/writing
	stream     bool               // Treat as a stream if true
	extFileObj bool               // True if FileObj is externally provided
	paxHeaders map[string]string  // PAX headers

	copyBufSize int                  // Buffer size for copying
	closed      bool                 // Whether the archive is closed
	members     []*TarInfo           // List of members
	loaded      bool                 // Whether all members are loaded
	offset      int64                // Current position in the archive
	inodes      map[[2]uint64]string // Cache of inodes for hard links
	firstMember *TarInfo             // First member for iteration

	// 添加互斥锁保证并发安全
	mu sync.RWMutex
}

// NewTarFile initializes a new TarFile instance.
func NewTarFile(name, mode string, fileobj io.ReadWriteSeeker, opts ...TarFileOption) (*TarFile, error) {
	modes := map[string]string{"r": "rb", "a": "r+b", "w": "wb", "x": "xb"}
	fileMode, ok := modes[mode]
	if !ok {
		return nil, fmt.Errorf("mode must be 'r', 'a', 'w' or 'x'")
	}

	tf := &TarFile{
		debug:       0,
		dereference: false,
		ignoreZeros: false,
		errorLevel:  1,
		format:      DEFAULT_FORMAT,
		encoding:    ENCODING,
		errors:      "surrogateescape",
		tarInfo:     func() *TarInfo { return NewTarInfo("") },
		fileObject:  func(tf *TarFile, ti *TarInfo) *ExFileObject { return NewExFileObject(tf, ti) },
		paxHeaders:  make(map[string]string),
		mode:        mode,
		fileMode:    fileMode,
		inodes:      make(map[[2]uint64]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(tf)
	}

	if fileobj == nil {
		if tf.mode == "a" && !fileExists(name) {
			tf.mode = "w"
			tf.fileMode = "wb"
		}
		f, err := os.OpenFile(name, osMode(tf.fileMode), 0666)
		if err != nil {
			return nil, err
		}
		tf.fileObj = f
		tf.extFileObj = false
	} else {
		tf.fileObj = fileobj
		tf.extFileObj = true
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
		tf.name = abs
	}

	tf.offset = tell(tf.fileObj)

	// Initialize based on mode
	var err error
	switch tf.mode {
	case "r":
		tf.firstMember, err = tf.Next()
		if err != nil {
			tf.Close()
			return nil, err
		}
	case "a":
		for {
			if _, err := tf.fileObj.Seek(tf.offset, io.SeekStart); err != nil {
				tf.Close()
				return nil, err
			}
			ti, err := tf.tarInfo().FromTarFile(tf)
			if err != nil {
				if _, ok := err.(*EOFHeaderError); ok {
					if _, err := tf.fileObj.Seek(tf.offset, io.SeekStart); err != nil {
						tf.Close()
						return nil, err
					}
					break
				}
				tf.Close()
				return nil, NewReadError(err.Error())
			}
			tf.members = append(tf.members, ti)
		}
	case "w", "x":
		tf.loaded = true
		if len(tf.paxHeaders) > 0 {
			buf, err := tf.tarInfo().CreatePaxGlobalHeader(tf.paxHeaders)
			if err != nil {
				tf.Close()
				return nil, err
			}
			if _, err := tf.fileObj.Write(buf); err != nil {
				tf.Close()
				return nil, err
			}
			tf.offset += int64(len(buf))
		}
	}

	return tf, nil
}

// TarFileOption defines options for NewTarFile.
type TarFileOption func(*TarFile)

// WithFormat sets the archive format.
func WithFormat(format int) TarFileOption {
	return func(tf *TarFile) { tf.format = format }
}

// WithEncoding sets the encoding.
func WithEncoding(encoding string) TarFileOption {
	return func(tf *TarFile) { tf.encoding = encoding }
}

// WithErrors sets the error handler.
func WithErrors(errors string) TarFileOption {
	return func(tf *TarFile) { tf.errors = errors }
}

// WithPaxHeaders sets the PAX headers.
func WithPaxHeaders(headers map[string]string) TarFileOption {
	return func(tf *TarFile) { tf.paxHeaders = headers }
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
		tf, err := NewTarFile(name, filemode, stream, append(opts, func(tf *TarFile) { tf.stream = true })...)
		if err != nil {
			stream.Close()
			return nil, err
		}
		tf.extFileObj = false
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
	if tf.closed {
		return nil
	}
	tf.closed = true
	defer func() {
		if !tf.extFileObj {
			if f, ok := tf.fileObj.(*os.File); ok {
				f.Close()
			}
		}
	}()

	if tf.mode == "a" || tf.mode == "w" || tf.mode == "x" {
		_, err := tf.fileObj.Write(make([]byte, BLOCKSIZE*2)) // Two zero blocks
		if err != nil {
			return err
		}
		tf.offset += BLOCKSIZE * 2
		_, remainder := divmod(tf.offset, RECORDSIZE)
		if remainder > 0 {
			_, err = tf.fileObj.Write(make([]byte, RECORDSIZE-remainder))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetMember returns a TarInfo object for the named member.
func (tf *TarFile) GetMember(name string) (*TarInfo, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	tf.check("r")
	tarinfo := tf.getMember(name)
	if tarinfo == nil {
		return nil, fmt.Errorf("member %q not found", name)
	}
	return tarinfo, nil
}

// GetMembers returns all members as a list of TarInfo objects.
func (tf *TarFile) GetMembers() ([]*TarInfo, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	tf.check("")
	if !tf.loaded {
		tf.load()
	}
	// 返回副本避免外部修改
	result := make([]*TarInfo, len(tf.members))
	copy(result, tf.members)
	return result, nil
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

	ti := tf.tarInfo()
	var stat syscall.Stat_t
	if fileobj == nil {
		if tf.dereference {
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
		if !tf.dereference && stat.Nlink > 1 && tf.inodes[inode] != "" && arcname != tf.inodes[inode] {
			ti.Type = LNKTYPE
			linkname = tf.inodes[inode]
		} else {
			ti.Type = REGTYPE
			if stat.Ino != 0 {
				tf.inodes[inode] = arcname
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
	if tf.name != "" && filepath.Clean(name) == tf.name {
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
	buf, err := ti.ToBuf(tf.format, tf.encoding, tf.errors)
	if err != nil {
		return err
	}
	if _, err := tf.fileObj.Write(buf); err != nil {
		return err
	}
	tf.offset += int64(len(buf))

	if fileobj != nil {
		if _, err := io.CopyN(tf.fileObj, fileobj, ti.Size); err != nil {
			return err
		}
		blocks, remainder := divmod(ti.Size, BLOCKSIZE)
		if remainder > 0 {
			_, err := tf.fileObj.Write(make([]byte, BLOCKSIZE-remainder))
			if err != nil {
				return err
			}
			blocks++
		}
		tf.offset += blocks * BLOCKSIZE
	}

	tf.members = append(tf.members, ti)
	return nil
}

// Next returns the next member of the archive.
func (tf *TarFile) Next() (*TarInfo, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	return tf.next()
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
	if !tf.stream {
		for {
			ti, err := tf.next() // 调用内部方法，不获取锁
			if err != nil {
				break // 或根据错误类型处理
			}
			if ti == nil {
				break
			}
		}
		tf.loaded = true
	}
}

func (tf *TarFile) check(mode string) error {
	if tf.closed {
		return fmt.Errorf("TarFile is closed")
	}
	if mode != "" && !strings.Contains(mode, tf.mode) {
		return fmt.Errorf("bad operation for mode %q", tf.mode)
	}
	return nil
}

func (tf *TarFile) dbg(level int, msg string) {
	if level <= tf.debug {
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

// 公开的访问器方法，提供并发安全的字段访问

// GetName returns the name of the tar file
func (tf *TarFile) GetName() string {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.name
}

// GetMode returns the mode of the tar file
func (tf *TarFile) GetMode() string {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.mode
}

// GetDebug returns the debug level
func (tf *TarFile) GetDebug() int {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.debug
}

// SetDebug sets the debug level
func (tf *TarFile) SetDebug(level int) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.debug = level
}

// GetDereference returns the dereference setting
func (tf *TarFile) GetDereference() bool {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.dereference
}

// SetDereference sets the dereference setting
func (tf *TarFile) SetDereference(dereference bool) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.dereference = dereference
}

// GetIgnoreZeros returns the ignore zeros setting
func (tf *TarFile) GetIgnoreZeros() bool {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.ignoreZeros
}

// SetIgnoreZeros sets the ignore zeros setting
func (tf *TarFile) SetIgnoreZeros(ignoreZeros bool) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.ignoreZeros = ignoreZeros
}

// GetErrorLevel returns the error level
func (tf *TarFile) GetErrorLevel() int {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.errorLevel
}

// SetErrorLevel sets the error level
func (tf *TarFile) SetErrorLevel(level int) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.errorLevel = level
}

// GetFormat returns the archive format
func (tf *TarFile) GetFormat() int {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.format
}

// SetFormat sets the archive format
func (tf *TarFile) SetFormat(format int) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.format = format
}

// GetEncoding returns the encoding
func (tf *TarFile) GetEncoding() string {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.encoding
}

// SetEncoding sets the encoding
func (tf *TarFile) SetEncoding(encoding string) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.encoding = encoding
}

// GetErrors returns the error handler
func (tf *TarFile) GetErrors() string {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.errors
}

// SetErrors sets the error handler
func (tf *TarFile) SetErrors(errors string) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.errors = errors
}

// GetPaxHeaders returns a copy of the PAX headers
func (tf *TarFile) GetPaxHeaders() map[string]string {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	headers := make(map[string]string)
	for k, v := range tf.paxHeaders {
		headers[k] = v
	}
	return headers
}

// SetPaxHeaders sets the PAX headers
func (tf *TarFile) SetPaxHeaders(headers map[string]string) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	tf.paxHeaders = make(map[string]string)
	for k, v := range headers {
		tf.paxHeaders[k] = v
	}
}

// IsClosed returns whether the archive is closed
func (tf *TarFile) IsClosed() bool {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.closed
}

// IsLoaded returns whether all members are loaded
func (tf *TarFile) IsLoaded() bool {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.loaded
}

// GetOffset returns the current position in the archive
func (tf *TarFile) GetOffset() int64 {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.offset
}

// IsStream returns whether the archive is treated as a stream
func (tf *TarFile) IsStream() bool {
	tf.mu.RLock()
	defer tf.mu.RUnlock()
	return tf.stream
}

// next is the internal implementation without locking (assumes lock is held)
func (tf *TarFile) next() (*TarInfo, error) {
	tf.check("ra")
	if tf.firstMember != nil {
		m := tf.firstMember
		tf.firstMember = nil
		return m, nil
	}

	if tf.offset != tell(tf.fileObj) {
		if tf.offset == 0 {
			return nil, nil
		}
		if _, err := tf.fileObj.Seek(tf.offset-1, io.SeekStart); err != nil {
			return nil, err
		}
		b := make([]byte, 1)
		if _, err := tf.fileObj.Read(b); err != nil {
			return nil, NewReadError("unexpected end of data")
		}
	}

	var tarinfo *TarInfo
	for {
		ti, err := tf.tarInfo().FromTarFile(tf)
		if err != nil {
			switch e := err.(type) {
			case *EOFHeaderError:
				if tf.ignoreZeros {
					tf.dbg(2, fmt.Sprintf("0x%X: %s", tf.offset, e))
					tf.offset += BLOCKSIZE
					continue
				}
			case *InvalidHeaderError:
				if tf.ignoreZeros {
					tf.dbg(2, fmt.Sprintf("0x%X: %s", tf.offset, e))
					tf.offset += BLOCKSIZE
					continue
				}
				if tf.offset == 0 {
					return nil, NewReadError(e.Error())
				}
			case *EmptyHeaderError:
				if tf.offset == 0 {
					return nil, NewReadError("empty file")
				}
			case *TruncatedHeaderError:
				if tf.offset == 0 {
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

	if tarinfo != nil && !tf.stream {
		tf.members = append(tf.members, tarinfo)
	} else {
		tf.loaded = true
	}
	return tarinfo, nil
}

// Extract extracts a member from the archive to the specified path
func (tf *TarFile) Extract(member *TarInfo, path string) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if err := tf.check("r"); err != nil {
		return err
	}

	return tf.extractMember(member, path)
}

// ExtractAll extracts all members from the archive to the specified path
func (tf *TarFile) ExtractAll(path string) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if err := tf.check("r"); err != nil {
		return err
	}

	members, err := tf.getMembers()
	if err != nil {
		return err
	}

	for _, member := range members {
		if err := tf.extractMember(member, path); err != nil {
			return fmt.Errorf("failed to extract %s: %w", member.Name, err)
		}
	}

	return nil
}

// extractMember is the internal implementation for extracting a member
func (tf *TarFile) extractMember(member *TarInfo, basePath string) error {
	targetPath := filepath.Join(basePath, member.Name)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	switch {
	case member.IsDir():
		return os.MkdirAll(targetPath, os.FileMode(member.Mode))

	case member.IsReg():
		return tf.extractFile(member, targetPath)

	case member.IsSym():
		return os.Symlink(member.Linkname, targetPath)

	case member.IsLnk():
		linkTarget := filepath.Join(basePath, member.Linkname)
		return os.Link(linkTarget, targetPath)

	default:
		// 对于设备文件、FIFO等，我们暂时跳过
		tf.dbg(1, fmt.Sprintf("Skipping special file %s (type: %s)", member.Name, member.Type))
		return nil
	}
}

// extractFile extracts a regular file
func (tf *TarFile) extractFile(member *TarInfo, targetPath string) error {
	// 移动到数据的开始位置
	if _, err := tf.fileObj.Seek(member.OffsetData, io.SeekStart); err != nil {
		return err
	}

	// 创建目标文件
	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(member.Mode))
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 复制数据
	_, err = io.CopyN(outFile, tf.fileObj, member.Size)
	if err != nil {
		return err
	}

	// 设置修改时间
	return os.Chtimes(targetPath, member.Mtime, member.Mtime)
}

// getMembers is the internal implementation without locking
func (tf *TarFile) getMembers() ([]*TarInfo, error) {
	if !tf.loaded {
		tf.load()
	}
	return tf.members, nil
}

// extractTo is a convenience method that extracts a named member
func (tf *TarFile) ExtractTo(memberName, targetPath string) error {
	member, err := tf.GetMember(memberName)
	if err != nil {
		return err
	}
	return tf.Extract(member, targetPath)
}
