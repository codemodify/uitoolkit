package style

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codemodify/paintengine2d"
)

// The icon themes the desktop already has.
//
// A uitoolkit icon set used to be one of two things: a set the toolkit
// draws in process (classic, sharp), or a folder of PNGs named after the
// toolkit's own actions under ~/.config/uitoolkit/icons/<set>/. This file
// adds the third, which is the one a Linux machine actually has dozens
// of: a freedesktop icon theme, under /usr/share/icons/<Theme>/ or
// ~/.local/share/icons/<Theme>/, with an index.theme that says what sizes
// it ships and what it inherits.
//
// Nothing is copied and nothing is vendored. The theme is read where the
// distribution put it, the same files Plasma and GNOME are drawing from,
// and the toolkit asks it for the freedesktop *name* of each of its own
// actions ([ToolIconThemeNames]): Save is "document-save", Copy is
// "edit-copy", the picture is whatever the theme has under that name at
// the wanted size.
//
// Two things are deliberately not done here.
//
// The first is the icon cache: index.theme is parsed, but
// icon-theme.cache (the mmap'd GTK hash) is not, because the point of
// that file is to save stat calls at start-up in a process that loads
// hundreds of icons, and this toolkit asks for sixteen.
//
// The second is colour management of somebody else's artwork. A themed
// icon is drawn in the pixels the theme drew, except when the file is
// monochrome — one colour and an alpha channel, which is what every
// symbolic set and the whole of Breeze is — and then it is tinted with
// the look's foreground, exactly as the toolkit's own PNG sets are. That
// is what the desktops do, and it is the only rule under which Oxygen's
// full-colour floppy disk and Breeze's one-colour outline can both be
// right.

// SystemIconPrefix marks an icon set that is an installed freedesktop
// theme rather than a folder under ~/.config/uitoolkit/icons. The rest of
// the id is the theme's directory name, in its own spelling, because that
// is the name the directory has on disk and the name index.theme's
// Inherits= lines use: "desktop:breeze-dark", "desktop:Papirus-Dark".
//
// The colon is what makes the two namespaces impossible to confuse:
// [SanitizeIconSetName] rejects it, so no folder under icons/ can ever
// produce an id that begins this way, and a look.json that names one can
// only have meant a system theme.
const SystemIconPrefix = "desktop:"

// ThemeSourceDesktop marks an icon set that belongs to the desktop, not
// to the toolkit: an installed freedesktop icon theme.
const ThemeSourceDesktop ThemeSource = "desktop"

// SystemIconSetName is the icon-set id of an installed theme directory.
func SystemIconSetName(dir string) IconSetName {
	return IconSetName(SystemIconPrefix + dir)
}

// SystemIconTheme is the theme directory an id names, and whether it is
// one at all.
func SystemIconTheme(name IconSetName) (string, bool) {
	s := string(name)
	if !strings.HasPrefix(s, SystemIconPrefix) {
		return "", false
	}
	dir := strings.TrimSpace(strings.TrimPrefix(s, SystemIconPrefix))
	if !safeThemeDirName(dir) {
		return "", false
	}
	return dir, true
}

// IsSystemIconSet reports whether name is an installed freedesktop theme.
func IsSystemIconSet(name IconSetName) bool {
	_, ok := SystemIconTheme(name)
	return ok
}

// safeThemeDirName keeps a name in look.json from reaching out of the
// icon directories it is joined to. A theme directory is one path
// element: no separator, no dots of its own, nothing empty.
func safeThemeDirName(s string) bool {
	if s == "" || s == "." || s == ".." || len(s) > 128 {
		return false
	}
	if strings.ContainsAny(s, `/\`) || strings.ContainsRune(s, 0) {
		return false
	}
	if strings.Contains(s, "..") {
		return false
	}
	return true
}

// IconThemeDirsEnv names the variable that replaces the search path for
// installed icon themes. It is a colon-separated list of directories that
// hold theme folders, and it exists so a test can hold a theme of its own
// somewhere the machine's real /usr/share/icons cannot reach it — the
// same reason [SystemFontsEnv] exists for fonts. Empty means "the
// freedesktop search path"; the single word "none" means no system themes
// at all, which is what makes a screenshot reproducible on a machine that
// has forty of them.
const IconThemeDirsEnv = "UITK_ICON_THEME_DIRS"

// iconThemeSearchDirs is the freedesktop icon-theme search path, in
// order: $XDG_DATA_HOME/icons (or ~/.local/share/icons), ~/.icons,
// then $XDG_DATA_DIRS/icons with the specified default, and
// /usr/share/pixmaps last.
func iconThemeSearchDirs() []string {
	if env, ok := os.LookupEnv(IconThemeDirsEnv); ok {
		if strings.TrimSpace(env) == "" || strings.EqualFold(strings.TrimSpace(env), "none") {
			return nil
		}
		var out []string
		for _, d := range filepath.SplitList(env) {
			if d = strings.TrimSpace(d); d != "" {
				out = append(out, d)
			}
		}
		return out
	}
	var out []string
	add := func(d string) {
		if d == "" {
			return
		}
		for _, have := range out {
			if have == d {
				return
			}
		}
		out = append(out, d)
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		add(filepath.Join(d, "icons"))
	} else if home, err := os.UserHomeDir(); err == nil && home != "" {
		add(filepath.Join(home, ".local", "share", "icons"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		add(filepath.Join(home, ".icons"))
	}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if strings.TrimSpace(dirs) == "" {
		dirs = "/usr/local/share:/usr/share"
	}
	for _, d := range filepath.SplitList(dirs) {
		if d = strings.TrimSpace(d); d != "" {
			add(filepath.Join(d, "icons"))
		}
	}
	return out
}

// ---- index.theme --------------------------------------------------------------

// iconThemeDir is one of a theme's size directories, as its index.theme
// section describes it.
type iconThemeDir struct {
	sub       string // "actions/22", "16x16/apps", "scalable/status"
	size      int
	scale     int
	minSize   int
	maxSize   int
	threshold int
	scalable  bool
}

// matches is the freedesktop "directory matches size" rule.
func (d iconThemeDir) matches(size, scale int) bool {
	if d.scale != scale {
		return false
	}
	if d.scalable {
		return d.minSize <= size && size <= d.maxSize
	}
	if d.threshold > 0 {
		return d.size-d.threshold <= size && size <= d.size+d.threshold
	}
	return d.size == size
}

// distance is how far a directory is from the wanted size, for picking
// the best of the ones that did not match.
func (d iconThemeDir) distance(size, scale int) int {
	s, sz := d.size*d.scale, size*scale
	if d.scalable {
		if sz < d.minSize*d.scale {
			return d.minSize*d.scale - sz
		}
		if sz > d.maxSize*d.scale {
			return sz - d.maxSize*d.scale
		}
		return 0
	}
	if s > sz {
		return s - sz
	}
	return sz - s
}

// iconTheme is one installed theme, as read from its index.theme.
type iconTheme struct {
	dir      string   // the directory name ("breeze-dark")
	display  string   // Name= ("Breeze Dark"), or the directory name
	roots    []string // every search directory that holds a folder of that name
	dirs     []iconThemeDir
	inherits []string
	hidden   bool
}

// readIconTheme parses <root>/<dir>/index.theme for every root that has
// one. A theme split across /usr/share and ~/.local/share is one theme
// with two roots, which is how a user's additions to a system theme are
// found.
func readIconTheme(dir string) (*iconTheme, bool) {
	if !safeThemeDirName(dir) {
		return nil, false
	}
	t := &iconTheme{dir: dir, display: dir}
	seen := map[string]bool{}
	for _, root := range iconThemeSearchDirs() {
		path := filepath.Join(root, dir)
		st, err := os.Stat(path)
		if err != nil || !st.IsDir() {
			continue
		}
		t.roots = append(t.roots, path)
		f, err := os.Open(filepath.Join(path, "index.theme"))
		if err != nil {
			continue
		}
		parseIndexTheme(f, t, seen)
		f.Close()
	}
	if len(t.roots) == 0 || len(t.dirs) == 0 {
		return nil, false
	}
	return t, true
}

// parseIndexTheme reads the desktop-entry file: the [Icon Theme] header's
// Name, Inherits, Hidden and Directories, then each directory's own
// section.
func parseIndexTheme(r *os.File, t *iconTheme, seen map[string]bool) {
	type section struct {
		name string
		kv   map[string]string
	}
	var order []string
	sections := map[string]map[string]string{}
	cur := section{name: "", kv: map[string]string{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	flush := func() {
		if cur.name == "" {
			return
		}
		if _, dup := sections[cur.name]; !dup {
			order = append(order, cur.name)
			sections[cur.name] = cur.kv
		}
	}
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			cur = section{name: line[1 : len(line)-1], kv: map[string]string{}}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		// Localized keys ("Name[de]") are not this toolkit's business:
		// it shows the untranslated name, the way it shows untranslated
		// pack names.
		if strings.Contains(k, "[") {
			continue
		}
		cur.kv[k] = strings.TrimSpace(v)
	}
	flush()
	_ = order

	head := sections["Icon Theme"]
	if head == nil {
		return
	}
	if n := head["Name"]; n != "" && t.display == t.dir {
		t.display = n
	}
	if strings.EqualFold(head["Hidden"], "true") {
		t.hidden = true
	}
	for _, in := range strings.Split(head["Inherits"], ",") {
		if in = strings.TrimSpace(in); in != "" {
			t.inherits = append(t.inherits, in)
		}
	}
	list := head["Directories"]
	if s := head["ScaledDirectories"]; s != "" {
		list += "," + s
	}
	for _, sub := range strings.Split(list, ",") {
		sub = strings.TrimSpace(sub)
		if sub == "" || seen[sub] {
			continue
		}
		kv := sections[sub]
		if kv == nil {
			continue
		}
		seen[sub] = true
		t.dirs = append(t.dirs, iconThemeDirFrom(sub, kv))
	}
}

func iconThemeDirFrom(sub string, kv map[string]string) iconThemeDir {
	atoi := func(k string, def int) int {
		if v, err := strconv.Atoi(strings.TrimSpace(kv[k])); err == nil {
			return v
		}
		return def
	}
	d := iconThemeDir{sub: sub, size: atoi("Size", 0), scale: atoi("Scale", 1)}
	if d.scale < 1 {
		d.scale = 1
	}
	switch strings.ToLower(strings.TrimSpace(kv["Type"])) {
	case "scalable":
		d.scalable = true
		d.minSize = atoi("MinSize", d.size)
		d.maxSize = atoi("MaxSize", d.size)
	case "fixed":
	default: // Threshold is the specified default
		d.threshold = atoi("Threshold", 2)
	}
	if d.scalable && d.maxSize < d.minSize {
		d.maxSize = d.minSize
	}
	return d
}

// ---- the theme cache ----------------------------------------------------------

var themeIndex = struct {
	mu       sync.Mutex
	gen      uint64
	themes   map[string]*iconTheme // directory name → parsed theme (nil: not one)
	listed   []IconSetInfo
	problems []IconThemeProblem
	at       time.Time
	// drawable caches whether a theme could actually produce one of the
	// toolkit's own actions, and why it could not; see
	// [iconThemeDrawable].
	drawable map[string]bool
	why      map[string]string
}{}

// lookupTheme parses a theme's index.theme, once per cache generation.
func lookupTheme(dir string) (*iconTheme, bool) {
	themeIndex.mu.Lock()
	gen := iconGeneration()
	if themeIndex.gen != gen {
		themeIndex.themes, themeIndex.listed, themeIndex.problems = nil, nil, nil
		themeIndex.drawable, themeIndex.why = nil, nil
		themeIndex.gen = gen
	}
	if themeIndex.themes == nil {
		themeIndex.themes = map[string]*iconTheme{}
	}
	if t, ok := themeIndex.themes[dir]; ok {
		themeIndex.mu.Unlock()
		return t, t != nil
	}
	themeIndex.mu.Unlock()

	t, ok := readIconTheme(dir)
	if !ok {
		t = nil
	}
	themeIndex.mu.Lock()
	if themeIndex.gen == gen {
		themeIndex.themes[dir] = t
	}
	themeIndex.mu.Unlock()
	return t, t != nil
}

func iconGeneration() uint64 {
	iconsCache.mu.Lock()
	defer iconsCache.mu.Unlock()
	return iconsCache.gen
}

// themeChain is the theme and everything it inherits, breadth first,
// with hicolor last: the freedesktop fallback every theme has whether it
// says so or not.
func themeChain(dir string) []*iconTheme {
	var out []*iconTheme
	seen := map[string]bool{}
	queue := []string{dir}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		t, ok := lookupTheme(name)
		if !ok {
			continue
		}
		out = append(out, t)
		queue = append(queue, t.inherits...)
	}
	if !seen["hicolor"] {
		if t, ok := lookupTheme("hicolor"); ok {
			out = append(out, t)
		}
	}
	return out
}

// iconThemeExts are the file types a themed icon may be, in the order
// they are tried. .xpm is not among them: it is a 1990s X pixmap that
// the toolkit cannot decode, and a theme whose only copy of an icon is
// one is a theme that does not have it.
var iconThemeExts = []string{".png", ".svg"}

// findThemeIcon is the freedesktop lookup: every theme in the chain, its
// directories that match the wanted size first and then the nearest one,
// each extension in turn. It returns the file it found, or "".
func findThemeIcon(chain []*iconTheme, names []string, size, scale int) string {
	if size < 1 {
		size = 1
	}
	if scale < 1 {
		scale = 1
	}
	// Pass one: an exact size match anywhere in the chain, for every
	// name, before any inexact one. A theme's own 22-pixel "document-save"
	// beats hicolor's, and beats the same theme's 128-pixel copy.
	for _, exact := range []bool{true, false} {
		for _, t := range chain {
			if p := t.find(names, size, scale, exact); p != "" {
				return p
			}
		}
	}
	return ""
}

func (t *iconTheme) find(names []string, size, scale int, exact bool) string {
	best, bestD := "", 1<<30
	for _, d := range t.dirs {
		if exact && !d.matches(size, scale) {
			continue
		}
		dist := 0
		if !exact {
			if d.matches(size, scale) {
				continue // already tried
			}
			dist = d.distance(size, scale)
			if dist >= bestD {
				continue
			}
		}
		for _, root := range t.roots {
			for _, name := range names {
				for _, ext := range iconThemeExts {
					p := filepath.Join(root, d.sub, name+ext)
					if st, err := os.Stat(p); err == nil && !st.IsDir() {
						if exact {
							return p
						}
						best, bestD = p, dist
					}
				}
			}
		}
	}
	return best
}

// ---- listing ------------------------------------------------------------------

// ListSystemIconThemes is every installed freedesktop icon theme this
// toolkit can actually draw, sorted by the name it calls itself.
//
// "Can actually draw" is checked, not assumed: a theme is asked for one
// of the toolkit's own actions and the file it answers with is opened.
// A cursor theme (breeze_cursors) has no icons and does not list; a
// theme of file types this toolkit cannot decode does not list either,
// and [UnavailableSystemIconThemes] says which those are and why, so the
// answer to "where is my icon theme" is a sentence rather than a chooser
// with a gap in it.
func ListSystemIconThemes() []IconSetInfo {
	all, _ := scanSystemIconThemes()
	return all
}

// UnavailableSystemIconThemes are the installed themes that were found
// and cannot be drawn, each with the reason, sorted. It is what the icon
// chooser's tip reads out.
func UnavailableSystemIconThemes() []IconThemeProblem {
	_, bad := scanSystemIconThemes()
	return bad
}

// IconThemeProblem is an installed theme the toolkit will not offer, and
// the reason it will not.
type IconThemeProblem struct {
	Dir    string // the directory name
	Name   string // what the theme calls itself
	Reason string
}

func scanSystemIconThemes() ([]IconSetInfo, []IconThemeProblem) {
	gen := iconGeneration()
	themeIndex.mu.Lock()
	if themeIndex.listed != nil && themeIndex.gen == gen && time.Since(themeIndex.at) < iconThemeTTL {
		out := themeIndex.listed
		bad := themeIndex.problems
		themeIndex.mu.Unlock()
		return out, bad
	}
	themeIndex.mu.Unlock()

	var dirs []string
	seen := map[string]bool{}
	for _, root := range iconThemeSearchDirs() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if seen[name] || !safeThemeDirName(name) {
				continue
			}
			if !e.IsDir() {
				// A symlink to a directory reads as a link here.
				if st, err := os.Stat(filepath.Join(root, name)); err != nil || !st.IsDir() {
					continue
				}
			}
			seen[name] = true
			dirs = append(dirs, name)
		}
	}
	sort.Strings(dirs)

	var out []IconSetInfo
	var bad []IconThemeProblem
	for _, dir := range dirs {
		t, ok := lookupTheme(dir)
		if !ok {
			continue // not a theme: a cursor folder, a stray directory
		}
		why := iconThemeDrawable(t)
		if why != "" {
			bad = append(bad, IconThemeProblem{Dir: dir, Name: t.display, Reason: why})
			continue
		}
		out = append(out, IconSetInfo{
			Name:   SystemIconSetName(dir),
			Label:  t.display,
			Source: ThemeSourceDesktop,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Label), strings.ToLower(out[j].Label)
		if a != b {
			return a < b
		}
		return out[i].Name < out[j].Name
	})
	sort.Slice(bad, func(i, j int) bool { return strings.ToLower(bad[i].Name) < strings.ToLower(bad[j].Name) })

	themeIndex.mu.Lock()
	if themeIndex.gen == gen {
		themeIndex.listed, themeIndex.problems, themeIndex.at = out, bad, time.Now()
	}
	themeIndex.mu.Unlock()
	return out, bad
}

// iconThemeTTL is how long a listing of the installed themes is trusted.
// Themes are installed by a package manager, not by the second.
const iconThemeTTL = 30 * time.Second

// probeIcons are the actions a theme is tried with. Two, not sixteen:
// the question is whether a set of files exists and can be decoded, and
// a theme that has "document-save" and "edit-copy" in a format this
// toolkit reads has the rest of them in it too. Sixteen probes across
// forty installed themes would be six hundred file opens on the frame
// that builds the chooser.
var probeIcons = []ToolIcon{IconSave, IconCopy}

// iconThemeDrawable is why a theme cannot be offered, or "" when it can.
func iconThemeDrawable(t *iconTheme) string {
	themeIndex.mu.Lock()
	if themeIndex.drawable == nil {
		themeIndex.drawable = map[string]bool{}
	}
	ok, cached := themeIndex.drawable[t.dir]
	why, cachedWhy := themeIndex.why[t.dir]
	themeIndex.mu.Unlock()
	if cached {
		if ok {
			return ""
		}
		if cachedWhy {
			return why
		}
		return "this toolkit cannot read its icon files"
	}

	chain := themeChain(t.dir)
	found, drew := 0, 0
	var lastErr string
	for _, icon := range probeIcons {
		p := findThemeIcon(chain, ToolIconThemeNames(icon), 24, 1)
		if p == "" {
			continue
		}
		found++
		if _, err := loadThemeIconFile(p, 24); err == nil {
			drew++
		} else {
			lastErr = err.Error()
		}
	}
	switch {
	case found == 0:
		why = "it has no icons for the toolkit's actions"
	case drew == 0:
		why = "this toolkit cannot draw its icon files (" + lastErr + ")"
	default:
		why = ""
	}
	themeIndex.mu.Lock()
	if themeIndex.why == nil {
		themeIndex.why = map[string]string{}
	}
	themeIndex.drawable[t.dir] = why == ""
	themeIndex.why[t.dir] = why
	themeIndex.mu.Unlock()
	return why
}

// ---- drawing ------------------------------------------------------------------

// themeIconEntry is one decoded themed icon: the pixels, and whether
// they are a shape to tint or a picture to blit.
type themeIconEntry struct {
	img  *paintengine2d.Image
	tint bool
}

var themeIcons = struct {
	mu  sync.Mutex
	gen uint64
	m   map[themeIconKey]*themeIconEntry
}{m: map[themeIconKey]*themeIconEntry{}}

type themeIconKey struct {
	path string
	size int
}

// themeIconSizes are the sizes a themed icon is asked for and rasterized
// at. The toolkit draws chrome icons at 16, 24 and 32 before the display
// scale, and at 2× those on a HiDPI screen; rounding up to one of these
// keeps a scalable theme from rasterizing a new pixmap for every
// fractional scale a window is dragged through.
var themeIconSizes = []int{16, 22, 24, 32, 48, 64, 96, 128}

func themeIconSize(destW float32) int {
	want := int(destW + 0.5)
	for _, s := range themeIconSizes {
		if s >= want {
			return s
		}
	}
	return themeIconSizes[len(themeIconSizes)-1]
}

// DrawSystemToolIcon paints icon from an installed freedesktop theme.
// It reports false when the theme has nothing for that action, which is
// the caller's cue to fall back (see [DrawToolIcon]).
func DrawSystemToolIcon(ctx *paintengine2d.Context, b paintengine2d.Rect, icon ToolIcon, col paintengine2d.Color, set IconSetName) bool {
	dir, ok := SystemIconTheme(set)
	if !ok || ctx == nil || b.Empty() || icon == IconNone {
		return false
	}
	size := themeIconSize(b.Dx())
	path := findThemeIcon(themeChain(dir), ToolIconThemeNames(icon), size, 1)
	if path == "" {
		return false
	}
	e, err := loadThemeIcon(path, size)
	if err != nil || e == nil || e.img == nil {
		return false
	}
	src := paintengine2d.XYWH(0, 0, float32(e.img.Width), float32(e.img.Height))
	if e.tint {
		return paintTintedIcon(ctx, b, e.img, col)
	}
	ctx.DrawImageRect(e.img, src, b)
	return true
}

// SystemIconThemeHas reports whether an installed theme can draw icon.
func SystemIconThemeHas(set IconSetName, icon ToolIcon) bool {
	dir, ok := SystemIconTheme(set)
	if !ok {
		return false
	}
	path := findThemeIcon(themeChain(dir), ToolIconThemeNames(icon), 24, 1)
	if path == "" {
		return false
	}
	_, err := loadThemeIcon(path, 24)
	return err == nil
}

func loadThemeIcon(path string, size int) (*themeIconEntry, error) {
	key := themeIconKey{path, size}
	gen := iconGeneration()
	themeIcons.mu.Lock()
	if themeIcons.gen != gen {
		themeIcons.m = map[themeIconKey]*themeIconEntry{}
		themeIcons.gen = gen
	}
	if e, ok := themeIcons.m[key]; ok {
		themeIcons.mu.Unlock()
		if e == nil {
			return nil, errThemeIcon
		}
		return e, nil
	}
	themeIcons.mu.Unlock()

	e, err := loadThemeIconFile(path, size)
	themeIcons.mu.Lock()
	if themeIcons.gen == gen {
		themeIcons.m[key] = e // a nil entry is remembered as a miss
	}
	themeIcons.mu.Unlock()
	return e, err
}

// errThemeIcon is the remembered failure of a file that would not decode.
var errThemeIcon = themeIconError("this toolkit cannot decode the file")

type themeIconError string

func (e themeIconError) Error() string { return string(e) }

// loadThemeIconFile decodes one themed icon file and decides whether it
// is a shape or a picture.
func loadThemeIconFile(path string, size int) (*themeIconEntry, error) {
	var img *paintengine2d.Image
	var err error
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		img, err = paintengine2d.DecodePNGFile(path)
	case ".svg":
		img, err = RasterizeIconSVG(path, size)
	default:
		return nil, errThemeIcon
	}
	if err != nil {
		return nil, err
	}
	if img == nil || img.Width < 1 || img.Height < 1 {
		return nil, errThemeIcon
	}
	if iconImageIsMonochrome(img) {
		return &themeIconEntry{img: toWhiteMask(img), tint: true}, nil
	}
	return &themeIconEntry{img: img}, nil
}

// iconImageIsMonochrome reports whether every pixel that is painted at
// all is the same colour, which is what makes an icon a shape the look
// may tint rather than a picture it must leave alone.
//
// The test is on un-premultiplied colour, because that is what separates
// the two cases: a symbolic icon is one ink and an alpha ramp, so its
// colour is constant and only its coverage varies, while a full-colour
// icon's colour varies by definition. Nearly-clear pixels are skipped —
// their colour is a rounding artefact of the premultiplied bytes — and
// so is a wholly transparent image, which is not monochrome, it is
// nothing.
func iconImageIsMonochrome(img *paintengine2d.Image) bool {
	var r0, g0, b0 int
	first := true
	painted := 0
	// A coarse sweep first: a 128-pixel icon does not need every pixel
	// read to answer this, and the step keeps the cost of listing forty
	// themes flat.
	step := 1
	if n := img.Width * img.Height; n > 64*64 {
		step = 2
	}
	for y := 0; y < img.Height; y += step {
		for x := 0; x < img.Width; x += step {
			r, g, b, a := img.PremulAt(x, y)
			if a < 32 {
				continue
			}
			painted++
			ur := int(r) * 255 / int(a)
			ug := int(g) * 255 / int(a)
			ub := int(b) * 255 / int(a)
			if first {
				r0, g0, b0 = ur, ug, ub
				first = false
				continue
			}
			if abs(ur-r0) > 24 || abs(ug-g0) > 24 || abs(ub-b0) > 24 {
				return false
			}
		}
	}
	return painted > 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
