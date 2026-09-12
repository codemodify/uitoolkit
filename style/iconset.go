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
	Source ThemeSource // builtin (drawn + premiere names) or user
}

// PremiereIconSets are the five shipped PNG families. After a manual
// copy into ~/.config/uitoolkit/icons/<name>/ they still list as
// Built-in in Settings. Any other folder is User.
var PremiereIconSets = []IconSetName{
	IconSetLucide,
	IconSetPhosphor,
	IconSetTabler,
	IconSetHeroicons,
	IconSetMaterialSymbols,
}

// IsPremiereIconSet reports whether name is one of the five shipped
// premiere PNG families (lucide, phosphor, tabler, heroicons,
// material-symbols).
func IsPremiereIconSet(name IconSetName) bool {
	switch ParseIconSet(string(name)) {
	case IconSetLucide, IconSetPhosphor, IconSetTabler, IconSetHeroicons, IconSetMaterialSymbols:
		return true
	default:
		return false
	}
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

// IsFileIconSet reports whether name is loaded from icons/<name>/*.png
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

func iconSetDisplay(name string) string {
	switch name {
	case "lucide":
		return "Lucide"
	case "phosphor":
		return "Phosphor"
	case "tabler":
		return "Tabler"
	case "heroicons":
		return "Heroicons"
	case "material-symbols":
		return "Material Symbols"
	default:
		return name
	}
}

func listInstalledIconDirs() map[string]string {
	out := map[string]string{}
	entries, err := os.ReadDir(IconsDir())
	if err != nil {
		return out
	}
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
		out[name] = e.Name()
	}
	return out
}

// ListBuiltinIconSets is drawn classic/sharp plus premiere PNG families
// that are present under icons/.
func ListBuiltinIconSets() []IconSetInfo {
	out := []IconSetInfo{
		{Name: IconSetClassic, Label: "Classic", Source: ThemeSourceBuiltin},
		{Name: IconSetSharp, Label: "Sharp", Source: ThemeSourceBuiltin},
	}
	installed := listInstalledIconDirs()
	for _, n := range PremiereIconSets {
		if _, ok := installed[string(n)]; ok {
			out = append(out, IconSetInfo{
				Name:   n,
				Label:  iconSetDisplay(string(n)),
				Source: ThemeSourceBuiltin,
			})
		}
	}
	return out
}

// ListUserIconSets is every icons/<name>/ folder that is not a premiere
// set (and not the drawn classic/sharp names).
func ListUserIconSets() []IconSetInfo {
	installed := listInstalledIconDirs()
	var names []string
	for n := range installed {
		if IsPremiereIconSet(IconSetName(n)) {
			continue
		}
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]IconSetInfo, 0, len(names))
	for _, n := range names {
		out = append(out, IconSetInfo{
			Name:   IconSetName(n),
			Label:  iconSetDisplay(n),
			Source: ThemeSourceUser,
		})
	}
	return out
}

// DeleteUserIconSet removes icons/<name>/ from disk. Drawn classic/sharp
// and premiere set names cannot be deleted this way.
func DeleteUserIconSet(name IconSetName) error {
	clean, err := SanitizeIconSetName(string(name))
	if err != nil {
		return err
	}
	if clean == string(IconSetClassic) || clean == string(IconSetSharp) || IsPremiereIconSet(IconSetName(clean)) {
		return fmt.Errorf("not a user icon set: %s", clean)
	}
	installed := listInstalledIconDirs()
	dirName, ok := installed[clean]
	if !ok {
		return fmt.Errorf("not a user icon set: %s", clean)
	}
	return os.RemoveAll(filepath.Join(IconsDir(), dirName))
}

// ListIconSets returns Built-in (drawn classic/sharp, then premiere
// names when present) followed by User folders, sorted. A directory
// counts if it contains at least one ToolIcon PNG (24 or @2x).
func ListIconSets() []IconSetInfo {
	builtins := ListBuiltinIconSets()
	users := ListUserIconSets()
	out := make([]IconSetInfo, 0, len(builtins)+len(users))
	out = append(out, builtins...)
	out = append(out, users...)
	return out
}

func iconSetHasGlyphs(dir string) bool {
	for _, icon := range AllToolIcons() {
		if _, err := os.Stat(filepath.Join(dir, ToolIconFileName(icon))); err == nil {
			return true
		}
		if _, err := os.Stat(filepath.Join(dir, ToolIconHiDPIFileName(icon))); err == nil {
			return true
		}
	}
	return false
}

type fileIconCache struct {
	mu    sync.Mutex
	imgs  map[string]*paintengine2d.Image // key: abs path
	miss  map[string]bool
	mtime map[string]int64
}

var iconsCache = &fileIconCache{
	imgs:  map[string]*paintengine2d.Image{},
	miss:  map[string]bool{},
	mtime: map[string]int64{},
}

func resetIconCache() {
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	iconsCache.imgs = map[string]*paintengine2d.Image{}
	iconsCache.miss = map[string]bool{}
	iconsCache.mtime = map[string]int64{}
}

func loadFileIcon(set IconSetName, icon ToolIcon, destW float32) (*paintengine2d.Image, bool) {
	if !IsFileIconSet(set) {
		return nil, false
	}
	dir := IconSetDir(set)
	for _, name := range toolIconFileCandidates(icon, destW) {
		if img, ok := loadPNGIcon(filepath.Join(dir, name)); ok {
			return img, true
		}
	}
	return nil, false
}

func loadPNGIcon(path string) (*paintengine2d.Image, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	mod := st.ModTime().UnixNano()
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	if iconsCache.miss[path] && iconsCache.mtime[path] == mod {
		return nil, false
	}
	if img, ok := iconsCache.imgs[path]; ok && iconsCache.mtime[path] == mod {
		return img, true
	}
	raw, err := paintengine2d.DecodePNGFile(path)
	if err != nil || raw == nil || raw.Width < 1 || raw.Height < 1 {
		iconsCache.miss[path] = true
		iconsCache.mtime[path] = mod
		return nil, false
	}
	img := toWhiteMask(raw)
	delete(iconsCache.miss, path)
	iconsCache.imgs[path] = img
	iconsCache.mtime[path] = mod
	return img, true
}

// toWhiteMask turns a monochrome/alpha PNG into a white premul atlas
// so DrawImageRectPaint can tint it with the Look foreground (same
// currentColor intent as the old SVGs).
func toWhiteMask(src *paintengine2d.Image) *paintengine2d.Image {
	w, h := src.Width, src.Height
	n := w * h
	transparent := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, _, _, a := src.PremulAt(x, y)
			if a < 24 {
				transparent++
			}
		}
	}
	useAlpha := n > 0 && transparent*20 > n // >5% clear → trust alpha
	out := paintengine2d.NewImage(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := src.PremulAt(x, y)
			var cov uint8
			if useAlpha {
				cov = a
			} else {
				lum := (uint32(r) + uint32(g) + uint32(b)) / 3
				if a > 0 {
					lum = lum * 255 / uint32(a)
					if lum > 255 {
						lum = 255
					}
				}
				cov = uint8(255 - lum)
			}
			i := y*out.RowStride() + x*4
			out.Pix[i+0] = cov
			out.Pix[i+1] = cov
			out.Pix[i+2] = cov
			out.Pix[i+3] = cov
		}
	}
	out.Touch()
	return out
}

// DrawFileToolIcon paints a tinted PNG from the named file set.
// It returns false when the file is missing or empty so the caller
// can fall back to a drawn classic/sharp glyph.
func DrawFileToolIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color, set IconSetName) bool {
	img, ok := loadFileIcon(set, icon, b.Dx())
	if !ok {
		return false
	}
	src := paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height))
	ctx.DrawImageRectPaint(img, src, b, paintengine2d.Paint{
		Color:  col,
		Filter: paintengine2d.FilterBilinear,
	})
	return true
}
