// Package fonts embeds the OFL UI typefaces shipped with uitoolkit.
//
//	Titillium Web  — default UI (labels, buttons, menus, fields)
//	JetBrains Mono — mono / code (Inspector, file preview)
package fonts

import (
	"embed"
	"fmt"
)

//go:embed TitilliumWeb-Regular.ttf TitilliumWeb-Bold.ttf JetBrainsMono-Regular.ttf JetBrainsMono-Bold.ttf
var files embed.FS

const (
	FileTitilliumRegular = "TitilliumWeb-Regular.ttf"
	FileTitilliumBold    = "TitilliumWeb-Bold.ttf"
	FileJetBrainsRegular = "JetBrainsMono-Regular.ttf"
	FileJetBrainsBold    = "JetBrainsMono-Bold.ttf"

	FamilyUI   = "Titillium Web"
	FamilyMono = "JetBrains Mono"
)

// Bytes returns an embedded TTF. The slice is safe to keep (embed is immutable).
func Bytes(name string) ([]byte, error) {
	b, err := files.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("fonts: missing %s: %w", name, err)
	}
	if len(b) < 16 {
		return nil, fmt.Errorf("fonts: %s is empty or truncated (%d bytes)", name, len(b))
	}
	return b, nil
}

func TitilliumRegular() ([]byte, error) { return Bytes(FileTitilliumRegular) }
func TitilliumBold() ([]byte, error)    { return Bytes(FileTitilliumBold) }
func JetBrainsRegular() ([]byte, error) { return Bytes(FileJetBrainsRegular) }
func JetBrainsBold() ([]byte, error)    { return Bytes(FileJetBrainsBold) }
