package style

import (
	"bytes"
	"testing"
	"testing/fstest"

	"github.com/codemodify/paintengine2d"
)

// A resized panel's artwork must land where the layout put its slots.
//
// A pixelated sheet is drawn at a *whole* multiple — that is a control-face
// policy, and the right one: a button, a field or a check drawn at 1× in the
// middle of a 1.75 box reads as pixel art rather than as a blurred mistake.
// Applied to a panel's own background it is wrong, because the controls
// standing on that background were placed by the layout at the true 1.75: a
// 38-design-pixel footer edge came out 38 device pixels while the slot the
// footer sits in came out 66.5, and the artwork and the controls parted
// company by 28 pixels.

// footerSkinDoc is a panel sprite whose bottom band is its own colour, so the
// painted band can be measured.
const footerSkinDoc = `{
  "skin": 1, "base": "breeze-night",
  "sheets": { "chrome": { "1x": "art/p.png", "2x": "art/p2.png", "pixelated": true } },
  "sprites": {
    "panel": { "sheet": "chrome", "at": [0, 0, 32, 32], "slice": [0, 0, 8, 0] }
  },
  "parts": { "panel": { "states": { "normal": "panel" } } }
}`

// footerPNG is blue with a red band across its bottom eight design rows.
func footerPNG(t *testing.T, scale float32) []byte {
	t.Helper()
	n := func(v float32) float32 { return v * scale }
	img := paintengine2d.NewImage(int(n(32)), int(n(32)))
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, n(32), n(32)), paintengine2d.Fill(paintengine2d.RGB(0, 0, 1)))
	ctx.DrawRect(paintengine2d.XYWH(0, n(24), n(32), n(8)), paintengine2d.Fill(paintengine2d.RGB(1, 0, 0)))
	img.Touch()
	var buf bytes.Buffer
	if err := img.WritePNG(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func footerLook(t *testing.T, id string, scale float32) LookAndFeel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	InvalidateSkinCache()
	fsys := fstest.MapFS{
		SkinFile:     {Data: []byte(footerSkinDoc)},
		"art/p.png":  {Data: footerPNG(t, 1)},
		"art/p2.png": {Data: footerPNG(t, 2)},
	}
	if err := RegisterSkinFS(fsys, id); err != nil {
		t.Fatalf("register: %v", err)
	}
	p, ok := LoadTheme(id)
	if !ok {
		t.Fatalf("%s does not load", id)
	}
	return WithScale(p.Look(), scale)
}

// redBandHeight is how many device rows of red there are, counting up from
// the row at from.
func redBandHeight(img *paintengine2d.Image, x, from int) int {
	n := 0
	for y := from; y >= 0; y-- {
		r, g, b, _ := img.PremulAt(x, y)
		if r > 200 && g < 60 && b < 60 {
			n++
			continue
		}
		break
	}
	return n
}

func TestAResizedPanelsSliceEdgeFollowsTheDisplayScale(t *testing.T) {
	for _, scale := range []float32{1.25, 1.5, 1.75} {
		lk := footerLook(t, "footerpack", scale)
		const W, H = 160, 220
		img := paintengine2d.NewImage(W, H)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.RGB(0, 0, 0))
		// A box that is not the sprite's own size: the layout resized it.
		if !DrawSkinSprite(lk, ctx, paintengine2d.XYWH(0, 0, W, H), "panel", paintengine2d.Color{}) {
			t.Fatalf("%.2fx: the sprite did not paint", scale)
		}
		img.Touch()

		got := redBandHeight(img, W/2, H-1)
		want := int(8*scale + 0.5) // the design band at the display scale
		if got < want-1 || got > want+1 {
			t.Errorf("%.2fx: the painted footer band is %d device rows, the layout places its slot at %d",
				scale, got, want)
		}
	}
}

// At whole scales nothing changes, which is where the two policies agree.
func TestAResizedPanelAtWholeScalesIsUnchanged(t *testing.T) {
	for _, scale := range []float32{1, 2} {
		lk := footerLook(t, "footerwhole", scale)
		const W, H = 160, 220
		img := paintengine2d.NewImage(W, H)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.RGB(0, 0, 0))
		DrawSkinSprite(lk, ctx, paintengine2d.XYWH(0, 0, W, H), "panel", paintengine2d.Color{})
		img.Touch()
		got, want := redBandHeight(img, W/2, H-1), int(8*scale+0.5)
		if got < want-1 || got > want+1 {
			t.Errorf("%.0fx: band %d rows, want %d", scale, got, want)
		}
	}
}

// A sprite drawn at its own size still takes the pixel-grid path, which is
// what keeps a panel's keys and digits meeting its face.
func TestAPanelSpriteAtItsOwnSizeIsStillOnTheGrid(t *testing.T) {
	lk := footerLook(t, "footerown", 1.75)
	sc := float32(1.75)
	side := float32(int(32*sc + 0.5))
	img := paintengine2d.NewImage(int(side)+8, int(side)+8)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.RGB(0, 0, 0))
	if !DrawSkinSprite(lk, ctx, paintengine2d.XYWH(0, 0, side, side), "panel", paintengine2d.Color{}) {
		t.Fatal("the sprite did not paint at its own size")
	}
	img.Touch()
	// The whole sprite is magnified, so the band is its own eight rows of
	// thirty-two, at the display scale.
	// From the sprite's own bottom edge, not the image's: the drawing is
	// only `side` tall and the rest of the image is the cleared ground.
	got, want := redBandHeight(img, int(side)/2, int(side)-1), int(8*sc+0.5)
	if got < want-2 || got > want+2 {
		t.Errorf("at its own size the band is %d rows, want about %d", got, want)
	}
}
