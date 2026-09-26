package style

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// A skin is a pack.
//
// That is the decision this file implements, and it is what makes the
// feature safe. A skin is not a mode an app opts into and not a second
// picker: it is a ThemePack with `"engine": "skin"`, listed in Settings
// beside the other 123, selected by the same look.json preference, applied
// live by the same watcher, and reachable with UITK_THEME=<id> like any
// other. So a skinned app is a themed app underneath, always, and dropping
// the skin at run time leaves a working app.
//
// The cost of that decision is three call sites in theme.go — the list, the
// load, and the file the watcher stamps. Everything else is here.

//go:embed skins/*/skin.json skins/*/art/*.png
var builtinSkinFS embed.FS

// SkinsDir is $XDG_CONFIG_HOME/uitoolkit/skins (or ~/.config/uitoolkit/skins),
// beside the user's themes and icon sets.
func SkinsDir() string { return filepath.Join(ConfigDir(), "skins") }

// SkinDir is skins/<name>/.
func SkinDir(name string) string { return filepath.Join(SkinsDir(), name) }

// skinRegistry holds the skins this process can paint: the ones embedded in
// the binary, and the ones under SkinsDir. User skins are re-scanned on the
// asset TTL so installing or editing one applies without a restart.
var skinRegistry = struct {
	mu      sync.RWMutex
	builtin map[string]*Skin
	order   []string

	user    map[string]*Skin
	userGen uint64
	userDir string
	userAt  time.Time
}{
	builtin: map[string]*Skin{},
	user:    map[string]*Skin{},
}

func init() {
	for _, sk := range loadBuiltinSkins() {
		skinRegistry.builtin[sk.Name] = sk
		skinRegistry.order = append(skinRegistry.order, sk.Name)
		RegisterPack(sk.Pack())
	}
}

// loadBuiltinSkins reads the skins embedded in the binary. A shipped skin
// that fails to load is a build error, not a run-time surprise: the test
// TestBuiltinSkinsLoad asserts every one of them parses.
func loadBuiltinSkins() []*Skin {
	var out []*Skin
	entries, err := fs.ReadDir(builtinSkinFS, "skins")
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub, err := fs.Sub(builtinSkinFS, "skins/"+e.Name())
		if err != nil {
			continue
		}
		sk, err := LoadSkinFS(sub, e.Name())
		if err != nil {
			continue
		}
		sk.Source = ThemeSourceBuiltin
		out = append(out, sk)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Year != out[j].Year {
			return out[i].Year < out[j].Year
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// invalidateSkinPacks drops the scanned user skins so the next list or load
// re-reads the directory. InvalidateSkinCache calls it; so does the look
// watcher, through InvalidateIconCache.
func invalidateSkinPacks() {
	skinRegistry.mu.Lock()
	skinRegistry.user = map[string]*Skin{}
	skinRegistry.userDir = ""
	skinRegistry.userAt = time.Time{}
	skinRegistry.mu.Unlock()
}

// userSkins is every loadable skin under SkinsDir, keyed by its canonical
// id. The scan is cached on the asset TTL, on the icon cache's generation
// and on the directory itself, so a frame that paints a skinned control
// costs no syscalls and an installed skin still shows up about a second
// later — while an app (or a test) that moves XDG_CONFIG_HOME sees the new
// directory at once rather than the old one's skins for a second.
func userSkins() map[string]*Skin {
	gen, dir := IconGeneration(), SkinsDir()
	skinRegistry.mu.RLock()
	fresh := skinRegistry.user != nil && skinRegistry.userGen == gen &&
		skinRegistry.userDir == dir && time.Since(skinRegistry.userAt) < skinAssetTTL
	if fresh {
		m := skinRegistry.user
		skinRegistry.mu.RUnlock()
		return m
	}
	skinRegistry.mu.RUnlock()

	m := scanUserSkins()

	skinRegistry.mu.Lock()
	skinRegistry.user = m
	skinRegistry.userGen = gen
	skinRegistry.userDir = dir
	skinRegistry.userAt = time.Now()
	skinRegistry.mu.Unlock()
	return m
}

// scanUserSkins reads SkinsDir. A folder holding a skin.json is a skin; so
// is a .uskin archive, which is what a skin shared as one file is. A skin
// that does not load is skipped, not fatal: one stranger's broken zip must
// not take the theme browser down with it.
func scanUserSkins() map[string]*Skin {
	out := map[string]*Skin{}
	entries, err := os.ReadDir(SkinsDir())
	if err != nil {
		return out
	}
	for _, e := range entries {
		raw := e.Name()
		id := raw
		archive := false
		if !e.IsDir() {
			if !strings.EqualFold(filepath.Ext(raw), SkinArchiveExt) {
				continue
			}
			id, archive = strings.TrimSuffix(raw, filepath.Ext(raw)), true
		}
		clean, err := SanitizeThemeName(id)
		if err != nil {
			continue
		}
		if _, dup := out[clean]; dup {
			continue // first (sorted) entry wins, deterministically
		}
		path := filepath.Join(SkinsDir(), raw)
		sk, err := loadSkinPath(path, clean, archive)
		if err != nil {
			continue
		}
		out[clean] = sk
	}
	return out
}

// loadSkinPath reads one skin from a directory or a .uskin archive.
func loadSkinPath(path, name string, archive bool) (*Skin, error) {
	if archive {
		// The archive is read once, into memory, and closed: a skin is a
		// few hundred KB of PNG and holding an open zip for the life of the
		// process would pin a file the user may want to replace.
		fsys, closeFn, err := openSkinArchive(path)
		if err != nil {
			return nil, err
		}
		defer func() { _ = closeFn() }()
		sk, err := LoadSkinFS(fsys, name)
		if err != nil {
			return nil, err
		}
		mem, err := snapshotSkinFS(fsys)
		if err != nil {
			return nil, err
		}
		sk.fsys = mem
		sk.Source, sk.Dir, sk.Path = ThemeSourceUser, path, path
		return sk, nil
	}
	sk, err := LoadSkinFS(os.DirFS(path), name)
	if err != nil {
		return nil, err
	}
	sk.Source, sk.Dir, sk.Path = ThemeSourceUser, path, filepath.Join(path, SkinFile)
	return sk, nil
}

// LoadSkin reads the named skin: a user skin under SkinsDir first, then one
// embedded in the binary — the same precedence LoadTheme gives user packs
// over builtins, so a user can shadow a shipped skin with their own.
func LoadSkin(name string) (*Skin, bool) {
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return nil, false
	}
	if sk, ok := userSkins()[clean]; ok {
		return sk, true
	}
	skinRegistry.mu.RLock()
	sk, ok := skinRegistry.builtin[clean]
	skinRegistry.mu.RUnlock()
	return sk, ok
}

// IsSkin reports whether a pack id names a skin.
func IsSkin(name string) bool {
	_, ok := LoadSkin(name)
	return ok
}

// ListSkins returns every skin this process can paint, built-in first, each
// as the ThemePack the theme browser already knows how to render.
func ListSkins() []ThemePack {
	// The skins are taken under the lock and their packs built outside it:
	// Pack resolves the base through LoadTheme, which reads this registry
	// again, and an RWMutex is not reentrant.
	skinRegistry.mu.RLock()
	builtin := make([]*Skin, 0, len(skinRegistry.order))
	for _, name := range skinRegistry.order {
		if sk, ok := skinRegistry.builtin[name]; ok {
			builtin = append(builtin, sk)
		}
	}
	skinRegistry.mu.RUnlock()

	out := make([]ThemePack, 0, len(builtin))
	for _, sk := range builtin {
		out = append(out, sk.Pack())
	}
	for _, sk := range sortedUserSkins() {
		out = append(out, sk.Pack())
	}
	return out
}

// ListUserSkins is the skins installed under SkinsDir, sorted by id.
func ListUserSkins() []ThemePack {
	var out []ThemePack
	for _, sk := range sortedUserSkins() {
		out = append(out, sk.Pack())
	}
	return out
}

func sortedUserSkins() []*Skin {
	m := userSkins()
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]*Skin, 0, len(names))
	for _, n := range names {
		out = append(out, m[n])
	}
	return out
}

// SkinSourceFile is the manifest backing a user skin, or "" for a builtin
// (or a name that is not a skin). theme.go's ThemeSourceFile returns it, so
// the look watcher stamps an edited skin.json and applies it live through
// exactly the machinery that already reloads an edited theme.json.
func SkinSourceFile(name string) string {
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return ""
	}
	if sk, ok := userSkins()[clean]; ok {
		return sk.Path
	}
	return ""
}

// Pack is the skin as a ThemePack: what Settings lists, what look.json
// names, and what LoadTheme hands back.
//
// Its tokens are the base pack's with the skin's own on top, so every colour
// and metric the skin does not state is the base look's — which is what
// makes an undescribed control look like it belongs rather than like a hole.
func (sk *Skin) Pack() ThemePack {
	if sk == nil {
		return ThemePack{}
	}
	tok := sk.Tokens
	fam := sk.Family
	if base, ok := sk.basePack(); ok {
		tok = mergeTokens(base.Tokens, sk.Tokens)
		if fam == "" {
			fam = base.Palette
		}
		// The era's typefaces come with the base pack. They are keyed by
		// the pack's name and its *engine*, and a skin's engine is "skin",
		// so a skin would otherwise be the only pack in the toolkit reading
		// in the default face — where a skin over Windows 95 should read in
		// Windows 95's. Resolved here against the base's own identity; a
		// skin that states its own "fonts" keeps them.
		era := withEraFonts(ThemeTokens{Engine: base.Tokens.Engine}, base.Name).Fonts
		if len(tok.Fonts.UI) == 0 {
			tok.Fonts.UI = era.UI
		}
		if len(tok.Fonts.Mono) == 0 {
			tok.Fonts.Mono = era.Mono
		}
	}
	tok.Engine = skinEngineID
	if fam != "" {
		tok.Family = fam
	}
	// A skin that paints its own faces has no bevel language: its pressed
	// state is a sprite, not a one-pixel shift, and its scrollbar is a
	// picture, not a pair of drawn arrow wells. Leaving the base pack's
	// bevel in place makes the stock painter add both on top of the art —
	// which is how a pressed button's bottom edge ended up a pixel outside
	// its box. A skin that states "bevel" explicitly keeps what it said,
	// and a skin that binds no faces at all keeps its base's, so a skin
	// describing nothing is still exactly its base pack.
	if sk.Tokens.Bevel == "" && sk.bindsAnyFace() {
		tok.Bevel = BevelNone
	}
	tok = tok.Resolve()
	summary := sk.Summary
	if summary == "" {
		summary = "A skin: " + sk.Label + ", painted from sprite sheets."
	}
	return ThemePack{
		Name:    sk.Name,
		Label:   sk.Label,
		Source:  sk.Source,
		Palette: tok.Family,
		Era:     EraSkin,
		Tokens:  tok,
		Year:    sk.Year,
		Lineage: sk.Lineage,
		Summary: summary,
	}
}

// EraSkin is the Settings grouping label for skins.
const EraSkin = "Skins"

// bindsAnyFace reports whether the skin paints any of the engine's Role
// faces itself.
func (sk *Skin) bindsAnyFace() bool {
	for _, name := range skinRolePart {
		if sk.has(name) {
			return true
		}
	}
	return false
}

// basePack is the pack painted under the art. A skin that names none takes
// the toolkit's default, so "base" is never empty and a skin never has to
// restate a whole palette to be usable.
//
// A skin may not stand on another skin. One level of fallback is a feature;
// a graph of them is a puzzle, and a cycle is a hang — a skin whose base is
// a skin whose base is the first would resolve for ever.
func (sk *Skin) basePack() (ThemePack, bool) {
	if name := strings.TrimSpace(sk.Base); name != "" && name != sk.Name && !IsSkin(name) {
		if p, ok := LoadTheme(name); ok && p.Tokens.Engine != skinEngineID {
			return p, true
		}
	}
	// The toolkit's default, read straight from the built-in packs rather
	// than through LoadTheme: a user skin installed under that name would
	// otherwise send the search round again.
	return builtinEraPack(DefaultThemeName)
}

// BaseEngineFor is the engine that paints what a skin leaves out: its base
// pack's own. It is memoised per look, so the lookup happens once.
func (sk *Skin) baseEngine(l *Classic) Engine {
	if sk == nil || l == nil {
		return baseEngine
	}
	return l.Memo(skinBaseKey{}, func() any {
		p, ok := sk.basePack()
		if !ok {
			return baseEngine
		}
		e := engineFor(p.Tokens)
		if e == nil || e.ID() == skinEngineID {
			return baseEngine
		}
		return e
	}).(Engine)
}

type skinBaseKey struct{}

// skinFor is the skin a look paints with, or nil for any other look. The
// look carries its pack's id, so this is one map read behind a memo.
func skinFor(l *Classic) *Skin {
	if l == nil {
		return nil
	}
	v := l.Memo(skinLookKey{}, func() any {
		sk, ok := LoadSkin(l.Pack())
		if !ok {
			return (*Skin)(nil)
		}
		return sk
	})
	sk, _ := v.(*Skin)
	return sk
}

type skinLookKey struct{}

// snapshotSkinFS copies an fs.FS into memory, so an archive can be closed
// while its art stays readable.
func snapshotSkinFS(src fs.FS) (fs.FS, error) {
	out := memSkinFS{}
	err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		out[p] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// memSkinFS is a read-only fs.FS over bytes held in memory.
type memSkinFS map[string][]byte

func (m memSkinFS) Open(name string) (fs.File, error) {
	b, ok := m[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &memSkinFile{name: name, data: b}, nil
}

type memSkinFile struct {
	name string
	data []byte
	off  int
}

func (f *memSkinFile) Stat() (fs.FileInfo, error) {
	return memSkinInfo{f.name, int64(len(f.data))}, nil
}
func (f *memSkinFile) Close() error { return nil }
func (f *memSkinFile) Read(p []byte) (int, error) {
	if f.off >= len(f.data) {
		return 0, io.EOF
	}
	n := copy(p, f.data[f.off:])
	f.off += n
	return n, nil
}

type memSkinInfo struct {
	name string
	size int64
}

func (i memSkinInfo) Name() string       { return filepath.Base(i.name) }
func (i memSkinInfo) Size() int64        { return i.size }
func (i memSkinInfo) Mode() fs.FileMode  { return 0o444 }
func (i memSkinInfo) ModTime() time.Time { return time.Time{} }
func (i memSkinInfo) IsDir() bool        { return false }
func (i memSkinInfo) Sys() any           { return nil }
