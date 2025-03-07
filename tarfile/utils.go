package tarfile

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func nts(s []byte, encoding, errors string) string {
	p := bytes.IndexByte(s, NUL)
	if p != -1 {
		s = s[:p]
	}
	// TODO: Implement proper encoding/decoding based on encoding and errors
	return string(s)
}

func nti(s []byte) (int64, error) {
	if s[0] == 0x80 || s[0] == 0xFF {
		n := int64(0)
		for i := 1; i < len(s); i++ {
			n = (n << 8) + int64(s[i])
		}
		if s[0] == 0xFF {
			n = -n
		}
		return n, nil
	}
	str := strings.TrimSpace(nts(s, "ascii", "strict"))
	if str == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(str, 8, 64)
	if err != nil {
		return 0, NewInvalidHeaderError("invalid number field")
	}
	return n, nil
}

func itn(n int64, digits int, format int) ([]byte, error) {
	if 0 <= n && n < int64(math.Pow(8, float64(digits-1))) {
		octal := fmt.Sprintf("%0*o", digits-1, n)
		return append([]byte(octal), NUL), nil
	} else if format == GNU_FORMAT && -int64(math.Pow(256, float64(digits-1))) <= n && n < int64(math.Pow(256, float64(digits-1))) {
		buf := make([]byte, digits)
		if n >= 0 {
			buf[0] = 0x80
		} else {
			buf[0] = 0xFF
			n = -n
		}
		for i := digits - 1; i > 0; i-- {
			buf[i] = byte(n & 0xFF)
			n >>= 8
		}
		return buf, nil
	}
	return nil, fmt.Errorf("overflow in number field")
}

func stn(s string, length int, encoding string) []byte {
	b := []byte(s)
	if len(b) > length {
		b = b[:length]
	}
	return append(b, make([]byte, length-len(b))...)
}

func calcChecksum(buf []byte) int64 {
	unsigned := int64(256) // 8 spaces
	for i, b := range buf {
		if i >= 148 && i < 156 {
			continue
		}
		unsigned += int64(b)
	}
	return unsigned
}

// divmod returns the quotient and remainder of a divided by b.
// It operates on int64 to handle large file sizes and offsets.
func divmod(a, b int64) (int64, int64) {
	return a / b, a % b
}

// divmodInt is an optional variant for int types, if explicitly needed.
func divmodInt(a, b int) (int, int) {
	return a / b, a % b
}
