// Package skinart draws the sprite sheets of the toolkit's own demo skins,
// and writes the skin.json that binds them.
//
// The art is generated rather than hand-painted for three reasons, and all
// three are about being able to say something true about it:
//
//   - It is demonstrably ours. Every pixel comes from the paths and
//     gradients in this package, so there is no question of where a bitmap
//     came from. The toolkit's licensing rule for engines — facts, never
//     transcribed code or artwork — extends to skins, and this is how it is
//     kept.
//   - It is regenerable at any scale. The 1× and 2× sheets are the same
//     drawing at two scales, and a 3× set is one flag away.
//   - The manifest cannot drift from the art. The cell layout is stated once,
//     in Go, and both the PNGs and skin.json are emitted from it — so a
//     sprite rect in skin.json is never off by the two pixels somebody moved
//     a cell by.
//
// `go run ./cmd/uitk-skingen` rewrites style/skins/; TestSkinArtIsReproducible
// regenerates into a temporary directory and compares, so a change to the
// drawing that was not committed fails the build.
package skinart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// SkinManifestName is the manifest's file name, kept here so the generator
// and its tests do not depend on the style package.
const SkinManifestName = "skin.json"

// Scales are the scale set every demo skin ships. 1 and 2 are the format's
// required pair: art is chosen upward and drawn down, so a 2× sheet covers
// every fractional scale up to 2 exactly.
var Scales = []float32{1, 2}

// ---- the plan -------------------------------------------------------------

// Cell is one sprite: where it sits on the sheet, how it slices, and how it
// is drawn. Coordinates are design pixels at 1×.
type Cell struct {
	Name       string
	X, Y, W, H int
	// Slice is the nine-slice inset (top, right, bottom, left); zero means
	// the sprite stretches whole.
	Slice [4]int
	// Middle is "", "tile" or "none".
	Middle string
	// Tint means the sprite is a single-colour glyph painted in the label's
	// colour, with its own alpha as coverage.
	Tint bool
	// Draw paints the cell into a context already translated to its origin
	// and scaled, so it draws in design pixels from (0, 0).
	Draw func(ctx *paintengine2d.Context, w, h float32)
}

// Sheet is one sprite sheet and its cells.
type Sheet struct {
	Name      string
	W, H      int
	Pixelated bool
	Cells     []Cell
}

// Plan is a whole skin: its identity, its sheets, and the bindings the
// manifest will carry.
type Plan struct {
	Name    string
	Label   string
	Year    int
	Lineage string
	Summary string
	Family  string
	Base    string

	Sheets []*Sheet
	// Parts maps a toolkit part name onto its state bindings. The value is
	// a state name → sprite name map; sprite names are the Cell names.
	Parts []PartBinding
	// Text are the named label roles.
	Text []TextRole
	// Colors, Metrics and Params are what art does not cover.
	Colors  map[string]string
	Metrics map[string]float32
	Params  map[string]float32
	Window  *WindowSpec
}

// PartBinding binds one part.
type PartBinding struct {
	Part   string
	States [][2]string // ordered {state, sprite} pairs, so the file is stable
	Text   string
	Pad    [4]int
}

// TextRole is one named label role.
type TextRole struct {
	Name                                              string
	Color, Hover, Pressed, Disabled, Checked, Default string
	Size                                              float32
	Bold                                              bool
}

// WindowSpec is the frame the skin asks for.
type WindowSpec struct {
	Border  [4]int
	Caption int
	Layout  string
	Radius  [4]int
	Shape   []ShapeRect
}

// ShapeRect is one rounded rect of the silhouette the skin declares for the
// day the toolkit can cut a window to it.
type ShapeRect struct {
	At                 [4]int
	Radius             [4]int
	StretchX, StretchY bool
}

// ---- rendering ------------------------------------------------------------

// Render draws one sheet at a scale.
//
// A pixelated sheet is drawn once at 1× and then doubled exactly, so its 2×
// asset is the same picture with square pixels rather than a re-rasterised
// approximation of it — which is the whole point of declaring a sheet
// pixelated in the first place.
func Render(sh *Sheet, scale float32) *paintengine2d.Image {
	if sh.Pixelated && scale != 1 {
		n := int(math.Round(float64(scale)))
		if n < 1 {
			n = 1
		}
		return magnify(Render(sh, 1), n)
	}
	w := int(math.Round(float64(float32(sh.W) * scale)))
	h := int(math.Round(float64(float32(sh.H) * scale)))
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	for _, c := range sh.Cells {
		if c.Draw == nil {
			continue
		}
		ctx.Save()
		ctx.Translate(float32(c.X)*scale, float32(c.Y)*scale)
		ctx.Scale(scale, scale)
		ctx.ClipRect(paintengine2d.XYWH(0, 0, float32(c.W), float32(c.H)))
		c.Draw(ctx, float32(c.W), float32(c.H))
		ctx.Restore()
	}
	return img
}

// magnify repeats every pixel n times: an exact integer enlargement, which
// is what nearest-neighbour would do at draw time and what a pixel-art sheet
// wants baked in.
func magnify(src *paintengine2d.Image, n int) *paintengine2d.Image {
	out := paintengine2d.NewImage(src.Width*n, src.Height*n)
	for y := 0; y < src.Height; y++ {
		for x := 0; x < src.Width; x++ {
			r, g, b, a := src.PremulAt(x, y)
			for dy := 0; dy < n; dy++ {
				row := (y*n + dy) * out.RowStride()
				for dx := 0; dx < n; dx++ {
					i := row + (x*n+dx)*4
					out.Pix[i+0], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = r, g, b, a
				}
			}
		}
	}
	out.Touch()
	return out
}

// ---- writing --------------------------------------------------------------

// Write renders a plan into dir/<name>/: the sheets under art/, and the
// manifest beside them. Files are written only when their bytes change, so
// re-running the generator does not churn mtimes (and does not make the look
// watcher think every skin was edited).
func Write(dir string, p *Plan) error {
	root := filepath.Join(dir, p.Name)
	if err := os.MkdirAll(filepath.Join(root, "art"), 0o755); err != nil {
		return err
	}
	for _, sh := range p.Sheets {
		for _, s := range Scales {
			img := Render(sh, s)
			var buf bytes.Buffer
			if err := img.WritePNG(&buf); err != nil {
				return fmt.Errorf("%s %s@%gx: %w", p.Name, sh.Name, s, err)
			}
			if err := writeIfChanged(filepath.Join(root, sheetFile(sh.Name, s)), buf.Bytes()); err != nil {
				return err
			}
		}
	}
	doc, err := Manifest(p)
	if err != nil {
		return err
	}
	return writeIfChanged(filepath.Join(root, SkinManifestName), doc)
}

// sheetFile is a sheet's path inside the skin, by the same @2x convention
// the toolkit's icon sets already use.
func sheetFile(name string, scale float32) string {
	if scale == 1 {
		return filepath.Join("art", name+".png")
	}
	return filepath.Join("art", fmt.Sprintf("%s@%gx.png", name, scale))
}

func writeIfChanged(path string, want []byte) error {
	if got, err := os.ReadFile(path); err == nil && bytes.Equal(got, want) {
		return nil
	}
	return os.WriteFile(path, want, 0o644)
}

// ---- the manifest ---------------------------------------------------------

// Manifest renders a plan as skin.json: indented, key-ordered and stable, so
// it reads as a worked example of the format and diffs as one line per
// change.
func Manifest(p *Plan) ([]byte, error) {
	doc := map[string]any{"skin": 1}
	put := func(k string, v any) { doc[k] = v }
	put("label", p.Label)
	if p.Year > 0 {
		put("year", p.Year)
	}
	if p.Lineage != "" {
		put("lineage", p.Lineage)
	}
	if p.Summary != "" {
		put("summary", p.Summary)
	}
	if p.Family != "" {
		put("family", p.Family)
	}
	if p.Base != "" {
		put("base", p.Base)
	}

	sheets := map[string]any{}
	sprites := map[string]any{}
	for _, sh := range p.Sheets {
		files := map[string]any{}
		for _, s := range Scales {
			files[scaleKey(s)] = filepath.ToSlash(sheetFile(sh.Name, s))
		}
		if sh.Pixelated {
			files["pixelated"] = true
			put("design", map[string]any{"pixelated": true})
		}
		sheets[sh.Name] = files
		sortCells(sh.Cells)
		for _, c := range sh.Cells {
			sp := map[string]any{"sheet": sh.Name, "at": []int{c.X, c.Y, c.W, c.H}}
			if c.Slice != [4]int{} {
				sp["slice"] = []int{c.Slice[0], c.Slice[1], c.Slice[2], c.Slice[3]}
			}
			if c.Middle != "" {
				sp["middle"] = c.Middle
			}
			if c.Tint {
				sp["tint"] = true
			}
			sprites[c.Name] = sp
		}
	}
	put("sheets", sheets)
	put("sprites", sprites)

	if len(p.Text) > 0 {
		text := map[string]any{}
		for _, t := range p.Text {
			r := map[string]any{}
			for _, kv := range [][2]string{
				{"color", t.Color}, {"hover", t.Hover}, {"pressed", t.Pressed},
				{"disabled", t.Disabled}, {"checked", t.Checked},
				{"default", t.Default},
			} {
				if kv[1] != "" {
					r[kv[0]] = kv[1]
				}
			}
			if t.Size > 0 {
				r["size"] = t.Size
			}
			if t.Bold {
				r["bold"] = true
			}
			text[t.Name] = r
		}
		put("text", text)
	}

	parts := map[string]any{}
	for _, b := range p.Parts {
		states := map[string]any{}
		for _, kv := range b.States {
			states[kv[0]] = kv[1]
		}
		obj := map[string]any{"states": states}
		if b.Text != "" {
			obj["text"] = b.Text
		}
		if b.Pad != [4]int{} {
			obj["pad"] = []int{b.Pad[0], b.Pad[1], b.Pad[2], b.Pad[3]}
		}
		parts[b.Part] = obj
	}
	put("parts", parts)

	if p.Window != nil {
		w := map[string]any{}
		if p.Window.Border != [4]int{} {
			w["border"] = p.Window.Border[:]
		}
		if p.Window.Caption > 0 {
			w["caption"] = p.Window.Caption
		}
		if p.Window.Layout != "" {
			w["layout"] = p.Window.Layout
		}
		if p.Window.Radius != [4]int{} {
			w["radius"] = p.Window.Radius[:]
		}
		for _, r := range p.Window.Shape {
			e := map[string]any{"at": r.At[:]}
			if r.Radius != [4]int{} {
				e["radius"] = r.Radius[:]
			}
			if r.StretchX {
				e["stretchX"] = true
			}
			if r.StretchY {
				e["stretchY"] = true
			}
			w["shape"] = append(asSlice(w["shape"]), e)
		}
		put("window", w)
	}
	if len(p.Colors) > 0 {
		put("colors", p.Colors)
	}
	if len(p.Metrics) > 0 {
		put("metrics", p.Metrics)
	}
	if len(p.Params) > 0 {
		put("params", p.Params)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return inlineNumberArrays(buf.Bytes()), nil
}

// inlineNumberArrays puts short arrays of numbers back on one line.
//
// encoding/json indents every array element, which turns `"at": [0, 0, 56,
// 30]` into five lines and a sprite table into something nobody reads. The
// manifest is meant to be a worked example of the format, so a rect stays a
// rect.
func inlineNumberArrays(b []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(b))
	for i := 0; i < len(b); {
		if b[i] != '[' {
			out.WriteByte(b[i])
			i++
			continue
		}
		end := bytes.IndexByte(b[i:], ']')
		if end < 0 {
			out.WriteByte(b[i])
			i++
			continue
		}
		body := b[i+1 : i+end]
		if !onlyNumbers(body) {
			out.WriteByte(b[i])
			i++
			continue
		}
		out.WriteByte('[')
		out.WriteString(strings.Join(strings.Fields(strings.ReplaceAll(string(body), ",", " ")), ", "))
		out.WriteByte(']')
		i += end + 1
	}
	return out.Bytes()
}

func onlyNumbers(b []byte) bool {
	digits := false
	for _, c := range b {
		switch {
		case c >= '0' && c <= '9':
			digits = true
		case c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' ||
			c == ',' || c == ' ' || c == '\n' || c == '\t' || c == '\r':
		default:
			return false
		}
	}
	return digits
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func scaleKey(s float32) string {
	if s == float32(int(s)) {
		return fmt.Sprintf("%dx", int(s))
	}
	return fmt.Sprintf("%gx", s)
}

// ---- drawing helpers ------------------------------------------------------
//
// Small and literal, like style/shapes.go: each one is an idiom the two demo
// skins use several times, so the drawing below reads as a description of
// the look rather than as a pile of coordinates.

// hex parses #rgb / #rrggbb / #rrggbbaa into a straight-alpha colour.
func hex(s string) paintengine2d.Color {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	val := func(i int) float32 {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			return float32(c - '0')
		case c >= 'a' && c <= 'f':
			return float32(c-'a') + 10
		case c >= 'A' && c <= 'F':
			return float32(c-'A') + 10
		}
		return 0
	}
	b2 := func(i int) float32 { return (val(i)*16 + val(i+1)) / 255 }
	switch len(s) {
	case 3:
		return paintengine2d.RGB(val(0)/15, val(1)/15, val(2)/15)
	case 6:
		return paintengine2d.RGB(b2(0), b2(2), b2(4))
	case 8:
		return paintengine2d.RGBA(b2(0), b2(2), b2(4), b2(6))
	}
	return paintengine2d.Color{}
}

// vgrad fills a rounded rect with a top-to-bottom gradient.
func vgrad(ctx *paintengine2d.Context, r paintengine2d.Rect, radius float32, stops ...paintengine2d.GradientStop) {
	ctx.DrawRoundRect(r, radius, radius, paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(r.Min.X, r.Min.Y),
		End:   paintengine2d.Pt(r.Min.X, r.Max.Y),
		Stops: stops,
	}))
}

func stop(at float32, c paintengine2d.Color) paintengine2d.GradientStop {
	return paintengine2d.GradientStop{Offset: at, Color: c}
}

// outline strokes a rounded rect on the half-pixel grid, so a 1px line lands
// on one row of pixels instead of two half-lit ones.
func outline(ctx *paintengine2d.Context, r paintengine2d.Rect, radius, width float32, col paintengine2d.Color) {
	ctx.DrawRoundRect(r.Inset(width*0.5), radius, radius, paintengine2d.StrokePaint(col, width))
}

// topLight is the one-pixel highlight along the inside of a face's top edge:
// the cheapest possible suggestion that a surface is lit from above, and the
// detail every raised-button look of the last thirty years has had.
func topLight(ctx *paintengine2d.Context, r paintengine2d.Rect, radius float32, col paintengine2d.Color) {
	ctx.Save()
	ctx.ClipRoundRect(r.Inset(1), radius, radius)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y+1, r.Dx(), 1), paintengine2d.Fill(col))
	ctx.Restore()
}

// innerShadow darkens the inside of a sunken face's top edge.
func innerShadow(ctx *paintengine2d.Context, r paintengine2d.Rect, radius, depth float32, col paintengine2d.Color) {
	ctx.Save()
	ctx.ClipRoundRect(r, radius, radius)
	ctx.DrawRect(paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), depth), paintengine2d.Linear(paintengine2d.LinearGradient{
		Start: paintengine2d.Pt(r.Min.X, r.Min.Y),
		End:   paintengine2d.Pt(r.Min.X, r.Min.Y+depth),
		Stops: []paintengine2d.GradientStop{stop(0, col), stop(1, col.WithAlpha(0))},
	}))
	ctx.Restore()
}

// px fills whole design pixels — the pixel skin's only drawing primitive.
func px(ctx *paintengine2d.Context, x, y, w, h float32, col paintengine2d.Color) {
	ctx.DrawRect(paintengine2d.XYWH(x, y, w, h), paintengine2d.Fill(col))
}

// bevel draws a hard two-tone pixel bevel: light on the top and left, dark
// on the bottom and right, exactly as every 16-bit interface did it.
func bevel(ctx *paintengine2d.Context, w, h, t float32, light, dark paintengine2d.Color) {
	px(ctx, 0, 0, w, t, light)
	px(ctx, 0, 0, t, h, light)
	px(ctx, 0, h-t, w, t, dark)
	px(ctx, w-t, 0, t, h, dark)
}

// chevron strokes an arrow glyph pointing in one of four directions, drawn
// white so the engine can tint it to whatever the label's colour is.
func chevron(ctx *paintengine2d.Context, w, h float32, dir string, width float32, col paintengine2d.Color) {
	cx, cy := w*0.5, h*0.5
	s := min(w, h) * 0.26
	p := paintengine2d.NewPath()
	switch dir {
	case "up":
		p.MoveTo(cx-s, cy+s*0.6)
		p.LineTo(cx, cy-s*0.6)
		p.LineTo(cx+s, cy+s*0.6)
	case "left":
		p.MoveTo(cx+s*0.6, cy-s)
		p.LineTo(cx-s*0.6, cy)
		p.LineTo(cx+s*0.6, cy+s)
	case "right":
		p.MoveTo(cx-s*0.6, cy-s)
		p.LineTo(cx+s*0.6, cy)
		p.LineTo(cx-s*0.6, cy+s)
	default:
		p.MoveTo(cx-s, cy-s*0.6)
		p.LineTo(cx, cy+s*0.6)
		p.LineTo(cx+s, cy-s*0.6)
	}
	st := paintengine2d.StrokePaint(col, width)
	st.Stroke.Cap = paintengine2d.CapRound
	st.Stroke.Join = paintengine2d.JoinRound
	ctx.DrawPath(p, st)
}

// ---- laying a sheet out ---------------------------------------------------

// lay packs cells into rows left to right and keeps the sheet's size. The
// gap between cells is not decoration: a bilinear tap reads the texel next
// to the one it wants, so a cell that abuts its neighbour can bleed into it
// under a downscale. The skin engine cuts every sprite into an image of its
// own, which removes the problem at draw time — the gap removes it at
// authoring time as well, and makes the sheet readable when a person opens
// it in an editor.
type lay struct {
	sh   *Sheet
	gap  int
	x, y int
	rowH int
	open bool

	// faceW, faceH and faceSlice are the defaults for face(), so a skin's
	// control cells are one grid.
	faceW, faceH int
	faceSlice    [4]int
}

// row starts a new row of the given height.
func (l *lay) row(h int) {
	if l.open {
		l.y += l.rowH + l.gap
	}
	l.open = true
	l.x, l.rowH = 0, h
}

// cell places one sprite and advances.
func (l *lay) cell(name string, w, h int, slice [4]int, middle string, tint bool, draw func(*paintengine2d.Context, float32, float32)) {
	if !l.open {
		l.row(h)
	}
	if h > l.rowH {
		l.rowH = h
	}
	l.sh.Cells = append(l.sh.Cells, Cell{
		Name: name, X: l.x, Y: l.y, W: w, H: h,
		Slice: slice, Middle: middle, Tint: tint, Draw: draw,
	})
	if l.x+w > l.sh.W {
		l.sh.W = l.x + w
	}
	l.x += w + l.gap
}

// face places a control face at the sheet's standard cell size.
func (l *lay) face(name string, draw func(*paintengine2d.Context, float32, float32)) {
	w, h, s := l.faceW, l.faceH, l.faceSlice
	if w == 0 {
		w, h, s = nocW, nocH, nocSlice
	}
	l.cell(name, w, h, s, "", false, draw)
}

// glyph places a square single-colour sprite the engine tints at paint time.
func (l *lay) glyph(name string, size int, draw func(*paintengine2d.Context, float32, float32)) {
	l.cell(name, size, size, [4]int{}, "", true, draw)
}

// close finishes the sheet, fixing its height.
func (l *lay) close() { l.sh.H = l.y + l.rowH }

// sortCells puts a sheet's cells in a stable order, so a regenerated sheet
// is byte-identical to the one before it.
func sortCells(cells []Cell) {
	sort.SliceStable(cells, func(i, j int) bool { return cells[i].Name < cells[j].Name })
}

// Plans is every demo skin the toolkit ships, in the order they are written.
//
// The first three are the format's own worked examples; the last three are
// the ones the player demos wear (internal/players). They are generated the
// same way and ship the same way for the same reason: a demo whose skin had
// to be installed by hand before it looked like anything is a demo nobody
// runs, and a skin the toolkit ships is a skin the reproducibility test
// regenerates and compares.
func Plans() []*Plan {
	return []*Plan{Nocturne(), Cassette(), Deck(), Minim(), Marquee(), Lantern()}
}
