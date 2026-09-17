package style

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// A skin is a theme made of pictures.
//
// The 29 engines in this package are code: win95's bevel is four colours and
// a rule, system7's is 1-bit pixel art drawn by hand. A skin is the other
// answer to the same question — the one WinAmp, VLC's skins2 and Windows
// Media Player gave: named sub-rects of a bitmap bound to the parts of a
// control. It is loaded as a pack of the "skin" engine (engine_skin.go), so
// it lists, applies and reloads exactly as the other 121 packs do.
//
// Three rules separate this format from the ones it takes its ideas from:
//
//   - Every number is in the art's own design pixels, the way a pack states
//     its metrics at 1×. The toolkit multiplies by the display scale and
//     chooses the asset at or above it; a skin author never writes a device
//     pixel. WinAmp's 275×116 was frozen forever because its format had no
//     such unit.
//   - A skin replaces painting and layout, never behaviour. There is no
//     script, no bytecode and no way to name a control the toolkit does not
//     have, so a skinned button is still a widgets.Button with a name, a
//     keyboard route, a focus ring and an accessibility node.
//   - Anything the skin leaves out is painted by its base pack. A skin is
//     always a partial override, so a half-finished one is a working app in
//     the base look rather than a hole, and dropping the skin at run time
//     leaves the app running.
//
// The manifest is skin.json beside the art, read from a directory or from a
// .uskin zip of one. docs/skins.md is the worked example.

// SkinVersion is the manifest format this build reads. A skin states its own
// in the "skin" key; one from the future is refused by name instead of
// half-read, because a key this build ignores may be the one that made the
// art make sense.
const SkinVersion = 1

// SkinFile is the manifest's name inside a skin directory or archive.
const SkinFile = "skin.json"

// SkinArchiveExt is the extension of a single-file skin: a zip of the
// directory, which is what every skin format of the era turned out to be
// (.wsz, .vlt and .wmz are all renamed zips).
const SkinArchiveExt = ".uskin"

// ---- errors ---------------------------------------------------------------

// SkinError names the manifest key a problem came from, so a skin author
// reads `parts.button.states.hover: no sprite "btn.h"` and not a byte
// offset. Key is the dotted path from the document root; it is empty only
// for the document itself.
type SkinError struct {
	Skin string // the skin's id, when it has one
	Key  string // dotted key path ("parts.button.slice")
	Msg  string
}

func (e *SkinError) Error() string {
	var b strings.Builder
	if e.Skin != "" {
		b.WriteString(e.Skin)
		b.WriteString(": ")
	}
	b.WriteString(SkinFile)
	b.WriteString(": ")
	if e.Key != "" {
		b.WriteString(e.Key)
		b.WriteString(": ")
	}
	b.WriteString(e.Msg)
	return b.String()
}

// skinErr builds a keyed manifest error.
func skinErr(key, format string, args ...any) *SkinError {
	return &SkinError{Key: key, Msg: fmt.Sprintf(format, args...)}
}

// joinKey appends a key segment to a dotted path.
func joinKey(base, k string) string {
	if base == "" {
		return k
	}
	return base + "." + k
}

// ---- the manifest ---------------------------------------------------------

// Skin is a loaded manifest: the identity a pack needs, the art, and the
// bindings from the toolkit's parts to it. Everything in it is immutable
// once loaded, so one Skin is shared by every look built from it.
type Skin struct {
	// Name is the pack id: the directory's name, not a value in the file,
	// so a skin cannot claim another pack's id by editing its manifest.
	Name string
	// Version is the manifest's declared format version.
	Version int

	// Pack identity, listed by Settings exactly as an engine pack's is.
	Label   string
	Year    int
	Lineage string
	Summary string
	Family  ThemeName

	// Base is the pack painted under the art: every control the skin does
	// not describe, and every colour and metric it does not state. Empty
	// means the family's starter.
	Base string

	// Design is the grid the art was drawn on.
	Design SkinDesign

	// Sheets, Sprites and Parts are the art and its bindings. Parts is
	// keyed by the names in skinPartNames — a skin cannot invent one.
	Sheets  map[string]*SkinSheet
	Sprites map[string]*SkinSprite
	Parts   map[string]*SkinPart
	// Text are the named label roles parts refer to.
	Text map[string]*SkinText

	// Window is the frame the skin asks for. Its silhouette is carried but
	// not yet consumed: a skin paints inside an ordinary rectangular window
	// until the shaped-window work lands (docs/skins.md, "What a skin
	// cannot do yet").
	Window *SkinWindow

	// Tokens are the colours, metrics and params for what art does not
	// cover, in theme.json's own vocabulary.
	Tokens ThemeTokens

	Source ThemeSource
	// Dir is the directory or archive the skin was read from ("" when it is
	// embedded in the binary); Path is the manifest inside it, which the
	// look watcher stamps so an edited skin applies without a restart.
	Dir  string
	Path string

	// fsys reads the art. It is the skin's own root, so a manifest can only
	// name files inside its own package.
	fsys fs.FS
}

// SkinDesign is the grid the art was drawn on. Every rect, inset and metric
// in the manifest is in these pixels.
type SkinDesign struct {
	// Scale is the design grid's own scale; 1 means the numbers are 1×
	// pixels, which is what a pack's metrics are. Art drawn at 2× states 2
	// and the toolkit halves every number once at load.
	Scale float32
	// Pixelated is the default sampling for the skin's sheets: nearest
	// neighbour at integer multiples (a deliberate pixel-art skin) rather
	// than a smooth downscale. A sheet may say otherwise.
	Pixelated bool
}

// SkinSheet is one sprite sheet at every scale it was drawn at. Files are
// chosen upward — the nearest asset at or above the display scale, drawn
// down — so art is never enlarged, which is where bitmap skins historically
// went soft. style/iconset.go picks @2x icons by the same rule.
type SkinSheet struct {
	Name string
	// Files maps a scale (1, 1.5, 2, 3) onto a path inside the skin.
	Files map[float32]string
	// Pixelated overrides the skin's default sampling for this sheet.
	Pixelated bool
	// scales are Files' keys, ascending, so the search is a walk.
	scales []float32
}

// SkinSprite is a named sub-rect of a sheet — WinAmp's fixed offsets, VLC's
// <SubBitmap> — with the nine-slice rule for stretching it.
type SkinSprite struct {
	Name  string
	Sheet *SkinSheet
	// X, Y, W, H are the sub-rect in design pixels.
	X, Y, W, H float32
	// Slice are the fixed insets (top, right, bottom, left): the corners
	// and edges that keep their size while the middle grows. Zero means the
	// sprite is stretched whole, which is right for a glyph and wrong for a
	// button.
	Slice Insets
	// Middle is how the stretchable parts are filled.
	Middle SkinFill
	// Tint paints the sprite in a colour from the part's text role instead
	// of its own, for single-colour glyphs (arrows, ticks) that should
	// follow the label. The sheet's alpha is the coverage.
	Tint bool
}

// Sliced reports whether the sprite has fixed corners.
func (s *SkinSprite) Sliced() bool { return s != nil && !s.Slice.Zero() }

// SkinFill is how a nine-slice's stretchable parts are filled.
type SkinFill uint8

const (
	// FillStretch scales the middle to fit (the usual case: a gradient, a
	// gloss, a flat face).
	FillStretch SkinFill = iota
	// FillTile repeats it at its own size (a texture, a hatch, a grille).
	FillTile
	// FillNone leaves it empty: the corners and edges only, so the art
	// frames whatever is behind it.
	FillNone
)

func (f SkinFill) String() string {
	switch f {
	case FillTile:
		return "tile"
	case FillNone:
		return "none"
	default:
		return "stretch"
	}
}

// SkinPart binds one of the toolkit's parts to art. A part with no art for
// a state falls back along skinStateFallback, and a part with no art at all
// falls through to the base pack, so binding only what matters is the
// supported way to write a skin.
type SkinPart struct {
	Name string
	// States maps a state name (skinStateNames) onto a sprite.
	States map[string]*SkinSprite
	// Text is the label role this part's text is drawn in.
	Text *SkinText
	// Pad is extra room the art needs inside its box, in design pixels,
	// beyond what the base pack's metrics keep.
	Pad Insets
}

// SkinText is a named label role: the colours and face a skin labels a
// family of controls in. Parts refer to one by name so a skin changes every
// label at once, which is what a palette is for in an ordinary pack.
type SkinText struct {
	Name string
	// Color is the resting label colour; the rest fall back to it.
	Color    paintengine2d.Color
	Hover    paintengine2d.Color
	Pressed  paintengine2d.Color
	Disabled paintengine2d.Color
	Checked  paintengine2d.Color
	// Size is the face's size in design pixels (0: the pack's), Bold the
	// weight. Widgets measure labels with the face the engine names, so a
	// skin whose buttons are bold gets buttons wide enough for them.
	Size float32
	Bold bool
}

// SkinWindow is the frame a skin asks for. Only Border and Caption are read
// today; Shape is carried, validated and reported so a skin written now is
// a complete document when shaped windows land.
type SkinWindow struct {
	Border  Insets
	Caption float32
	Layout  string
	Radius  [4]float32
	// Shape is the window's silhouette in design pixels, as a union of
	// rounded rects. It is deliberately data and not an SVG path: nothing
	// in the toolkit parses path strings, and a rect union is the form both
	// the compositor's input region and a hit test want anyway.
	Shape []SkinShapeRect
}

// SkinShapeRect is one rounded rect of a silhouette, in design pixels.
// Right and Bottom are measured from the window's right and bottom edges
// when Anchor says so, so a shape follows a resize.
type SkinShapeRect struct {
	X, Y, W, H float32
	Radius     [4]float32
	// Stretch grows this rect with the window instead of pinning it: X
	// stretch keeps W as a margin from the right edge, Y from the bottom.
	StretchX, StretchY bool
}

// ---- vocabularies ---------------------------------------------------------

// skinPartNames is every part a skin may bind, and what painting it changes.
// A skin that names anything else fails validation: new controls come from
// the toolkit, not from a stranger's zip file.
//
// The first thirteen are the engine's Role faces (docs/theme-engines.md):
// binding one changes every control that paints it, so "button" is also the
// scrollbar's step buttons and "bar" is the menu, tool and status bars.
var skinPartNames = map[string]string{
	"button":   "push buttons, dialog buttons, scrollbar step buttons",
	"tool":     "tool bar buttons and free toggles",
	"field":    "text fields and text areas",
	"check":    "the check box and radio well",
	"row":      "list, tree and table row highlights",
	"tab":      "tabs",
	"thumb":    "scrollbar thumbs",
	"track":    "scrollbar and slider tracks",
	"menu":     "the hot menu item and open menu title",
	"combo":    "combo boxes",
	"splitter": "splitter handles",
	"bar":      "menu, tool, status bars and tab strips",
	"panel":    "panels, cards and group boxes",

	// Parts below are glyphs and whole controls the Role faces cannot
	// express, each bound to one engine hook.
	"check.mark":    "the tick inside a checked box",
	"radio.mark":    "the dot inside a selected radio",
	"arrow.up":      "up arrows (scroll, spin, sort)",
	"arrow.down":    "down arrows (combo, scroll, spin, sort)",
	"arrow.left":    "left arrows",
	"arrow.right":   "right arrows and closed submenu markers",
	"expander.open": "an expanded tree or accordion disclosure",
	"expander.shut": "a collapsed one",
	"slider.track":  "the groove a slider thumb travels",
	"slider.fill":   "the travelled part of that groove",
	"slider.thumb":  "the slider thumb",
	"progress.back": "a progress bar's trough",
	"progress.fill": "its filled part",
	"switch.track":  "a switch's track",
	"switch.knob":   "its knob",
	"focus":         "the keyboard focus ring (a skin may re-draw it, never remove it)",
	"window":        "the window background, tiled or stretched behind everything",
	"caption":       "the top-level window's caption band",
	"menu.frame":    "the frame a menu popup is drawn on",
	"tooltip":       "a tool tip's frame",
}

// skinStateNames is every state a part may carry art for. They are the
// toolkit's own ControlState bits, named, so what a skin can say and what a
// widget can be are the same list.
var skinStateNames = []string{
	"normal", "hover", "pressed", "disabled",
	"focus", "checked", "checkedHover", "checkedPressed",
	"default", "inactive",
}

// skinStateFallback is how a missing state resolves. It is the whole reason
// a terse skin still looks deliberate: art for "normal" alone gives a
// control every state, and each extra sprite sharpens one more.
//
// Read each row as "when this state has no art, try these, in order".
var skinStateFallback = map[string][]string{
	"normal":         nil,
	"hover":          {"normal"},
	"pressed":        {"hover", "normal"},
	"disabled":       {"normal"},
	"focus":          {"hover", "normal"},
	"checked":        {"pressed", "hover", "normal"},
	"checkedHover":   {"checked", "pressed", "hover", "normal"},
	"checkedPressed": {"checked", "pressed", "hover", "normal"},
	"default":        {"hover", "normal"},
	"inactive":       {"disabled", "normal"},
}

// skinStateSet is skinStateNames as a set.
var skinStateSet = func() map[string]bool {
	m := make(map[string]bool, len(skinStateNames))
	for _, s := range skinStateNames {
		m[s] = true
	}
	return m
}()

// SkinPartNames lists every part a skin may bind, sorted. The lint command
// and docs/skins.md are generated from it, so the format's vocabulary has
// one definition.
func SkinPartNames() []string {
	out := make([]string, 0, len(skinPartNames))
	for k := range skinPartNames {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// SkinPartDoc is what binding a part changes ("" for an unknown name).
func SkinPartDoc(name string) string { return skinPartNames[name] }

// SkinStateNames lists every state a part may carry art for, in resolution
// order.
func SkinStateNames() []string { return append([]string(nil), skinStateNames...) }

// SkinStateFallback is the order a missing state's art is looked for in.
func SkinStateFallback(state string) []string {
	return append([]string(nil), skinStateFallback[state]...)
}

// ---- the document ---------------------------------------------------------

// skinFileJSON is skin.json, decoded. It mirrors themeFileJSON in theme.go:
// the on-disk shape stays in one struct so the format is readable as a
// struct and every default lives in the loader below it.
type skinFileJSON struct {
	Skin int `json:"skin"`

	Label   string `json:"label,omitempty"`
	Year    int    `json:"year,omitempty"`
	Lineage string `json:"lineage,omitempty"`
	Summary string `json:"summary,omitempty"`
	Family  string `json:"family,omitempty"`
	Base    string `json:"base,omitempty"`

	Design *skinDesignJSON `json:"design,omitempty"`

	Sheets  map[string]json.RawMessage `json:"sheets,omitempty"`
	Sprites map[string]json.RawMessage `json:"sprites,omitempty"`
	Parts   map[string]json.RawMessage `json:"parts,omitempty"`
	Text    map[string]json.RawMessage `json:"text,omitempty"`
	Window  json.RawMessage            `json:"window,omitempty"`

	Colors  map[string]string  `json:"colors,omitempty"`
	Metrics *chromeMetricsJSON `json:"metrics,omitempty"`
	Params  map[string]float32 `json:"params,omitempty"`
	Extra   map[string]string  `json:"extra,omitempty"`
	Fonts   *FontPrefs         `json:"fonts,omitempty"`
	Bevel   string             `json:"bevel,omitempty"`
}

type skinDesignJSON struct {
	Scale     float32 `json:"scale,omitempty"`
	Pixelated bool    `json:"pixelated,omitempty"`
}

type skinSheetJSON struct {
	Pixelated *bool `json:"pixelated,omitempty"`
	// Scale files are free-form keys ("1x", "1.5x", "2x", "3x"); they are
	// read from the raw object below, because a fixed struct would freeze
	// the scale set at whatever this build happened to know.
	files map[float32]string
}

type skinSpriteJSON struct {
	Sheet  string    `json:"sheet,omitempty"`
	At     []float32 `json:"at,omitempty"`
	Slice  []float32 `json:"slice,omitempty"`
	Middle string    `json:"middle,omitempty"`
	Tint   bool      `json:"tint,omitempty"`
	X      *float32  `json:"x,omitempty"`
	Y      *float32  `json:"y,omitempty"`
	W      *float32  `json:"w,omitempty"`
	H      *float32  `json:"h,omitempty"`
}

type skinPartJSON struct {
	States map[string]json.RawMessage `json:"states,omitempty"`
	Strip  *skinStripJSON             `json:"strip,omitempty"`
	Text   json.RawMessage            `json:"text,omitempty"`
	Pad    []float32                  `json:"pad,omitempty"`
}

// skinStripJSON is the shorthand every real sprite sheet is laid out for: a
// run of equal cells across a sheet, one per state. It is what turns a
// hundred-line sprite table into one line per part.
type skinStripJSON struct {
	Sheet  string    `json:"sheet"`
	At     []float32 `json:"at"`
	States []string  `json:"states"`
	Slice  []float32 `json:"slice,omitempty"`
	Middle string    `json:"middle,omitempty"`
	Tint   bool      `json:"tint,omitempty"`
	// Down lays the cells down the sheet instead of across it.
	Down bool `json:"down,omitempty"`
	// Gap is the space between cells, in design pixels (0: they abut).
	Gap float32 `json:"gap,omitempty"`
}

type skinTextJSON struct {
	Color    string   `json:"color,omitempty"`
	Hover    string   `json:"hover,omitempty"`
	Pressed  string   `json:"pressed,omitempty"`
	Disabled string   `json:"disabled,omitempty"`
	Checked  string   `json:"checked,omitempty"`
	Size     *float32 `json:"size,omitempty"`
	Bold     bool     `json:"bold,omitempty"`
}

type skinWindowJSON struct {
	Border  []float32           `json:"border,omitempty"`
	Caption *float32            `json:"caption,omitempty"`
	Layout  string              `json:"layout,omitempty"`
	Radius  []float32           `json:"radius,omitempty"`
	Shape   []skinShapeRectJSON `json:"shape,omitempty"`
}

type skinShapeRectJSON struct {
	At       []float32 `json:"at"`
	Radius   []float32 `json:"radius,omitempty"`
	StretchX bool      `json:"stretchX,omitempty"`
	StretchY bool      `json:"stretchY,omitempty"`
}

// ---- decoding -------------------------------------------------------------

// decodeSkinJSON decodes one manifest object strictly: a key this build does
// not know is an error naming the key, not a silent drop. A stranger's skin
// that half-applies is worse than one that refuses to load and says why.
func decodeSkinJSON(key string, raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return skinErr(key, "%s", skinDecodeMessage(err))
	}
	return nil
}

// skinDecodeMessage turns encoding/json's wording into the format's own, so
// a skin author reads about the key they wrote rather than about a Go type.
func skinDecodeMessage(err error) string {
	if name, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		return "unknown key " + name
	}
	if t, ok := err.(*json.UnmarshalTypeError); ok {
		where := ""
		if t.Field != "" {
			where = t.Field + ": "
		}
		return fmt.Sprintf("%sexpected %s, found %s", where, skinTypeName(t.Type.String()), t.Value)
	}
	return err.Error()
}

// skinTypeName names a Go type the way the format's documentation does.
func skinTypeName(t string) string {
	switch t {
	case "float32", "float64", "int":
		return "a number"
	case "string":
		return "text"
	case "bool":
		return "true or false"
	case "[]float32":
		return "an array of numbers"
	}
	if strings.HasPrefix(t, "map[") || strings.HasPrefix(t, "style.") {
		return "an object"
	}
	return t
}

// rect4 reads a [x, y, w, h] / [top, right, bottom, left] array, which is
// how every rect and inset in the format is written (CSS order for insets,
// so an author reading the file is never guessing).
func rect4(key string, v []float32, what string) ([4]float32, error) {
	var out [4]float32
	if len(v) == 0 {
		return out, nil
	}
	if len(v) != 4 {
		return out, skinErr(key, "%s needs 4 numbers, found %d", what, len(v))
	}
	copy(out[:], v)
	return out, nil
}

// insets4 reads [top, right, bottom, left].
func insets4(key string, v []float32, what string) (Insets, error) {
	a, err := rect4(key, v, what)
	if err != nil {
		return Insets{}, err
	}
	return Insets{Top: a[0], Right: a[1], Bottom: a[2], Left: a[3]}, nil
}

// parseSkinFill reads a middle policy.
func parseSkinFill(key, s string) (SkinFill, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "stretch":
		return FillStretch, nil
	case "tile":
		return FillTile, nil
	case "none":
		return FillNone, nil
	}
	return 0, skinErr(key, "middle must be stretch, tile or none, found %q", s)
}

// parseSheetScale reads a scale-set key ("1x", "1.5x", "2x", "3x").
func parseSheetScale(key, s string) (float32, error) {
	t := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), "x")
	v, err := strconv.ParseFloat(t, 32)
	if err != nil || v <= 0 || v > 8 {
		return 0, skinErr(key, "scale key must be a positive scale such as 1x, 1.5x or 2x, found %q", s)
	}
	return float32(v), nil
}

// ---- loading --------------------------------------------------------------

// LoadSkinFS reads a skin from fsys, whose root holds skin.json. It is the
// whole loader: a directory (os.DirFS), an archive (zip.Reader) and a test
// fixture (fstest.MapFS) all arrive here, so there is one code path and one
// set of error messages.
func LoadSkinFS(fsys fs.FS, name string) (*Skin, error) {
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return nil, &SkinError{Skin: name, Msg: err.Error()}
	}
	raw, err := fs.ReadFile(fsys, SkinFile)
	if err != nil {
		return nil, &SkinError{Skin: clean, Msg: "cannot read the manifest: " + err.Error()}
	}
	sk, err := parseSkin(clean, raw, fsys)
	if err != nil {
		if se, ok := err.(*SkinError); ok {
			se.Skin = clean
		}
		return nil, err
	}
	return sk, nil
}

// parseSkin decodes and validates a manifest. Every error it returns names
// the key it came from.
func parseSkin(name string, raw []byte, fsys fs.FS) (*Skin, error) {
	var doc skinFileJSON
	if err := decodeSkinJSON("", raw, &doc); err != nil {
		return nil, err
	}
	switch {
	case doc.Skin == 0:
		return nil, skinErr("skin", "the format version is required (this build reads %d)", SkinVersion)
	case doc.Skin < 1:
		return nil, skinErr("skin", "format version %d is not a version", doc.Skin)
	case doc.Skin > SkinVersion:
		return nil, skinErr("skin", "format version %d is newer than this build reads (%d)", doc.Skin, SkinVersion)
	}

	sk := &Skin{
		Name:    name,
		Version: doc.Skin,
		Label:   strings.TrimSpace(doc.Label),
		Year:    doc.Year,
		Lineage: strings.TrimSpace(doc.Lineage),
		Summary: strings.TrimSpace(doc.Summary),
		Base:    strings.ToLower(strings.TrimSpace(doc.Base)),
		Family:  ParseTheme(doc.Family),
		Design:  SkinDesign{Scale: 1},
		Sheets:  map[string]*SkinSheet{},
		Sprites: map[string]*SkinSprite{},
		Parts:   map[string]*SkinPart{},
		Text:    map[string]*SkinText{},
		fsys:    fsys,
	}
	if sk.Label == "" {
		sk.Label = name
	}
	if doc.Design != nil {
		if doc.Design.Scale < 0 {
			return nil, skinErr("design.scale", "must be positive, found %g", doc.Design.Scale)
		}
		if doc.Design.Scale > 0 {
			sk.Design.Scale = doc.Design.Scale
		}
		sk.Design.Pixelated = doc.Design.Pixelated
	}

	if err := sk.loadSheets(doc.Sheets); err != nil {
		return nil, err
	}
	if err := sk.loadText(doc.Text); err != nil {
		return nil, err
	}
	if err := sk.loadSprites(doc.Sprites); err != nil {
		return nil, err
	}
	if err := sk.loadParts(doc.Parts); err != nil {
		return nil, err
	}
	if err := sk.loadWindow(doc.Window); err != nil {
		return nil, err
	}
	if err := sk.loadTokens(doc); err != nil {
		return nil, err
	}
	return sk, nil
}

func (sk *Skin) loadSheets(m map[string]json.RawMessage) error {
	for name, raw := range m {
		key := joinKey("sheets", name)
		var obj map[string]json.RawMessage
		if err := decodeSkinJSON(key, raw, &obj); err != nil {
			return err
		}
		sh := &SkinSheet{Name: name, Files: map[float32]string{}, Pixelated: sk.Design.Pixelated}
		for k, v := range obj {
			if k == "pixelated" {
				var b bool
				if err := decodeSkinJSON(joinKey(key, k), v, &b); err != nil {
					return err
				}
				sh.Pixelated = b
				continue
			}
			scale, err := parseSheetScale(joinKey(key, k), k)
			if err != nil {
				return err
			}
			var file string
			if err := decodeSkinJSON(joinKey(key, k), v, &file); err != nil {
				return err
			}
			file = path.Clean(strings.TrimSpace(file))
			if file == "" || file == "." {
				return skinErr(joinKey(key, k), "the file name is empty")
			}
			if path.IsAbs(file) || file == ".." || strings.HasPrefix(file, "../") {
				// A manifest may only name art inside its own package: a
				// skin is a stranger's zip file and must not be able to
				// read the user's home directory through a path.
				return skinErr(joinKey(key, k), "%q leaves the skin's directory", file)
			}
			if _, err := fs.Stat(sk.fsys, file); err != nil {
				return skinErr(joinKey(key, k), "missing art file %q", file)
			}
			sh.Files[scale] = file
		}
		if len(sh.Files) == 0 {
			return skinErr(key, "names no art; a sheet needs at least a 1x file")
		}
		for s := range sh.Files {
			sh.scales = append(sh.scales, s)
		}
		sort.Slice(sh.scales, func(i, j int) bool { return sh.scales[i] < sh.scales[j] })
		sk.Sheets[name] = sh
	}
	return nil
}

func (sk *Skin) loadText(m map[string]json.RawMessage) error {
	for name, raw := range m {
		key := joinKey("text", name)
		t, err := sk.parseText(key, raw, name)
		if err != nil {
			return err
		}
		sk.Text[name] = t
	}
	return nil
}

func (sk *Skin) parseText(key string, raw []byte, name string) (*SkinText, error) {
	var doc skinTextJSON
	if err := decodeSkinJSON(key, raw, &doc); err != nil {
		return nil, err
	}
	t := &SkinText{Name: name, Bold: doc.Bold}
	col := func(field, s string, dst *paintengine2d.Color) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		c, ok := ParseHexColor(s)
		if !ok {
			return skinErr(joinKey(key, field), "%q is not a colour (#rgb, #rrggbb or #rrggbbaa)", s)
		}
		*dst = markExplicitColor(c)
		return nil
	}
	for _, f := range []struct {
		name string
		s    string
		dst  *paintengine2d.Color
	}{
		{"color", doc.Color, &t.Color},
		{"hover", doc.Hover, &t.Hover},
		{"pressed", doc.Pressed, &t.Pressed},
		{"disabled", doc.Disabled, &t.Disabled},
		{"checked", doc.Checked, &t.Checked},
	} {
		if err := col(f.name, f.s, f.dst); err != nil {
			return nil, err
		}
	}
	if doc.Size != nil {
		if *doc.Size <= 0 || *doc.Size > 200 {
			return nil, skinErr(joinKey(key, "size"), "must be between 1 and 200 design pixels, found %g", *doc.Size)
		}
		t.Size = *doc.Size
	}
	return t, nil
}

func (sk *Skin) loadSprites(m map[string]json.RawMessage) error {
	for name, raw := range m {
		sp, err := sk.parseSprite(joinKey("sprites", name), raw, name)
		if err != nil {
			return err
		}
		sk.Sprites[name] = sp
	}
	return nil
}

func (sk *Skin) parseSprite(key string, raw []byte, name string) (*SkinSprite, error) {
	var doc skinSpriteJSON
	if err := decodeSkinJSON(key, raw, &doc); err != nil {
		return nil, err
	}
	sheetName := strings.TrimSpace(doc.Sheet)
	if sheetName == "" {
		if len(sk.Sheets) != 1 {
			return nil, skinErr(joinKey(key, "sheet"), "is required (the skin has %d sheets)", len(sk.Sheets))
		}
		for n := range sk.Sheets {
			sheetName = n
		}
	}
	sh, ok := sk.Sheets[sheetName]
	if !ok {
		return nil, skinErr(joinKey(key, "sheet"), "no sheet %q", sheetName)
	}
	sp := &SkinSprite{Name: name, Sheet: sh, Tint: doc.Tint}
	switch {
	case len(doc.At) > 0:
		at, err := rect4(joinKey(key, "at"), doc.At, "at")
		if err != nil {
			return nil, err
		}
		sp.X, sp.Y, sp.W, sp.H = at[0], at[1], at[2], at[3]
	case doc.W != nil && doc.H != nil:
		if doc.X != nil {
			sp.X = *doc.X
		}
		if doc.Y != nil {
			sp.Y = *doc.Y
		}
		sp.W, sp.H = *doc.W, *doc.H
	default:
		return nil, skinErr(key, `needs "at": [x, y, w, h]`)
	}
	if sp.W <= 0 || sp.H <= 0 {
		return nil, skinErr(joinKey(key, "at"), "is empty (%g×%g)", sp.W, sp.H)
	}
	if sp.X < 0 || sp.Y < 0 {
		return nil, skinErr(joinKey(key, "at"), "starts outside the sheet (%g, %g)", sp.X, sp.Y)
	}
	in, err := insets4(joinKey(key, "slice"), doc.Slice, "slice")
	if err != nil {
		return nil, err
	}
	if in.Left+in.Right >= sp.W || in.Top+in.Bottom >= sp.H {
		return nil, skinErr(joinKey(key, "slice"), "leaves no middle: %g+%g wide and %g+%g tall in a %g×%g sprite",
			in.Left, in.Right, in.Top, in.Bottom, sp.W, sp.H)
	}
	if in.Left < 0 || in.Right < 0 || in.Top < 0 || in.Bottom < 0 {
		return nil, skinErr(joinKey(key, "slice"), "insets cannot be negative")
	}
	sp.Slice = in
	if sp.Middle, err = parseSkinFill(joinKey(key, "middle"), doc.Middle); err != nil {
		return nil, err
	}
	return sp, nil
}

func (sk *Skin) loadParts(m map[string]json.RawMessage) error {
	for name, raw := range m {
		key := joinKey("parts", name)
		if _, ok := skinPartNames[name]; !ok {
			return skinErr(key, "%q is not a part of this toolkit (see SkinPartNames; new controls come from the toolkit, not the skin)", name)
		}
		p, err := sk.parsePart(key, raw, name)
		if err != nil {
			return err
		}
		sk.Parts[name] = p
	}
	return nil
}

func (sk *Skin) parsePart(key string, raw []byte, name string) (*SkinPart, error) {
	var doc skinPartJSON
	if err := decodeSkinJSON(key, raw, &doc); err != nil {
		return nil, err
	}
	p := &SkinPart{Name: name, States: map[string]*SkinSprite{}}

	if doc.Strip != nil {
		if err := sk.expandStrip(joinKey(key, "strip"), doc.Strip, p); err != nil {
			return nil, err
		}
	}
	for state, raw := range doc.States {
		sk2 := joinKey(joinKey(key, "states"), state)
		if !skinStateSet[state] {
			return nil, skinErr(sk2, "%q is not a control state (one of %s)", state, strings.Join(skinStateNames, ", "))
		}
		sp, err := sk.spriteRef(sk2, raw, name+"."+state)
		if err != nil {
			return nil, err
		}
		p.States[state] = sp
	}
	if len(p.States) > 0 && p.States["normal"] == nil {
		return nil, skinErr(joinKey(key, "states"), `needs "normal": every other state falls back to it`)
	}

	if len(doc.Text) > 0 {
		t, err := sk.textRef(joinKey(key, "text"), doc.Text, name)
		if err != nil {
			return nil, err
		}
		p.Text = t
	}
	pad, err := insets4(joinKey(key, "pad"), doc.Pad, "pad")
	if err != nil {
		return nil, err
	}
	p.Pad = pad
	return p, nil
}

// expandStrip lays a run of equal cells across (or down) a sheet, one per
// state. It is the shorthand real sheets are drawn for.
func (sk *Skin) expandStrip(key string, s *skinStripJSON, p *SkinPart) error {
	if len(s.States) == 0 {
		return skinErr(joinKey(key, "states"), "names no states")
	}
	at, err := rect4(joinKey(key, "at"), s.At, "at")
	if err != nil {
		return err
	}
	if len(s.At) == 0 {
		return skinErr(joinKey(key, "at"), `is required: [x, y, w, h] of the first cell`)
	}
	sheetName := strings.TrimSpace(s.Sheet)
	if sheetName == "" && len(sk.Sheets) == 1 {
		for n := range sk.Sheets {
			sheetName = n
		}
	}
	sh, ok := sk.Sheets[sheetName]
	if !ok {
		return skinErr(joinKey(key, "sheet"), "no sheet %q", sheetName)
	}
	in, err := insets4(joinKey(key, "slice"), s.Slice, "slice")
	if err != nil {
		return err
	}
	fill, err := parseSkinFill(joinKey(key, "middle"), s.Middle)
	if err != nil {
		return err
	}
	x, y, w, h := at[0], at[1], at[2], at[3]
	if w <= 0 || h <= 0 {
		return skinErr(joinKey(key, "at"), "cell is empty (%g×%g)", w, h)
	}
	if in.Left+in.Right >= w || in.Top+in.Bottom >= h {
		return skinErr(joinKey(key, "slice"), "leaves no middle in a %g×%g cell", w, h)
	}
	for i, state := range s.States {
		// "-" skips a cell, so a strip whose sheet has a gap in it still
		// reads as the run of cells it is.
		if state == "-" {
			continue
		}
		if !skinStateSet[state] {
			return skinErr(joinKey(key, "states"), "%q is not a control state (one of %s)", state, strings.Join(skinStateNames, ", "))
		}
		step := float32(i) * (w + s.Gap)
		sp := &SkinSprite{
			Name: p.Name + "." + state, Sheet: sh,
			X: x, Y: y, W: w, H: h,
			Slice: in, Middle: fill, Tint: s.Tint,
		}
		if s.Down {
			sp.Y += step
		} else {
			sp.X += step
		}
		p.States[state] = sp
		// A strip's cells are addressable by name too, so another part can
		// share one without repeating its rect.
		if _, dup := sk.Sprites[sp.Name]; !dup {
			sk.Sprites[sp.Name] = sp
		}
	}
	return nil
}

// spriteRef reads either a sprite name or an inline sprite object, so a
// one-off needs no entry in the sprite table and a shared one needs no
// repetition.
func (sk *Skin) spriteRef(key string, raw []byte, inlineName string) (*SkinSprite, error) {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		sp, ok := sk.Sprites[name]
		if !ok {
			return nil, skinErr(key, "no sprite %q", name)
		}
		return sp, nil
	}
	return sk.parseSprite(key, raw, inlineName)
}

// textRef reads either a text-role name or an inline role.
func (sk *Skin) textRef(key string, raw []byte, inlineName string) (*SkinText, error) {
	var name string
	if err := json.Unmarshal(raw, &name); err == nil {
		t, ok := sk.Text[name]
		if !ok {
			return nil, skinErr(key, "no text role %q", name)
		}
		return t, nil
	}
	return sk.parseText(key, raw, inlineName)
}

func (sk *Skin) loadWindow(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var doc skinWindowJSON
	if err := decodeSkinJSON("window", raw, &doc); err != nil {
		return err
	}
	w := &SkinWindow{Layout: strings.TrimSpace(doc.Layout)}
	border, err := insets4("window.border", doc.Border, "border")
	if err != nil {
		return err
	}
	w.Border = border
	if doc.Caption != nil {
		if *doc.Caption < 0 {
			return skinErr("window.caption", "cannot be negative")
		}
		w.Caption = *doc.Caption
	}
	if w.Radius, err = rect4("window.radius", doc.Radius, "radius"); err != nil {
		return err
	}
	for i, r := range doc.Shape {
		key := fmt.Sprintf("window.shape[%d]", i)
		at, err := rect4(joinKey(key, "at"), r.At, "at")
		if err != nil {
			return err
		}
		if len(r.At) == 0 {
			return skinErr(joinKey(key, "at"), "is required: [x, y, w, h]")
		}
		// Under stretchX / stretchY the third and fourth numbers are margins
		// from the far edge rather than a size, so zero is a rect that runs
		// to the window's edge — the commonest shape there is.
		if (at[2] <= 0 && !r.StretchX) || (at[3] <= 0 && !r.StretchY) {
			return skinErr(joinKey(key, "at"), "is empty (%g×%g); with stretchX / stretchY the size is a margin and may be 0", at[2], at[3])
		}
		if at[2] < 0 || at[3] < 0 {
			return skinErr(joinKey(key, "at"), "cannot be negative")
		}
		rad, err := rect4(joinKey(key, "radius"), r.Radius, "radius")
		if err != nil {
			return err
		}
		w.Shape = append(w.Shape, SkinShapeRect{
			X: at[0], Y: at[1], W: at[2], H: at[3],
			Radius: rad, StretchX: r.StretchX, StretchY: r.StretchY,
		})
	}
	sk.Window = w
	return nil
}

// loadTokens reads the colours, metrics and params a skin states for what
// art does not cover. They are theme.json's own keys, decoded by theme.go's
// own code, so a skin author who has written a pack already knows them.
func (sk *Skin) loadTokens(doc skinFileJSON) error {
	for k, v := range doc.Colors {
		if _, ok := ParseHexColor(v); !ok {
			return skinErr(joinKey("colors", k), "%q is not a colour (#rgb, #rrggbb or #rrggbbaa)", v)
		}
	}
	for k, v := range doc.Extra {
		if _, ok := ParseHexColor(v); !ok {
			return skinErr(joinKey("extra", k), "%q is not a colour (#rgb, #rrggbb or #rrggbbaa)", v)
		}
	}
	sk.Tokens = tokensFromJSON(themeFileJSON{
		Engine:  skinEngineID,
		Family:  string(sk.Family),
		Bevel:   doc.Bevel,
		Colors:  doc.Colors,
		Metrics: doc.Metrics,
		Params:  doc.Params,
		Extra:   doc.Extra,
		Fonts:   doc.Fonts,
	})
	// ParseBevel answers BevelNone for anything it does not know, including
	// the empty string. A skin that says nothing about bevels must keep its
	// base pack's — the flat fallback is a choice, not a default — so the
	// unstated case is cleared here and mergeTokens lets the base through.
	if strings.TrimSpace(doc.Bevel) == "" {
		sk.Tokens.Bevel = ""
	}
	return nil
}

// ---- reading an archive ---------------------------------------------------

// openSkinArchive reads a .uskin (a zip of the skin's directory). Archives
// whose manifest sits in a single top-level folder — which is what most
// archivers produce — are rooted at that folder, so a skin zipped either way
// loads.
func openSkinArchive(path string) (fs.FS, func() error, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, nil, err
	}
	var root fs.FS = zr
	if _, err := fs.Stat(root, SkinFile); err != nil {
		if dir, ok := singleSkinRoot(zr); ok {
			if sub, err := fs.Sub(root, dir); err == nil {
				root = sub
			}
		}
	}
	return root, zr.Close, nil
}

// singleSkinRoot finds the one top-level folder holding a manifest.
func singleSkinRoot(zr *zip.ReadCloser) (string, bool) {
	var found string
	for _, f := range zr.File {
		dir, base := path.Split(f.Name)
		if base != SkinFile {
			continue
		}
		dir = strings.TrimSuffix(dir, "/")
		if dir == "" || strings.Contains(dir, "/") {
			continue
		}
		if found != "" && found != dir {
			return "", false // two candidates: refuse to guess
		}
		found = dir
	}
	return found, found != ""
}
