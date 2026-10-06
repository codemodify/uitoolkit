package style

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/codemodify/paintengine2d"
)

// The skin contract lists "bar" as the face for menu, tool and status bars.
// A pack could bind a perfectly valid one and the stock bars painted the
// look's grey anyway: all four painters called paintBezel, the raw token
// drawing, instead of paintFace, which goes through the engine. The skin
// engine hands the *layout* of a bar to the base painter precisely because
// the face comes back through Face(), as it does for a text field — so the
// bound art was never asked for, and an application wrapped
// LookAndFeel.DrawMenuBar to paint its own.

// barSkinDoc is a pack whose only job is a solid magenta bar.
const barSkinDoc = `{
  "skin": 1, "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/bar.png", "2x": "art/bar2.png" } },
  "sprites": { "barface": { "sheet": "chrome", "at": [0, 0, 8, 8] } },
  "parts": { "bar": { "states": { "normal": "barface" } } }
}`

func barSkinPNG(t *testing.T, scale float32) []byte {
	t.Helper()
	n := int(8 * scale)
	img := paintengine2d.NewImage(n, n)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(n), float32(n)),
		paintengine2d.Fill(paintengine2d.RGB(1, 0, 1)))
	img.Touch()
	var buf bytes.Buffer
	if err := img.WritePNG(&buf); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return buf.Bytes()
}

func barSkinLook(t *testing.T, id string) LookAndFeel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	InvalidateSkinCache()
	fsys := fstest.MapFS{
		SkinFile:       {Data: []byte(barSkinDoc)},
		"art/bar.png":  {Data: barSkinPNG(t, 1)},
		"art/bar2.png": {Data: barSkinPNG(t, 2)},
	}
	if err := RegisterSkinFS(fsys, id); err != nil {
		t.Fatalf("register: %v", err)
	}
	p, ok := LoadTheme(id)
	if !ok {
		t.Fatalf("%s does not load", id)
	}
	lk := p.Look()
	if sk := skinFor(lk); sk == nil || !sk.has("bar") {
		t.Fatal("the pack does not bind a bar part")
	}
	return lk
}

// paintedAt draws one bar and reads a pixel well inside it.
func paintedAt(t *testing.T, draw func(*paintengine2d.Context, paintengine2d.Rect)) paintengine2d.Color {
	t.Helper()
	const W, H = 120, 28
	img := paintengine2d.NewImage(W, H)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.RGB(0, 0, 0))
	draw(ctx, paintengine2d.XYWH(0, 0, W, H))
	img.Touch()
	r, g, b, a := img.PremulAt(W/2, H/2)
	return paintengine2d.Color{R: float32(r) / 255, G: float32(g) / 255, B: float32(b) / 255, A: float32(a) / 255}
}

func TestTheStockBarsPaintTheSkinsBoundBarFace(t *testing.T) {
	lk := barSkinLook(t, "barpack")
	magenta := paintengine2d.RGB(1, 0, 1)
	for _, tc := range []struct {
		what string
		draw func(*paintengine2d.Context, paintengine2d.Rect)
	}{
		{"the menu bar", func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMenuBar(ctx, b) }},
		{"the tool bar", func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawToolBar(ctx, b) }},
		{"the tab bar", func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTabBar(ctx, b) }},
		{"the status bar", func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawStatusBar(ctx, b, nil) }},
	} {
		got := paintedAt(t, tc.draw)
		if !nearColor(got, magenta) {
			t.Errorf("%s painted %v, want the skin's magenta %v", tc.what, got, magenta)
		}
	}
}

// A look with no skin paints exactly what it did: the bar's face goes
// through the engine now, and the engine's answer for roleBar is the same
// SurfaceAlt and Divider the bezel calls passed by hand.
func TestABarWithNoSkinIsUnchanged(t *testing.T) {
	lk := DarkLook()
	got := paintedAt(t, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawMenuBar(ctx, b) })
	want := lk.Palette().SurfaceAlt
	if !nearColor(got, want) {
		t.Errorf("an unskinned menu bar painted %v, want the look's SurfaceAlt %v", got, want)
	}
}
