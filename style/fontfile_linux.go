//go:build linux

package style

import (
	"os"
	"syscall"
)

// readFontFile maps a font file read-only rather than reading it into the
// heap, as FreeType does: the parsed face keeps the bytes for as long as
// the process runs, and mapped they are the page cache's clean, shared
// pages — every app on the desktop that uses the face reads the same ones,
// and none of them counts it as its own memory (read in, Noto Sans and its
// bold were 1.8 MB of each app's). A package update replaces a font file
// with a new one rather than rewriting it, so the mapping stays valid. An
// empty file, or one that cannot be mapped, is read as before.
func readFontFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if n := st.Size(); n > 0 && n == int64(int(n)) {
		if b, err := syscall.Mmap(int(f.Fd()), 0, int(n), syscall.PROT_READ, syscall.MAP_SHARED); err == nil {
			return b, nil
		}
	}
	return os.ReadFile(path)
}
