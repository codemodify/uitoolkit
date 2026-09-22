//go:build !linux

package style

import "os"

// readFontFile reads a font file into memory (fontfile_linux.go maps it).
func readFontFile(path string) ([]byte, error) { return os.ReadFile(path) }
