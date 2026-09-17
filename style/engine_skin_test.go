package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// skinPackNames are the skins the toolkit ships.
var skinPackNames = []string{"cassette", "nocturne"}

// The registration convention every engine's packs follow: the pack resolves,
// names this engine, and its look paints with it.
func TestSkinPacksRegisteredWithTheSkinEngine(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for _, n := range skinPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, ok := LoadTheme(n)
		if !ok {
			t.Fatalf("pack %q does not load", n)
		}
		if p.Tokens.Engine != skinEngineID {
			t.Fatalf("%s engine %q", n, p.Tokens.Engine)
		}
		if lk := p.Look(); lk.Engine().ID() != skinEngineID {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
		if p.Year == 0 || p.Lineage == "" || p.Summary == "" {
			t.Errorf("%s: a pack needs a year, a lineage and a summary to sort and list", n)
		}
	}
}

// The paint contract every engine is held to: every control, in every
// interesting state, at 1×, 1.75× and 2×, painted inside its own rect without
// panicking and without leaking a saved context state.
//
// A skin is where this matters most. Its faces are pictures blitted into a
// box the widget chose, and the whole nine-slice arithmetic — whole-pixel
// corners, a stretched middle, corners that shrink rather than overlap in a
// box too small for them — exists so that a blit never lands outside it.
func TestSkinPaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range skinPackNames {
		p, ok := LoadTheme(n)
		if !ok {
			t.Fatalf("pack %q does not load", n)
		}
		for _, scale := range []float32{1, 1.75, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			name := fmt.Sprintf("%s@%gx", n, scale)
			aquaExercise(t, name, lk)
			kdeExtras(t, name, lk)
		}
	}
}

// A skinned frame is still a frame: its close button is inside its caption,
// and its caption is tall enough to hold one.
func TestSkinFrameCloseRectIsInsideTheCaption(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range skinPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 1.75, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			b := paintengine2d.XYWH(10, 10, 320, 200)
			cr := lk.WindowCloseRect(b)
			if cr.Empty() {
				continue // an in-app frame without a close box is legitimate
			}
			top := lk.WindowFrameInsets().Top
			if cr.Max.X > b.Max.X || cr.Max.Y > b.Min.Y+top {
				t.Errorf("%s@%gx: close %v outside the title bar of %v", n, scale, cr, b)
			}
		}
	}
}

// Every label a skin states has to read on the art behind it. The shared
// per-pack tests check the palette; these are the skin's own text roles,
// which the palette does not see.
func TestSkinLabelsReadOnTheirArt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range skinPackNames {
		sk, ok := LoadSkin(n)
		if !ok {
			t.Fatalf("%s does not load", n)
		}
		p, _ := LoadTheme(n)
		lk := p.Look()
		for _, part := range []string{"button", "field", "combo", "tab", "tool", "row", "menu"} {
			if !sk.has(part) {
				continue
			}
			for _, st := range []ControlState{StateNone, StateHovered, StateChecked, StatePrimary} {
				fg := sk.textColor(lk, sk.part(part), st)
				if colorUnset(fg) {
					continue // the base look's ink, already checked per pack
				}
				bg := skinFaceInk(t, lk, sk, part, st)
				if bg.A < 0.5 {
					continue // the art is transparent here: the surface shows
				}
				if r := ContrastRatio(fg, bg); r < 3 {
					t.Errorf("%s: %s label %v on its own art %v is %.1f:1, wanted 3:1",
						n, part, fg, bg, r)
				}
			}
		}
	}
}

// skinFaceInk paints a part's art and reads the pixel under where its label
// would sit.
func skinFaceInk(t *testing.T, lk *Classic, sk *Skin, part string, st ControlState) paintengine2d.Color {
	t.Helper()
	img := paintengine2d.NewImage(120, 40)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Color{})
	sk.draw(lk, ctx, paintengine2d.XYWH(0, 0, 120, 40), part, st)
	c := img.NRGBAAt(60, 20)
	return paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255)
}

// Skins must not opt into an engine hook the platform never had. The accent
// test asserts the exact set of engines that take the desktop's accent
// colour; a skin's colours are the skin author's, not the desktop's.
func TestSkinDoesNotTakeTheDesktopAccent(t *testing.T) {
	e, ok := EngineByID(skinEngineID)
	if !ok {
		t.Fatal("the skin engine is not registered")
	}
	if _, takes := e.(AccentEngine); takes {
		t.Fatal("a skin's colours are its author's; it must not take the desktop accent")
	}
}
