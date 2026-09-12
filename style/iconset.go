package style

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/codemodify/paintengine2d"
)

// IconSetInfo is a chrome icon set listed by Settings.
type IconSetInfo struct {
	Name   IconSetName
	Label  string
	Source string // "builtin" (classic/sharp) or "user" (XDG files)
}

// IconsDir is $XDG_CONFIG_HOME/uitoolkit/icons
// (or ~/.config/uitoolkit/icons).
func IconsDir() string {
	return filepath.Join(ConfigDir(), "icons")
}

// IconSetDir is icons/<name>/.
func IconSetDir(name IconSetName) string {
	return filepath.Join(IconsDir(), string(ParseIconSet(string(name))))
}

// SanitizeIconSetName lowercases and accepts [a-z][a-z0-9_-]{0,63}.
func SanitizeIconSetName(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	if s == "" || s == "." || s == ".." {
		return "", fmt.Errorf("icon set name is required")
	}
	if strings.ContainsAny(s, `/\:`) {
		return "", fmt.Errorf("icon set name cannot contain path separators")
	}
	if !themeNameRe.MatchString(s) {
		return "", fmt.Errorf("icon set name must start with a letter and use only lowercase letters, digits, hyphen, or underscore")
	}
	return s, nil
}

// IsFileIconSet reports whether name is loaded from icons/<name>/*.svg
// (not the drawn classic/sharp fallbacks).
func IsFileIconSet(name IconSetName) bool {
	switch ParseIconSet(string(name)) {
	case IconSetClassic, IconSetSharp:
		return false
	default:
		return true
	}
}

// FallbackIcons is the drawn set used when a file glyph is missing.
// File sets fall back to classic.
func FallbackIcons(name IconSetName) IconSetName {
	if ParseIconSet(string(name)) == IconSetSharp {
		return IconSetSharp
	}
	return IconSetClassic
}

// ListIconSets returns drawn builtins (classic, sharp) then installed
// file sets from ~/.config/uitoolkit/icons/* (sorted). A directory
// counts if it contains at least one ToolIcon SVG.
func ListIconSets() []IconSetInfo {
	out := []IconSetInfo{
		{Name: IconSetClassic, Label: "Classic  · builtin", Source: "builtin"},
		{Name: IconSetSharp, Label: "Sharp  · builtin", Source: "builtin"},
	}
	entries, err := os.ReadDir(IconsDir())
	if err != nil {
		return out
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name, err := SanitizeIconSetName(e.Name())
		if err != nil {
			continue
		}
		if name == string(IconSetClassic) || name == string(IconSetSharp) {
			continue
		}
		if !iconSetHasGlyphs(filepath.Join(IconsDir(), e.Name())) {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, n := range names {
		out = append(out, IconSetInfo{
			Name:   IconSetName(n),
			Label:  n + "  · installed",
			Source: "user",
		})
	}
	return out
}

func iconSetHasGlyphs(dir string) bool {
	for _, icon := range AllToolIcons() {
		if _, err := os.Stat(filepath.Join(dir, ToolIconFileName(icon))); err == nil {
			return true
		}
	}
	return false
}

type fileIconCache struct {
	mu    sync.Mutex
	docs  map[string]svgDoc // key: abs path
	miss  map[string]bool
	mtime map[string]int64
}

var iconsCache = &fileIconCache{
	docs:  map[string]svgDoc{},
	miss:  map[string]bool{},
	mtime: map[string]int64{},
}

func resetIconCache() {
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	iconsCache.docs = map[string]svgDoc{}
	iconsCache.miss = map[string]bool{}
	iconsCache.mtime = map[string]int64{}
}

func loadFileIcon(set IconSetName, icon ToolIcon) (svgDoc, bool) {
	name := ToolIconFileName(icon)
	if name == "" || !IsFileIconSet(set) {
		return svgDoc{}, false
	}
	path := filepath.Join(IconSetDir(set), name)
	st, err := os.Stat(path)
	if err != nil {
		return svgDoc{}, false
	}
	mod := st.ModTime().UnixNano()
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	if iconsCache.miss[path] && iconsCache.mtime[path] == mod {
		return svgDoc{}, false
	}
	if doc, ok := iconsCache.docs[path]; ok && iconsCache.mtime[path] == mod {
		return doc, true
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		iconsCache.miss[path] = true
		iconsCache.mtime[path] = mod
		return svgDoc{}, false
	}
	doc, err := parseSVG(raw)
	if err != nil || doc.empty() {
		iconsCache.miss[path] = true
		iconsCache.mtime[path] = mod
		return svgDoc{}, false
	}
	delete(iconsCache.miss, path)
	iconsCache.docs[path] = doc
	iconsCache.mtime[path] = mod
	return doc, true
}

// DrawFileToolIcon paints a tinted SVG from the named file set.
// It returns false when the file is missing or empty so the caller
// can fall back to a drawn classic/sharp glyph.
func DrawFileToolIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color, set IconSetName) bool {
	doc, ok := loadFileIcon(set, icon)
	if !ok {
		return false
	}
	doc.draw(ctx, b, col)
	return true
}
