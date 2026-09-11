package platform

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultINCRThreshold = 16 * 1024
	defaultINCRChunk     = 16 * 1024
	minINCRThreshold     = 64
)

// INCRThreshold is the X11 selection size that switches to the ICCCM
// incremental (INCR) protocol. UITK_X11_INCR_THRESHOLD overrides this
// (bytes). maxRequestBytes is XMaxRequestSize*4 when known.
func INCRThreshold(maxRequestBytes int) int {
	if s := strings.TrimSpace(os.Getenv("UITK_X11_INCR_THRESHOLD")); s != "" {
		n, err := strconv.Atoi(s)
		if err == nil && n >= minINCRThreshold {
			return n
		}
	}
	if maxRequestBytes > 4096 {
		t := maxRequestBytes / 4
		if t < defaultINCRThreshold {
			return t
		}
	}
	return defaultINCRThreshold
}

// INCRChunkSize is one incremental property payload.
func INCRChunkSize(threshold int) int {
	if threshold < 256 {
		return 256
	}
	if threshold > defaultINCRChunk {
		return defaultINCRChunk
	}
	return threshold
}

// INCRChunks splits data into INCR property-sized pieces. The last
// chunk may be empty (ICCCM end-of-transfer is an empty property, which
// the caller sends after the last non-empty chunk).
func INCRChunks(data []byte, chunk int) [][]byte {
	if chunk < 1 {
		chunk = defaultINCRChunk
	}
	if len(data) == 0 {
		return [][]byte{nil}
	}
	var out [][]byte
	for off := 0; off < len(data); off += chunk {
		end := off + chunk
		if end > len(data) {
			end = len(data)
		}
		out = append(out, data[off:end])
	}
	return out
}

// AppendINCRPiece concatenates one received INCR property. Empty piece
// means the transfer is complete.
func AppendINCRPiece(dst, piece []byte) (out []byte, done bool) {
	if len(piece) == 0 {
		if dst == nil {
			return []byte{}, true
		}
		return dst, true
	}
	return append(dst, piece...), false
}
