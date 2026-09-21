package style

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A look's silhouette: which looks declare one, where it is dropped, what a
// skin's rect union resolves to, and what a skinned control's art says about
// where the control is.
//
// Everything here is geometry and coverage, read back by rasterising what the
// look said. The other half of the story — the window system being told, the
// press falling through — is app/shape_test.go and widget/shape_test.go.

// ---- reading a silhouette back ---------------------------------------------

// silCoverage rasterises a silhouette at w×h and hands back one byte a pixel,
// so a test can ask what the look actually described rather than what its
// arithmetic looks like.
//
// A mask-backed silhouette is read at its own size (a control's mask is
// painted at exactly the box it was asked about); a path is drawn white on
// nothing and its alpha taken, which is what platform.Shape does.
func silCoverage(t *testing.T, s *Silhouette, w, h int) []uint8 {
	t.Helper()
	if s == nil {
		t.Fatal("no silhouette")
	}
	out := make([]uint8, w*h)
	if s.Mask != nil {
		if s.Mask.Width != w || s.Mask.Height != h {
			t.Fatalf("mask is %d×%d, asked about %d×%d", s.Mask.Width, s.Mask.Height, w, h)
		}
		stride, bpp := s.Mask.RowStride(), s.Mask.BytesPerPixel()
		for y := 0; y < h; y++ {
			row := s.Mask.Pix[y*stride:]
			for x := 0; x < w; x++ {
				if bpp == 1 {
					out[y*w+x] = row[x]
				} else {
					out[y*w+x] = row[x*bpp+3]
				}
			}
		}
		return out
	}
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	paint := paintengine2d.Fill(paintengine2d.White)
	paint.AntiAlias = true
	if s.EvenOdd {
		paint.FillRule = paintengine2d.FillEvenOdd
	}
	ctx.DrawPath(s.Path, paint)
	stride := img.RowStride()
	for y := 0; y < h; y++ {
		row := img.Pix[y*stride:]
		for x := 0; x < w; x++ {
			out[y*w+x] = row[x*4+3]
		}
	}
	return out
}

// inside reports whether the pixel is at least half covered — the same
// threshold platform.Shape uses to decide the input region, so a test and a
// compositor agree about where the window is.
func inside(cov []uint8, w, h, x, y int) bool {
	if x < 0 || y < 0 || x >= w || y >= h {
		return false
	}
	return cov[y*w+x] >= 128
}

// probeFrame is a window box with a caption band across it, in the
// coordinates WindowShape is handed: the window at the origin.
func probeFrame(w, h, capW, capH float32) DecorationFrame {
	return DecorationFrame{
		Window:  paintengine2d.XYWH(0, 0, w, h),
		Caption: paintengine2d.XYWH(0, 0, capW, capH),
	}
}

// ---- which looks say anything at all ----------------------------------------

// The regression test over every pack: a look that does not declare a
// silhouette frames a rectangle with a full-width caption, and nothing about
// this feature changed that for any of them.
//
// Six packs say otherwise, on purpose and by name: BeOS, whose tab is its
// window's real outline; Deck, the skin drawn to demonstrate the format; and
// the four the player demos wear — Minim's stepped chin, Minim Silver's four
// round corners, Marquee's brow and dome, and Lantern's asymmetric skirt.
// Nocturne, Cassette and Minim Classic carry the hook (every skin does) and
// declare no shape, which is the case worth pinning: having the hook is not
// having an outline.
func TestOnlyTheLooksThatMeanToDeclareASilhouette(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	shaped := map[string]bool{
		"beos": true, "deck": true,
		"minim": true, "minim-silver": true, "marquee": true, "lantern": true,
	}
	fitted := map[string]bool{"beos": true}
	st := DecorationState{Active: true}
	for _, p := range ListBuiltinThemes() {
		for _, scale := range []float32{1, 1.75} {
			lk := WithScale(p.Look(), scale)
			f := probeFrame(600*scale, 400*scale, 240*scale, 40*scale)
			if got := WindowShapeOf(lk, f, st) != nil; got != shaped[p.Name] {
				t.Errorf("%s@%gx: declares a silhouette = %v, wanted %v", p.Name, scale, got, shaped[p.Name])
			}
			if got := DecorationOf(lk, st).CaptionFits; got != fitted[p.Name] {
				t.Errorf("%s@%gx: fitted caption = %v, wanted %v", p.Name, scale, got, fitted[p.Name])
			}
		}
	}
}

// And the control half of the same promise: a component takes input across
// its whole box in every pack the toolkit ships. Only a skin says otherwise,
// because only a skin has art to read it from.
func TestOnlyASkinShapesItsControls(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	box := paintengine2d.XYWH(0, 0, 80, 30)
	for _, p := range ListBuiltinThemes() {
		lk := p.Look()
		skin := p.Tokens.Engine == skinEngineID
		if got := ControlShapes(lk); got != skin {
			t.Errorf("%s: shapes its controls = %v, wanted %v", p.Name, got, skin)
		}
		if skin {
			continue
		}
		for _, role := range []Role{RoleButton, RoleTool, RoleField, RoleCheck, RoleTab, RolePanel} {
			if ControlShapeOf(lk, box, role) != nil {
				t.Errorf("%s: role %v is not its whole box", p.Name, role)
			}
		}
	}
}

// ---- the states that drop it -------------------------------------------------

// A silhouette is dropped where the frame's corners and shadow are dropped,
// and for the same reason: a maximized or tiled window fills a box the
// desktop chose and shares its edges with the screen or a neighbour, so a
// window that stopped short of them would leave gaps nobody can click.
//
// The fitted caption goes with it. The two are halves of one frame — a band
// narrower than its window is a gap unless the window really stops there —
// so they are dropped together or not at all.
func TestASilhouetteIsDroppedWhereTheFrameIs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cases := []struct {
		name string
		st   DecorationState
		want bool
	}{
		{"restored", DecorationState{Active: true}, true},
		{"backdrop", DecorationState{}, true},
		// An uncomposited screen keeps its silhouette: X11's bounding shape
		// still cuts the window there, and it is the one case where cutting
		// is the only thing that works at all.
		{"uncomposited", DecorationState{Active: true, Solid: true}, true},
		{"maximized", DecorationState{Active: true, Maximized: true}, false},
		{"tiled left", DecorationState{Active: true, Tiled: EdgeLeft}, false},
	}
	for _, name := range []string{"beos", "deck"} {
		p, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("no pack %q", name)
		}
		for _, c := range cases {
			f := probeFrame(600, 400, 240, 40)
			if got := WindowShapeOf(p.Look(), f, c.st) != nil; got != c.want {
				t.Errorf("%s %s: silhouette = %v, wanted %v", name, c.name, got, c.want)
			}
		}
	}
	// The fitted caption follows the same rule, for the look that asks.
	lk := themePack(t, "beos").Look()
	for _, c := range cases {
		if got := DecorationOf(lk, c.st).CaptionFits; got != c.want {
			t.Errorf("beos %s: fitted caption = %v, wanted %v", c.name, got, c.want)
		}
	}
}

// ---- a skin's rect union ------------------------------------------------------

// A skin's silhouette is stated in design pixels and resolved against the
// window, so the same outline comes out at every display scale rather than
// one outline stretched. That is the whole reason the manifest carries
// rectangles instead of a bitmap.
func TestASkinsShapeResolvesTheSameAtEveryScale(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sk, ok := LoadSkin("deck")
	if !ok {
		t.Fatal("no deck skin")
	}
	var want []paintengine2d.Rect
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		got := SkinWindowRects(sk, paintengine2d.XYWH(0, 0, 600*scale, 400*scale), scale)
		if len(got) < 2 {
			t.Fatalf("@%gx: %d rects, wanted the shoulder and the body", scale, len(got))
		}
		back := make([]paintengine2d.Rect, len(got))
		for i, r := range got {
			back[i] = paintengine2d.XYWH(r.Min.X/scale, r.Min.Y/scale, r.Dx()/scale, r.Dy()/scale)
		}
		if want == nil {
			want = back
			continue
		}
		if len(back) != len(want) {
			t.Fatalf("@%gx: %d rects, wanted %d", scale, len(back), len(want))
		}
		for i := range back {
			if !rectNear(back[i], want[i], 0.01) {
				t.Errorf("@%gx: rect %d resolves to %v, at 1× it is %v", scale, i, back[i], want[i])
			}
		}
	}
	// And it really is an outline with the desktop beside it: the shoulder
	// runs the window's full width and the body does not.
	if want[0].Dx() <= want[1].Dx() {
		t.Fatalf("deck's shoulder %v is no wider than its body %v", want[0], want[1])
	}
}

// near8 compares two channel bytes within the tolerance a gradient's dither
// needs.
func near8(a, b uint8) bool {
	if a > b {
		a, b = b, a
	}
	return b-a < 12
}

func rectNear(a, b paintengine2d.Rect, eps float32) bool {
	d := func(x, y float32) bool { return x-y < eps && y-x < eps }
	return d(a.Min.X, b.Min.X) && d(a.Min.Y, b.Min.Y) && d(a.Max.X, b.Max.X) && d(a.Max.Y, b.Max.Y)
}

// A union of overlapping rectangles is one silhouette, not a rectangle with a
// hole in it: the fill rule is nonzero, and even-odd here would cut the
// overlap away — which is exactly what joins a tab to the body under it.
func TestASilhouetteUnionsItsRectangles(t *testing.T) {
	s := SilhouetteOfRects(
		SilhouetteRect{Rect: paintengine2d.XYWH(0, 0, 60, 30)},
		SilhouetteRect{Rect: paintengine2d.XYWH(0, 20, 100, 60)},
	)
	cov := silCoverage(t, s, 100, 80)
	for _, p := range [][2]int{{10, 10}, {30, 25}, {90, 60}} {
		if !inside(cov, 100, 80, p[0], p[1]) {
			t.Errorf("(%d,%d) is outside a union that covers it", p[0], p[1])
		}
	}
	if inside(cov, 100, 80, 90, 10) {
		t.Error("(90,10) is inside: the union grew past both rectangles")
	}
}

// ---- BeOS's tab ---------------------------------------------------------------

// The silhouette is the tab that is painted, to the pixel.
//
// This is the invariant worth pinning rather than the arithmetic: the yellow
// the engine draws and the region the compositor is given come from one
// function, so a window can never look like it covers a pixel it does not
// take clicks on. The test finds the painted tab's right edge by looking for
// the yellow, and asks the silhouette about the pixels either side of it.
func TestTheBeOSSilhouetteIsItsPaintedTab(t *testing.T) {
	for _, scale := range []float32{1, 1.75} {
		lk := retroLook(t, "beos", scale)
		st := DecorationState{Active: true}
		w, h := int(600*scale), int(400*scale)
		capH := DecorationOf(lk, st).Caption
		f := probeFrame(float32(w), float32(h), 220*scale, capH)

		// Paint the frame and find the tab. Its rightmost column is the dark
		// rule down its edge, not the yellow, so the edge is the last column
		// of row 2 that is not the bare panel beside it.
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Transparent)
		DrawDecorationOf(lk, ctx, f, st)
		bare := func(x int) bool {
			r, g, b, _ := img.PremulAt(x, 2)
			br, bg, bb, _ := img.PremulAt(w-4, 2)
			return near8(r, br) && near8(g, bg) && near8(b, bb)
		}
		yellow := -1
		for x := w - 1; x >= 0; x-- {
			if r, g, b, _ := img.PremulAt(x, 2); r > 200 && g > 140 && b < 90 {
				yellow = x
				break
			}
		}
		edge := -1
		for x := w - 1; x > yellow; x-- {
			if !bare(x) {
				edge = x
				break
			}
		}
		if yellow < 0 || edge < 0 || edge > w/2 {
			t.Fatalf("@%gx: no fitted tab found (yellow %d, edge %d of %d)", scale, yellow, edge, w)
		}

		s := WindowShapeOf(lk, f, st)
		if s == nil {
			t.Fatalf("@%gx: no silhouette", scale)
		}
		cov := silCoverage(t, s, w, h)
		if !inside(cov, w, h, edge-1, 2) {
			t.Errorf("@%gx: the tab's own pixel (%d,2) is not the window", scale, edge-1)
		}
		// Past the tab, the top of the window is not the window at all.
		for _, x := range []int{edge + 2, w / 2, w - 2} {
			if inside(cov, w, h, x, 2) {
				t.Errorf("@%gx: (%d,2) is window, but the tab ends at %d", scale, x, edge)
			}
		}
		// The body below the tab runs the whole width, and it starts where
		// the tab ends — the tab's last row is the frame's top line.
		top := -1
		for y := 0; y < h; y++ {
			if inside(cov, w, h, w-2, y) {
				top = y
				break
			}
		}
		if top <= 0 || float32(top) >= capH {
			t.Fatalf("@%gx: the body starts at row %d, outside the caption band (%.0f)", scale, top, capH)
		}
		for _, p := range [][2]int{{2, h - 2}, {w - 2, h - 2}, {w / 2, top + 2}} {
			if !inside(cov, w, h, p[0], p[1]) {
				t.Errorf("@%gx: (%d,%d) is not the window, but the body is full width", scale, p[0], p[1])
			}
		}
	}
}

// ---- a control cut from its own art ---------------------------------------------

// shapeSkinPNG is a sheet holding two faces of the same size: a disc, whose
// corners are empty, and a square, which fills its cell. One is the point of
// the feature and the other is the case that must not pay for it.
func shapeSkinPNG(t *testing.T, scale float32) []byte {
	t.Helper()
	n := func(v float32) float32 { return v * scale }
	img := paintengine2d.NewImage(int(n(64)), int(n(32)))
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	ctx.DrawCircle(paintengine2d.Pt(n(16), n(16)), n(16), paintengine2d.Fill(paintengine2d.RGB(1, 0, 0)))
	ctx.DrawRect(paintengine2d.XYWH(n(32), 0, n(32), n(32)), paintengine2d.Fill(paintengine2d.RGB(0, 0, 1)))
	var buf bytes.Buffer
	if err := img.WritePNG(&buf); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return buf.Bytes()
}

// shapeSkinDoc binds the disc to the push button and the square to the tool
// button, and gives the window an outline of two rectangles.
const shapeSkinDoc = `{
  "skin": 1,
  "label": "Round", "year": 2026, "lineage": "test",
  "family": "dark", "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": {
    "disc":   { "sheet": "chrome", "at": [0, 0, 32, 32] },
    "square": { "sheet": "chrome", "at": [32, 0, 32, 32] }
  },
  "parts": {
    "button": { "states": { "normal": "disc" } },
    "tool":   { "states": { "normal": "square" } }
  },
  "window": {
    "caption": 30,
    "shape": [
      { "at": [0, 0, 0, 30], "stretchX": true },
      { "at": [10, 24, 10, 0], "stretchX": true, "stretchY": true }
    ]
  }
}`

// installShapeSkin writes the fixture skin under the current config dir and
// returns its look, so the engine reaches it exactly as it reaches a skin a
// user installed.
func installShapeSkin(t *testing.T, name, doc string, scale float32) *Classic {
	t.Helper()
	dir := filepath.Join(SkinsDir(), name)
	if err := os.MkdirAll(filepath.Join(dir, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	for file, art := range map[string][]byte{
		"sheet.png":  shapeSkinPNG(t, 1),
		"sheet2.png": shapeSkinPNG(t, 2),
	} {
		if err := os.WriteFile(filepath.Join(dir, "art", file), art, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, SkinFile), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateSkinCache()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("skin %q does not load as a pack", name)
	}
	return WithScale(p.Look(), scale).(*Classic)
}

// A skinned control is where its art is. The disc's ink takes the pointer and
// the corners of its box do not, at 1× and at a fractional scale, because the
// mask is painted at the size the control actually is rather than resampled
// from the sheet.
func TestASpriteShapesItsControl(t *testing.T) {
	for _, scale := range []float32{1, 1.75} {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		lk := installShapeSkin(t, "round", shapeSkinDoc, scale)
		side := int(48 * scale)
		box := paintengine2d.XYWH(0, 0, float32(side), float32(side))

		s := ControlShapeOf(lk, box, RoleButton)
		if s == nil {
			t.Fatalf("@%gx: a disc-faced button is still its whole box", scale)
		}
		cov := silCoverage(t, s, side, side)
		if !inside(cov, side, side, side/2, side/2) {
			t.Errorf("@%gx: the middle of the disc is not the button", scale)
		}
		for _, p := range [][2]int{{1, 1}, {side - 2, 1}, {1, side - 2}, {side - 2, side - 2}} {
			if inside(cov, side, side, p[0], p[1]) {
				t.Errorf("@%gx: corner (%d,%d) is the button, but the art is a disc", scale, p[0], p[1])
			}
		}
		// Art that fills its box says so by saying nothing: the control keeps
		// the box it has always had, and no mask is kept for it.
		if got := ControlShapeOf(lk, box, RoleTool); got != nil {
			t.Errorf("@%gx: a square face narrowed its own box", scale)
		}
		// A part the skin never bound is its base pack's, and the base pack
		// paints faces that fill their box.
		if got := ControlShapeOf(lk, box, RoleField); got != nil {
			t.Errorf("@%gx: an unbound part claimed a silhouette", scale)
		}
	}
}

// The mask is derived once per sprite, size and scale. It is the one
// expensive thing here — a control's box painted into a scratch — so a hit
// test that asks twice must not pay twice.
func TestAControlsMaskIsDerivedOnce(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := installShapeSkin(t, "round", shapeSkinDoc, 1)
	box := paintengine2d.XYWH(0, 0, 48, 48)
	first := ControlShapeOf(lk, box, RoleButton)
	if first == nil {
		t.Fatal("no silhouette")
	}
	for i := 0; i < 4; i++ {
		if got := ControlShapeOf(lk, box, RoleButton); got == nil || got.Mask != first.Mask {
			t.Fatalf("ask %d re-derived the mask", i)
		}
	}
	// A different size is a different mask, and both stay in the cache.
	other := ControlShapeOf(lk, paintengine2d.XYWH(0, 0, 60, 60), RoleButton)
	if other == nil || other.Mask == first.Mask {
		t.Fatal("a control of another size reused the first mask")
	}
	if again := ControlShapeOf(lk, box, RoleButton); again == nil || again.Mask != first.Mask {
		t.Fatal("the first mask was dropped by the second")
	}
}

// A skin may also hand the silhouette its artwork instead of rectangles,
// which is the fixed panel whose edge is in the picture and nowhere else.
func TestASkinsWindowShapeCanBeItsArt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	doc := `{
  "skin": 1, "label": "Round", "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": { "disc": { "sheet": "chrome", "at": [0, 0, 32, 32] } },
  "parts": { "window": { "states": { "normal": "disc" } } },
  "window": { "shape": { "art": "disc" } }
}`
	lk := installShapeSkin(t, "arty", doc, 1)
	sk := skinFor(lk)
	if sk == nil || sk.Window == nil || sk.Window.ShapeArt == nil {
		t.Fatal("the manifest's art silhouette did not load")
	}
	if len(sk.Window.Shape) != 0 {
		t.Fatal("art and rectangles are alternatives")
	}
	s := WindowShapeOf(lk, probeFrame(200, 200, 200, 30), DecorationState{Active: true})
	if s == nil || s.Mask == nil {
		t.Fatalf("no mask-backed silhouette: %+v", s)
	}
	cov := silCoverage(t, s, s.Mask.Width, s.Mask.Height)
	w, h := s.Mask.Width, s.Mask.Height
	if !inside(cov, w, h, w/2, h/2) {
		t.Error("the middle of the art is not the window")
	}
	if inside(cov, w, h, 0, 0) {
		t.Error("the empty corner of the art is the window")
	}
}

// A manifest that states both forms, or names a sprite that is not there, is
// refused by name like every other mistake in the format.
func TestAWindowShapeRefusesWhatItCannotMean(t *testing.T) {
	sheets := `"sheets": { "chrome": { "1x": "art/sheet.png" } }`
	cases := []struct{ name, doc, wantKey, wantMsg string }{
		{"art names no sprite", `{"skin": 1, ` + sheets + `,
			"window": {"shape": {"art": "ghost"}}}`, "window.shape.art", `no sprite "ghost"`},
		{"art is missing", `{"skin": 1, ` + sheets + `,
			"window": {"shape": {}}}`, "window.shape", `needs "art"`},
		{"unknown key in the object", `{"skin": 1, ` + sheets + `,
			"window": {"shape": {"arts": "x"}}}`, "window.shape", `unknown key "arts"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := LoadSkinFS(skinTestFS(t, c.doc), "probe")
			if err == nil {
				t.Fatal("loaded")
			}
			se, ok := err.(*SkinError)
			if !ok || se.Key != c.wantKey || !strings.Contains(se.Msg, c.wantMsg) {
				t.Fatalf("error %q, wanted key %q holding %q", err, c.wantKey, c.wantMsg)
			}
		})
	}
}
