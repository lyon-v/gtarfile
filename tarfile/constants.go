package tarfile

const (
	NUL           = byte(0) // Null character
	BLOCKSIZE     = 512     // Length of processing blocks
	RECORDSIZE    = BLOCKSIZE * 20
	LENGTH_NAME   = 100 // Max length of filename
	LENGTH_LINK   = 100 // Max length of linkname
	LENGTH_PREFIX = 155 // Max length of prefix field
	GNU_MAGIC     = "ustar  \x00"
	POSIX_MAGIC   = "ustar\x00\x00"

	REGTYPE          = "0"    // Regular file
	AREGTYPE         = "\x00" // Regular file (old format)
	LNKTYPE          = "1"    // Hard link
	SYMTYPE          = "2"    // Symbolic link
	CHRTYPE          = "3"    // Character device
	BLKTYPE          = "4"    // Block device
	DIRTYPE          = "5"    // Directory
	FIFOTYPE         = "6"    // FIFO
	CONTTYPE         = "7"    // Contiguous file
	GNUTYPE_LONGNAME = "L"    // GNU long name
	GNUTYPE_LONGLINK = "K"    // GNU long link
	GNUTYPE_SPARSE   = "S"    // GNU sparse file
	XHDTYPE          = "x"    // POSIX.1-2001 extended header
	XGLTYPE          = "g"    // POSIX.1-2001 global header
	SOLARIS_XHDTYPE  = "X"    // Solaris extended header

	USTAR_FORMAT   = 0 // POSIX.1-1988 (ustar) format
	GNU_FORMAT     = 1 // GNU tar format
	PAX_FORMAT     = 2 // POSIX.1-2001 (pax) format
	DEFAULT_FORMAT = PAX_FORMAT

	ENCODING = "utf-8" // Default encoding
)

var (
	SUPPORTED_TYPES = []string{REGTYPE, AREGTYPE, LNKTYPE, SYMTYPE, DIRTYPE, FIFOTYPE, CONTTYPE, CHRTYPE, BLKTYPE, GNUTYPE_LONGNAME, GNUTYPE_LONGLINK, GNUTYPE_SPARSE}
	REGULAR_TYPES   = []string{REGTYPE, AREGTYPE, CONTTYPE, GNUTYPE_SPARSE}
	GNU_TYPES       = []string{GNUTYPE_LONGNAME, GNUTYPE_LONGLINK, GNUTYPE_SPARSE}
)
