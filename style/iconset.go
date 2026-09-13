package style

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/codemodify/paintengine2d"
)

//go:embed no-icon.png
var embeddedNoIcon24 []byte

//go:embed no-icon@2x.png
var embeddedNoIcon48 []byte

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

// IconSetDir is icons/<name>/. When a folder with a different spelling
// (case, spaces) canonicalizes to name, that folder is returned.
func IconSetDir(name IconSetName) string {
	set := string(ParseIconSet(string(name)))
	if dir, ok := iconDirIndex()[set]; ok {
		return filepath.Join(IconsDir(), dir)
	}
	return filepath.Join(IconsDir(), set)
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

// FallbackIcons is the drawn set used when a file set is not installed.
// An installed premiere / user folder never falls through to this — missing
// stems stay inside the same set (see loadFileIcon).
func FallbackIcons(name IconSetName) IconSetName {
	if ParseIconSet(string(name)) == IconSetSharp {
		return IconSetSharp
	}
	return IconSetClassic
}

func fileIconSetInstalled(set IconSetName) bool {
	if !IsFileIconSet(set) {
		return false
	}
	return iconSetHasGlyphs(IconSetDir(set))
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

// scanIconDirs reads icons/ and maps each canonical set id onto the folder
// that holds it, so a folder named "My Icons" both lists and loads as
// "my-icons" instead of listing as installed and then falling back to the
// embedded placeholder for every stem.
func scanIconDirs() map[string]string {
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
		if _, dup := out[name]; dup {
			continue // first (sorted) entry wins, deterministically
		}
		out[name] = e.Name()
	}
	return out
}

func listInstalledIconDirs() map[string]string { return scanIconDirs() }

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

// iconCacheTTL is how long a cached stat result is trusted. Within a frame
// nothing is re-stat'ed (a missing stem used to cost 8 stats per draw); a PNG
// edited on disk still shows up about a second later.
const iconCacheTTL = time.Second

// iconEntry caches one path. A nil img is a negative entry: the file is
// missing or undecodable, and it is not stat'ed again this generation.
type iconEntry struct {
	img     *paintengine2d.Image
	mod     int64
	size    int64
	checked time.Time
	gen     uint64
}

type fileIconCache struct {
	mu sync.Mutex
	// gen invalidates every cached path and directory listing at once.
	// Bumped by InvalidateIconCache (theme / look reload).
	gen      uint64
	entries  map[string]*iconEntry
	embedded map[string]*paintengine2d.Image
	dirs     map[string]string
	dirsGen  uint64
	dirsMod  int64
	dirsAt   time.Time
}

var iconsCache = &fileIconCache{
	entries:  map[string]*iconEntry{},
	embedded: map[string]*paintengine2d.Image{},
}

// IconGeneration is the current icon cache generation. It changes whenever
// cached icon files and directory listings are invalidated.
func IconGeneration() uint64 {
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	return iconsCache.gen
}

// InvalidateIconCache drops cached icon files and directory listings. Call it
// after the icon set changes on disk (the look watcher does).
func InvalidateIconCache() {
	iconsCache.mu.Lock()
	iconsCache.gen++
	iconsCache.entries = map[string]*iconEntry{}
	iconsCache.dirs = nil
	iconsCache.dirsMod = 0
	iconsCache.dirsAt = time.Time{}
	iconsCache.mu.Unlock()
	missingStemLog.mu.Lock()
	missingStemLog.seen = map[string]bool{}
	missingStemLog.mu.Unlock()
}

func resetIconCache() { InvalidateIconCache() }

// iconDirIndex maps a canonical set id onto its folder name. It is consulted
// on every icon draw, so it is cached and revalidated with a single stat of
// icons/ (whose mtime changes when a set folder is added, renamed, or
// removed) plus the cache generation.
func iconDirIndex() map[string]string {
	iconsCache.mu.Lock()
	gen := iconsCache.gen
	// Inside the TTL the index is reused without touching the filesystem,
	// so a frame full of icons costs no syscalls at all.
	if iconsCache.dirs != nil && iconsCache.dirsGen == gen && time.Since(iconsCache.dirsAt) < iconCacheTTL {
		dirs := iconsCache.dirs
		iconsCache.mu.Unlock()
		return dirs
	}
	iconsCache.mu.Unlock()

	var mod int64
	if st, err := os.Stat(IconsDir()); err == nil {
		mod = st.ModTime().UnixNano()
	}
	iconsCache.mu.Lock()
	if iconsCache.dirs != nil && iconsCache.dirsGen == gen && iconsCache.dirsMod == mod {
		dirs := iconsCache.dirs
		iconsCache.dirsAt = time.Now()
		iconsCache.mu.Unlock()
		return dirs
	}
	iconsCache.mu.Unlock()

	dirs := scanIconDirNames()

	iconsCache.mu.Lock()
	if iconsCache.gen == gen {
		iconsCache.dirs = dirs
		iconsCache.dirsGen = gen
		iconsCache.dirsMod = mod
		iconsCache.dirsAt = time.Now()
	}
	iconsCache.mu.Unlock()
	return dirs
}

// scanIconDirNames maps canonical set id → folder name without looking
// inside the folders (path resolution only).
func scanIconDirNames() map[string]string {
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
		if _, dup := out[name]; dup {
			continue // first (sorted) entry wins, deterministically
		}
		out[name] = e.Name()
	}
	return out
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
	for _, name := range stemFileCandidates(noIconStem, destW) {
		if img, ok := loadPNGIcon(filepath.Join(dir, name)); ok {
			logMissingStemOnce(set, icon, noIconStem)
			return img, true
		}
	}
	if img, ok := loadEmbeddedNoIcon(destW); ok {
		logMissingStemOnce(set, icon, noIconStem)
		return img, true
	}
	return nil, false
}

func loadEmbeddedNoIcon(destW float32) (*paintengine2d.Image, bool) {
	raw := embeddedNoIcon24
	key := "embed:no-icon.png"
	if destW > iconNative1x && len(embeddedNoIcon48) > 0 {
		raw = embeddedNoIcon48
		key = "embed:no-icon@2x.png"
	}
	if len(raw) == 0 {
		return nil, false
	}
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	if img, ok := iconsCache.embedded[key]; ok {
		return img, true
	}
	dec, err := paintengine2d.DecodePNG(bytes.NewReader(raw))
	if err != nil || dec == nil || dec.Width < 1 || dec.Height < 1 {
		return nil, false
	}
	img := toWhiteMask(dec)
	iconsCache.embedded[key] = img
	return img, true
}

var missingStemLog = struct {
	mu   sync.Mutex
	seen map[string]bool
}{seen: map[string]bool{}}

func logMissingStemOnce(set IconSetName, icon ToolIcon, used string) {
	want := ToolIconName(icon)
	if want == "" {
		want = fmt.Sprintf("icon-%d", int(icon))
	}
	key := string(set) + ":" + want
	missingStemLog.mu.Lock()
	defer missingStemLog.mu.Unlock()
	if missingStemLog.seen[key] {
		return
	}
	missingStemLog.seen[key] = true
	log.Printf("uitk icons: %s missing %s.png, using %s", set, want, used)
}

// loadPNGIcon decodes path into a white mask, caching hits and misses. A
// missing file costs one stat per TTL instead of one per draw.
func loadPNGIcon(path string) (*paintengine2d.Image, bool) {
	c := iconsCache
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if e, ok := c.entries[path]; ok && e.gen == c.gen && now.Sub(e.checked) < iconCacheTTL {
		return e.img, e.img != nil
	}
	st, err := os.Stat(path)
	if err != nil {
		c.entries[path] = &iconEntry{checked: now, gen: c.gen}
		return nil, false
	}
	mod, size := st.ModTime().UnixNano(), st.Size()
	if e, ok := c.entries[path]; ok && e.gen == c.gen && e.mod == mod && e.size == size {
		e.checked = now
		return e.img, e.img != nil
	}
	raw, err := paintengine2d.DecodePNGFile(path)
	if err != nil || raw == nil || raw.Width < 1 || raw.Height < 1 {
		c.entries[path] = &iconEntry{mod: mod, size: size, checked: now, gen: c.gen}
		return nil, false
	}
	img := toWhiteMask(raw)
	c.entries[path] = &iconEntry{img: img, mod: mod, size: size, checked: now, gen: c.gen}
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
// Lookup is exact stem, then documented aliases, then no-icon (pack
// file or the embedded placeholder). It never paints a drawn classic
// scribble. False only if even the embedded placeholder cannot decode.
func DrawFileToolIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color, set IconSetName) bool {
	img, ok := loadFileIcon(set, icon, b.Dx())
	if !ok {
		return false
	}
	return paintTintedIcon(ctx, b, img, col)
}

func paintTintedIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, img *paintengine2d.Image, col paintengine2d.Color) bool {
	if ctx == nil || img == nil || b.Empty() {
		return false
	}
	src := paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height))
	ctx.DrawImageRectPaint(img, src, b, paintengine2d.Paint{
		Color:  col,
		Filter: paintengine2d.FilterBilinear,
	})
	return true
}

func drawEmbeddedNoIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, col paintengine2d.Color) bool {
	img, ok := loadEmbeddedNoIcon(b.Dx())
	if !ok {
		return false
	}
	return paintTintedIcon(ctx, b, img, col)
}
