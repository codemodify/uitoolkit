//go:build theme_engine_all || theme_engine_hangar

package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func hangarLook(t *testing.T) *Classic {
	t.Helper()
	p, ok := LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	lk := p.Look()
	if lk == nil || lk.Engine().ID() != "hangar" {
		t.Fatalf("the pack paints with %v", lk.Engine().ID())
	}
	return lk
}

// shot draws into a small image over the pack's own ground.
func hangarShot(t *testing.T, lk *Classic, w, h int, draw func(*paintengine2d.Context, paintengine2d.Rect)) *paintengine2d.Image {
	t.Helper()
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(0, 0, float32(w), float32(h))
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Background))
	draw(ctx, b)
	img.Touch()
	return img
}

// near reports whether two colours are within tol on every channel.
func hangarNear(a, b paintengine2d.Color, tol float32) bool {
	d := func(x, y float32) bool { return x-y < tol && y-x < tol }
	return d(a.R, b.R) && d(a.G, b.G) && d(a.B, b.B)
}

// countNear is how many pixels of img are within tol of col.
func hangarCount(img *paintengine2d.Image, col paintengine2d.Color, tol float32) int {
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			c := img.NRGBAAt(x, y)
			if hangarNear(paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1), col, tol) {
				n++
			}
		}
	}
	return n
}

// **The ember is spent on what is on, never on a hover.** It is the one
// rule the whole look hangs from: a console where the hot colour means
// "the pointer is here" is one where it no longer means "this is on".
func TestHangarKeepsItsAccentForWhatIsOn(t *testing.T) {
	lk := hangarLook(t)
	accent := lk.Palette().Accent
	ember := func(st ControlState, d ButtonDraw) int {
		img := hangarShot(t, lk, 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawButton(ctx, b, st, d)
		})
		return hangarCount(img, accent, 0.06)
	}
	plain := ember(StateNone, ButtonDraw{Label: "Run"})
	hover := ember(StateHovered, ButtonDraw{Label: "Run"})
	press := ember(StatePressed, ButtonDraw{Label: "Run"})
	primary := ember(StatePrimary, ButtonDraw{Label: "Run"})

	if plain > 20 || hover > 20 || press > 20 {
		t.Errorf("an ordinary button wears the accent: plain %d, hover %d, pressed %d", plain, hover, press)
	}
	if primary < 1000 {
		t.Errorf("the default button is only %d pixels of ember; it is the accent's own button", primary)
	}
	// A hover is still visible — it is a wash of white, not nothing.
	lit := hangarShot(t, lk, 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawButton(ctx, b, StateHovered, ButtonDraw{Label: "Run"})
	})
	rest := hangarShot(t, lk, 120, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawButton(ctx, b, StateNone, ButtonDraw{Label: "Run"})
	})
	if diffPixels(lit, rest) < 200 {
		t.Error("a hover changed almost nothing: the wash is missing")
	}
}

// Chrome labels are painted in capitals and the application's words are
// untouched — the same rule a skin's "case": "upper" follows.
func TestHangarPaintsChromeLabelsInCapitals(t *testing.T) {
	lk := hangarLook(t)
	for _, role := range []Role{RoleButton, RoleTool, RoleTab, RoleCombo} {
		if got := lk.ControlLabel(role, "Run test"); got != "RUN TEST" {
			t.Errorf("role %v paints %q", role, got)
		}
	}
	// What a control shows of the person's own content is theirs.
	for _, role := range []Role{RoleField, RoleRow, RoleMenu} {
		if got := lk.ControlLabel(role, "Run test"); got != "Run test" {
			t.Errorf("role %v paints %q, want the words as written", role, got)
		}
	}
	// And a pack can turn it off.
	p, _ := LoadTheme("hangar")
	p.Tokens.Params = map[string]float32{"caps": 0}
	plain := p.Look()
	if got := plain.ControlLabel(RoleButton, "Run test"); got != "Run test" {
		t.Errorf("with caps off a button paints %q", got)
	}
}

// Everything is square and one pixel thick: no radius, no bevel.
func TestHangarIsSquareAndHairline(t *testing.T) {
	lk := hangarLook(t)
	m := lk.Metrics()
	if m.Radius != 0 || m.RadiusSmall != 0 {
		t.Errorf("radii %g / %g, want square", m.Radius, m.RadiusSmall)
	}
	p, _ := LoadTheme("hangar")
	if p.Tokens.Bevel != BevelNone {
		t.Errorf("bevel %q", p.Tokens.Bevel)
	}
	if !p.Tokens.Metrics.Square && !lk.Engine().DefaultMetrics().Square {
		t.Error("the look does not call itself square, so the Corners pref can round it")
	}
}

// A meter is segmented: blocks with gaps, not one bar. The fill is the
// accent and the gaps are not, so a half-full meter has both.
func TestHangarMetersAreSegmented(t *testing.T) {
	lk := hangarLook(t)
	img := hangarShot(t, lk, 120, 10, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawProgressBar(ctx, b, StateNone, 1, false, 0)
	})
	// Along the middle row the ember must start and stop more than once.
	runs, was := 0, false
	y := img.Height / 2
	for x := 0; x < img.Width; x++ {
		c := img.NRGBAAt(x, y)
		on := hangarNear(paintengine2d.RGBA(float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, 1), lk.Palette().Accent, 0.08)
		if on && !was {
			runs++
		}
		was = on
	}
	if runs < 3 {
		t.Errorf("a full meter is %d blocks of ember, want it segmented", runs)
	}
}

// The window ground is a gradient: the top-left corner is the gunmetal
// and the bottom-right is warmer, and the whole of it depends on the
// window's box alone, so a partial repaint paints the same pixels.
func TestHangarGroundIsAGradientOfTheWindow(t *testing.T) {
	lk := hangarLook(t)
	const w, h = 160, 100
	whole := hangarShot(t, lk, w, h, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawWindowBackground(ctx, b)
	})
	tl := whole.NRGBAAt(2, 2)
	br := whole.NRGBAAt(w-3, h-3)
	if !(br.R > tl.R+6 && br.B <= tl.B+2) {
		t.Errorf("the ground runs from %v to %v; want it warming towards the far corner", tl, br)
	}
	// The same call under a clip paints the same pixels: that is what
	// "depends on the window's box alone" means in practice.
	clipped := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(clipped)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(lk.Palette().Background))
	ctx.Save()
	ctx.ClipRect(paintengine2d.XYWH(40, 20, 50, 40))
	lk.DrawWindowBackground(ctx, paintengine2d.XYWH(0, 0, w, h))
	ctx.Restore()
	clipped.Touch()
	for y := 21; y < 59; y++ {
		for x := 41; x < 89; x++ {
			if whole.NRGBAAt(x, y) != clipped.NRGBAAt(x, y) {
				t.Fatalf("a clipped repaint differs at %d,%d: %v against %v",
					x, y, clipped.NRGBAAt(x, y), whole.NRGBAAt(x, y))
			}
		}
	}
}

// The close button is the ember block — the one button on the panel you
// are not meant to press by accident, and the only one that is marked.
func TestHangarMarksItsCloseButton(t *testing.T) {
	lk := hangarLook(t)
	st := DecorationState{Active: true}
	ember := func(k CaptionButton) int {
		img := hangarShot(t, lk, 30, 30, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
			DrawCaptionButtonOf(lk, ctx, b, k, StateNone, st)
		})
		return hangarCount(img, lk.Palette().Accent, 0.06)
	}
	if n := ember(CaptionClose); n < 400 {
		t.Errorf("the close button is %d pixels of ember, want the block", n)
	}
	if n := ember(CaptionMinimize); n > 20 {
		t.Errorf("the minimize button wears %d pixels of ember", n)
	}
}

// The pack says what it is, the way every pack in here has to.
func TestHangarPackDescribesItself(t *testing.T) {
	p, ok := LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	if p.Label == "" || p.Lineage == "" || p.Era == "" || p.Year == 0 {
		t.Errorf("pack %+v", p)
	}
	if s := strings.TrimSpace(p.Summary); s == "" || strings.Contains(s, "\n") {
		t.Errorf("summary %q must be one line", p.Summary)
	}
	if p.Palette != ThemeDark {
		t.Errorf("family %v, want dark", p.Palette)
	}
	if len(p.Tokens.Fonts.UI) == 0 || len(p.Tokens.Fonts.Mono) == 0 {
		t.Error("the pack names no typefaces, so the genre's face is left to the machine")
	}
}
