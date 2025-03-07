package tarfile

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// TarInfo represents metadata about a single tar archive member.
type TarInfo struct {
	Name       string            // Name of the archive member
	Mode       int64             // Permission bits
	UID        int               // User ID
	GID        int               // Group ID
	Size       int64             // Size in bytes
	Mtime      time.Time         // Modification time
	Chksum     int               // Header checksum
	Type       string            // File type (e.g., REGTYPE, DIRTYPE)
	Linkname   string            // Target file name for links
	Uname      string            // User name
	Gname      string            // Group name
	DevMajor   int               // Device major number
	DevMinor   int               // Device minor number
	Offset     int64             // Offset of the header in the tar file
	OffsetData int64             // Offset of the data in the tar file
	PaxHeaders map[string]string // PAX extended header key-value pairs
	Sparse     [][2]int64        // Sparse file info: [offset, size]
	tarfile    *TarFile          // Reference to the containing TarFile (undocumented, deprecated)
}

// NewTarInfo creates a new TarInfo object with default values.
func NewTarInfo(name string) *TarInfo {
	return &TarInfo{
		Name:       name,
		Mode:       0644,
		UID:        0,
		GID:        0,
		Size:       0,
		Mtime:      time.Unix(0, 0),
		Chksum:     0,
		Type:       REGTYPE,
		Linkname:   "",
		Uname:      "",
		Gname:      "",
		DevMajor:   0,
		DevMinor:   0,
		Offset:     0,
		OffsetData: 0,
		PaxHeaders: make(map[string]string),
		Sparse:     nil,
	}
}

// Path returns the name (alias for PAX "path").
func (ti *TarInfo) Path() string {
	return ti.Name
}

// SetPath sets the name (alias for PAX "path").
func (ti *TarInfo) SetPath(name string) {
	ti.Name = name
}

// Linkpath returns the linkname (alias for PAX "linkpath").
func (ti *TarInfo) Linkpath() string {
	return ti.Linkname
}

// SetLinkpath sets the linkname (alias for PAX "linkpath").
func (ti *TarInfo) SetLinkpath(linkname string) {
	ti.Linkname = linkname
}

// String returns a string representation of the TarInfo.
func (ti *TarInfo) String() string {
	return fmt.Sprintf("<%s %q at %p>", "TarInfo", ti.Name, ti)
}

// Replace creates a copy of the TarInfo with specified fields replaced.
func (ti *TarInfo) Replace(name, linkname, uname, gname *string, mtime *time.Time, mode, uid, gid *int64) *TarInfo {
	result := *ti
	result.PaxHeaders = make(map[string]string)
	for k, v := range ti.PaxHeaders {
		result.PaxHeaders[k] = v
	}
	if name != nil {
		result.Name = *name
	}
	if linkname != nil {
		result.Linkname = *linkname
	}
	if uname != nil {
		result.Uname = *uname
	}
	if gname != nil {
		result.Gname = *gname
	}
	if mtime != nil {
		result.Mtime = *mtime
	}
	if mode != nil {
		result.Mode = *mode
	}
	if uid != nil {
		result.UID = int(*uid)
	}
	if gid != nil {
		result.GID = int(*gid)
	}
	return &result
}

// GetInfo returns the TarInfo's attributes as a map.
func (ti *TarInfo) GetInfo() map[string]interface{} {
	info := map[string]interface{}{
		"name":     ti.Name,
		"mode":     ti.Mode & 07777,
		"uid":      ti.UID,
		"gid":      ti.GID,
		"size":     ti.Size,
		"mtime":    ti.Mtime.Unix(),
		"chksum":   ti.Chksum,
		"type":     ti.Type,
		"linkname": ti.Linkname,
		"uname":    ti.Uname,
		"gname":    ti.Gname,
		"devmajor": ti.DevMajor,
		"devminor": ti.DevMinor,
	}
	if ti.Type == DIRTYPE && !strings.HasSuffix(info["name"].(string), "/") {
		info["name"] = info["name"].(string) + "/"
	}
	return info
}

// ToBuf converts the TarInfo to a 512-byte tar header block.
func (ti *TarInfo) ToBuf(format int, encoding, errors string) ([]byte, error) {
	info := ti.GetInfo()
	for k, v := range info {
		if v == nil {
			return nil, fmt.Errorf("%s may not be None", k)
		}
	}
	switch format {
	case USTAR_FORMAT:
		return ti.createUstarHeader(info, encoding, errors)
	case GNU_FORMAT:
		return ti.createGnuHeader(info, encoding, errors)
	case PAX_FORMAT:
		return ti.createPaxHeader(info, encoding)
	default:
		return nil, fmt.Errorf("invalid format")
	}
}

func (ti *TarInfo) createUstarHeader(info map[string]interface{}, encoding, errors string) ([]byte, error) {
	info["magic"] = POSIX_MAGIC
	if len(info["linkname"].(string)) > LENGTH_LINK {
		return nil, fmt.Errorf("linkname is too long")
	}
	if len(info["name"].(string)) > LENGTH_NAME {
		prefix, name, err := ti.posixSplitName(info["name"].(string), encoding, errors)
		if err != nil {
			return nil, err
		}
		info["prefix"] = prefix
		info["name"] = name
	}
	return ti.createHeader(info, USTAR_FORMAT, encoding, errors)
}

func (ti *TarInfo) createGnuHeader(info map[string]interface{}, encoding, errors string) ([]byte, error) {
	info["magic"] = GNU_MAGIC
	buf := []byte{}
	if len(info["linkname"].(string)) > LENGTH_LINK {
		longLink, err := ti.createGnuLongHeader(info["linkname"].(string), GNUTYPE_LONGLINK, encoding, errors)
		if err != nil {
			return nil, err
		}
		buf = append(buf, longLink...)
	}
	if len(info["name"].(string)) > LENGTH_NAME {
		longName, err := ti.createGnuLongHeader(info["name"].(string), GNUTYPE_LONGNAME, encoding, errors)
		if err != nil {
			return nil, err
		}
		buf = append(buf, longName...)
	}
	header, err := ti.createHeader(info, GNU_FORMAT, encoding, errors)
	if err != nil {
		return nil, err
	}
	return append(buf, header...), nil
}

func (ti *TarInfo) createPaxHeader(info map[string]interface{}, encoding string) ([]byte, error) {
	info["magic"] = POSIX_MAGIC
	paxHeaders := make(map[string]string)
	for k, v := range ti.PaxHeaders {
		paxHeaders[k] = v
	}

	// 定义字段映射
	fields := [][3]interface{}{
		{"name", "path", LENGTH_NAME},
		{"linkname", "linkpath", LENGTH_LINK},
		{"uname", "uname", 32},
		{"gname", "gname", 32},
	}

	// 遍历字段映射
	for _, field := range fields {
		name := field[0].(string)
		hname := field[1].(string)
		length := field[2].(int)

		n := info[name].(string)
		if _, ok := paxHeaders[hname]; ok {
			continue
		}
		// 检查是否为纯数字（模拟 Python 的 ASCII 检查）
		if _, err := strconv.ParseUint(n, 10, 64); err == nil {
			paxHeaders[hname] = n
			continue
		}
		if len(n) > length {
			paxHeaders[hname] = n
		}
	}

	// 处理数字字段
	for name, digits := range map[string]int{
		"uid":   8,
		"gid":   8,
		"size":  12,
		"mtime": 12,
	} {
		if name == "mtime" {
			continue // Handle mtime separately
		}
		val := info[name].(int)
		if val < 0 || val >= int(math.Pow(8, float64(digits-1))) {
			info[name] = 0
			if _, ok := paxHeaders[name]; !ok {
				paxHeaders[name] = strconv.Itoa(val)
			}
		}
	}
	// Handle mtime as int64
	mtime := info["mtime"].(int64)
	if mtime < 0 || mtime >= int64(math.Pow(8, 11)) {
		info["mtime"] = int64(0)
		if _, ok := paxHeaders["mtime"]; !ok {
			paxHeaders["mtime"] = strconv.FormatInt(mtime, 10)
		}
	}

	var buf []byte
	if len(paxHeaders) > 0 {
		paxBuf, err := ti.createPaxGenericHeader(paxHeaders, XHDTYPE, encoding)
		if err != nil {
			return nil, err
		}
		buf = paxBuf
	}
	header, err := ti.createHeader(info, USTAR_FORMAT, "ascii", "replace")
	if err != nil {
		return nil, err
	}
	return append(buf, header...), nil
}
func (ti *TarInfo) posixSplitName(name, encoding, errors string) (string, string, error) {
	components := strings.Split(name, "/")
	for i := 1; i < len(components); i++ {
		prefix := strings.Join(components[:i], "/")
		rest := strings.Join(components[i:], "/")
		if len(prefix) <= LENGTH_PREFIX && len(rest) <= LENGTH_NAME {
			return prefix, rest, nil
		}
	}
	return "", "", fmt.Errorf("name is too long")
}

func (ti *TarInfo) createHeader(info map[string]interface{}, format int, encoding, errors string) ([]byte, error) {
	hasDeviceFields := info["type"] == CHRTYPE || info["type"] == BLKTYPE
	var devMajor, devMinor []byte
	var err error
	if hasDeviceFields {
		devMajor, err = itn(int64(info["devmajor"].(int)), 8, format)
		if err != nil {
			return nil, err
		}
		devMinor, err = itn(int64(info["devminor"].(int)), 8, format)
		if err != nil {
			return nil, err
		}
	} else {
		devMajor = stn("", 8, encoding)
		devMinor = stn("", 8, encoding)
	}

	filetype := info["type"].(string)
	parts := make([][]byte, 15) // 预分配 15 个元素，与字段数一致
	parts[0] = stn(info["name"].(string), 100, encoding)

	// mode
	parts[1], err = itn(info["mode"].(int64), 8, format)
	if err != nil {
		return nil, fmt.Errorf("mode field failed: %v", err)
	}

	// uid
	parts[2], err = itn(int64(info["uid"].(int)), 8, format)
	if err != nil {
		return nil, fmt.Errorf("uid field failed: %v", err)
	}

	// gid
	parts[3], err = itn(int64(info["gid"].(int)), 8, format)
	if err != nil {
		return nil, fmt.Errorf("gid field failed: %v", err)
	}

	// size
	parts[4], err = itn(info["size"].(int64), 12, format)
	if err != nil {
		return nil, fmt.Errorf("size field failed: %v", err)
	}

	// mtime
	parts[5], err = itn(info["mtime"].(int64), 12, format)
	if err != nil {
		return nil, fmt.Errorf("mtime field failed: %v", err)
	}

	parts[6] = []byte("        ") // checksum placeholder (8 spaces)
	parts[7] = []byte(filetype)
	parts[8] = stn(info["linkname"].(string), 100, encoding)
	parts[9] = []byte(info["magic"].(string))
	parts[10] = stn(info["uname"].(string), 32, encoding)
	parts[11] = stn(info["gname"].(string), 32, encoding)
	parts[12] = devMajor
	parts[13] = devMinor
	parts[14] = stn(info["prefix"].(string), 155, encoding)

	// 检查 nil 值
	for i := 1; i < 6; i++ {
		if parts[i] == nil {
			return nil, fmt.Errorf("field %d is nil", i)
		}
	}

	buf := bytes.NewBuffer(nil)
	for _, part := range parts {
		buf.Write(part)
	}
	for buf.Len() < BLOCKSIZE {
		buf.WriteByte(NUL)
	}
	b := buf.Bytes()
	chksum := calcChecksum(b)
	// 修正 checksum 格式：6位八进制数 + NUL + 空格
	checksumBytes := fmt.Sprintf("%06o\x00 ", chksum)
	b = append(b[:148], []byte(checksumBytes)...)
	b = append(b, buf.Bytes()[156:]...)
	return b[:BLOCKSIZE], nil
}
func (ti *TarInfo) createGnuLongHeader(name, typ, encoding, errors string) ([]byte, error) {
	nameBytes := append([]byte(name), NUL)
	info := map[string]interface{}{
		"name":  "././@LongLink",
		"type":  typ,
		"size":  int64(len(nameBytes)),
		"magic": GNU_MAGIC,
	}
	header, err := ti.createHeader(info, USTAR_FORMAT, encoding, errors)
	if err != nil {
		return nil, err
	}
	payload := ti.createPayload(nameBytes)
	return append(header, payload...), nil
}

func (ti *TarInfo) createPaxGenericHeader(paxHeaders map[string]string, typ, encoding string) ([]byte, error) {
	binary := false
	for _, v := range paxHeaders {
		if _, err := strconv.ParseUint(v, 10, 64); err != nil {
			binary = true
			break
		}
	}

	records := []byte{}
	if binary {
		records = append(records, []byte("21 hdrcharset=BINARY\n")...)
	}

	for k, v := range paxHeaders {
		kBytes := []byte(k)
		var vBytes []byte
		if binary {
			vBytes = []byte(v)
		} else {
			vBytes = []byte(v)
		}
		l := len(kBytes) + len(vBytes) + 3 // " " + "=" + "\n"
		n := 0
		for {
			p := l + len(strconv.Itoa(n))
			if p == n {
				break
			}
			n = p
		}
		records = append(records, []byte(fmt.Sprintf("%d %s=%s\n", n, k, v))...)
	}

	info := map[string]interface{}{
		"name":  "././@PaxHeader",
		"type":  typ,
		"size":  int64(len(records)),
		"magic": POSIX_MAGIC,
	}
	header, err := ti.createHeader(info, USTAR_FORMAT, "ascii", "replace")
	if err != nil {
		return nil, err
	}
	payload := ti.createPayload(records)
	return append(header, payload...), nil
}

func (ti *TarInfo) createPayload(payload []byte) []byte {
	_, remainder := divmodInt(len(payload), BLOCKSIZE)
	if remainder > 0 {
		payload = append(payload, make([]byte, BLOCKSIZE-remainder)...)
	}
	return payload
}

// FromTarFile reads a TarInfo from the TarFile's current position.
func (ti *TarInfo) FromTarFile(tf *TarFile) (*TarInfo, error) {
	buf := make([]byte, BLOCKSIZE)
	n, err := tf.FileObj.Read(buf)
	if err != nil {
		if err == io.EOF && n == 0 {
			return nil, NewEOFHeaderError("end of file header")
		}
		return nil, NewTruncatedHeaderError("truncated header")
	}
	if n != BLOCKSIZE {
		return nil, NewTruncatedHeaderError("truncated header")
	}

	ti, err = FromBuf(buf, tf.Encoding, tf.Errors)
	if err != nil {
		return nil, err
	}
	ti.Offset = tf.Offset
	ti.OffsetData = tf.Offset + BLOCKSIZE
	tf.Offset += BLOCKSIZE
	return ti, nil
}

// CreatePaxGlobalHeader creates a PAX global header from headers.
func (ti *TarInfo) CreatePaxGlobalHeader(headers map[string]string) ([]byte, error) {
	return ti.createPaxGenericHeader(headers, XGLTYPE, "ascii")
}

// FromBuf constructs a TarInfo from a 512-byte buffer.
func FromBuf(buf []byte, encoding, errors string) (*TarInfo, error) {
	if len(buf) == 0 {
		return nil, NewEmptyHeaderError("empty header")
	}
	if len(buf) != BLOCKSIZE {
		return nil, NewTruncatedHeaderError("truncated header")
	}
	if bytes.Count(buf, []byte{NUL}) == BLOCKSIZE {
		return nil, NewEOFHeaderError("end of file header")
	}

	chksum, err := nti(buf[148:156])
	if err != nil {
		return nil, err
	}
	if chksum != calcChecksum(buf) {
		return nil, NewInvalidHeaderError("bad checksum")
	}

	ti := NewTarInfo("")
	ti.Name = nts(buf[0:100], encoding, errors)

	// Mode
	mode, err := nti(buf[100:108])
	if err != nil {
		return nil, err
	}
	ti.Mode = mode

	// UID
	uid, err := nti(buf[108:116])
	if err != nil {
		return nil, err
	}
	ti.UID = int(uid)

	// GID
	gid, err := nti(buf[116:124])
	if err != nil {
		return nil, err
	}
	ti.GID = int(gid)

	// Size
	size, err := nti(buf[124:136])
	if err != nil {
		return nil, err
	}
	ti.Size = size

	// Mtime
	mtime, err := nti(buf[136:148])
	if err != nil {
		return nil, err
	}
	ti.Mtime = time.Unix(mtime, 0)

	ti.Chksum = int(chksum)
	ti.Type = string(buf[156:157])
	ti.Linkname = nts(buf[157:257], encoding, errors)
	ti.Uname = nts(buf[265:297], encoding, errors)
	ti.Gname = nts(buf[297:329], encoding, errors)

	// DevMajor
	devMajor, err := nti(buf[329:337])
	if err != nil {
		return nil, err
	}
	ti.DevMajor = int(devMajor)

	// DevMinor
	devMinor, err := nti(buf[337:345])
	if err != nil {
		return nil, err
	}
	ti.DevMinor = int(devMinor)

	prefix := nts(buf[345:500], encoding, errors)

	if ti.Type == AREGTYPE && strings.HasSuffix(ti.Name, "/") {
		ti.Type = DIRTYPE
	}
	if ti.Type == GNUTYPE_SPARSE {
		var structs [][2]int64
		pos := 386
		for i := 0; i < 4; i++ {
			offset, err := nti(buf[pos : pos+12])
			if err != nil {
				return nil, err
			}
			numbytes, err := nti(buf[pos+12 : pos+24])
			if err != nil {
				return nil, err
			}
			if offset == 0 && numbytes == 0 {
				break
			}
			structs = append(structs, [2]int64{offset, numbytes})
			pos += 24
		}
		isExtended := buf[482] != 0
		origSize, err := nti(buf[483:495])
		if err != nil {
			return nil, err
		}
		if len(structs) > 0 || isExtended {
			ti.Sparse = structs
			if isExtended {
				// TODO: Handle extended sparse headers
			}
			ti.Size = origSize
		}
	}

	if ti.IsDir() {
		ti.Name = strings.TrimSuffix(ti.Name, "/")
	}
	if prefix != "" && !contains(ti.Type, GNU_TYPES) {
		ti.Name = prefix + "/" + ti.Name
	}
	return ti, nil
}

// IsReg returns true if the TarInfo represents a regular file.
func (ti *TarInfo) IsReg() bool {
	return contains(ti.Type, REGULAR_TYPES)
}

// IsFile returns true if the TarInfo represents a regular file (alias for IsReg).
func (ti *TarInfo) IsFile() bool {
	return ti.IsReg()
}

// IsDir returns true if the TarInfo represents a directory.
func (ti *TarInfo) IsDir() bool {
	return ti.Type == DIRTYPE
}

// IsSym returns true if the TarInfo represents a symbolic link.
func (ti *TarInfo) IsSym() bool {
	return ti.Type == SYMTYPE
}

// IsLnk returns true if the TarInfo represents a hard link.
func (ti *TarInfo) IsLnk() bool {
	return ti.Type == LNKTYPE
}

// IsChr returns true if the TarInfo represents a character device.
func (ti *TarInfo) IsChr() bool {
	return ti.Type == CHRTYPE
}

// IsBlk returns true if the TarInfo represents a block device.
func (ti *TarInfo) IsBlk() bool {
	return ti.Type == BLKTYPE
}

// IsFifo returns true if the TarInfo represents a FIFO.
func (ti *TarInfo) IsFifo() bool {
	return ti.Type == FIFOTYPE
}

// IsSparse returns true if the TarInfo represents a sparse file.
func (ti *TarInfo) IsSparse() bool {
	return ti.Sparse != nil
}

// IsDev returns true if the TarInfo represents a device (character, block, or FIFO).
func (ti *TarInfo) IsDev() bool {
	return ti.Type == CHRTYPE || ti.Type == BLKTYPE || ti.Type == FIFOTYPE
}

// Helper function to check if a string is in a slice.
func contains(s string, slice []string) bool {
	for _, v := range slice {
		if s == v {
			return true
		}
	}
	return false
}
