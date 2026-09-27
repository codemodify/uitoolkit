//go:build theme_engine_all || theme_engine_neumorphism || (!theme_engine_adwaita && !theme_engine_adwaita48 && !theme_engine_aero && !theme_engine_amiga && !theme_engine_aqua && !theme_engine_beos && !theme_engine_bluecurve && !theme_engine_breeze && !theme_engine_breeze6 && !theme_engine_clearlooks && !theme_engine_flatlaf && !theme_engine_fluent && !theme_engine_fusion && !theme_engine_kde1 && !theme_engine_kde2 && !theme_engine_keramik && !theme_engine_luna && !theme_engine_macos && !theme_engine_macos_tahoe && !theme_engine_material && !theme_engine_material_expressive && !theme_engine_metal && !theme_engine_metro && !theme_engine_motif && !theme_engine_next && !theme_engine_nimbus && !theme_engine_openlook && !theme_engine_os2 && !theme_engine_oxygen && !theme_engine_plastik && !theme_engine_platinum && !theme_engine_skin && !theme_engine_system7 && !theme_engine_web && !theme_engine_win31 && !theme_engine_win95)

package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func neuLook(t *testing.T, name string) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("no %q pack", name)
	}
	lk := p.Look()
	if got := lk.Engine().ID(); got != "neumorphism" {
		t.Fatalf("%s is painted by %q", name, got)
	}
	return lk
}

// neuPaint paints one control into its own image and hands back the pixels.
func neuPaint(lk *Classic, w, h int, draw func(*paintengine2d.Context, paintengine2d.Rect)) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(lk.Palette().Background)
	draw(ctx, paintengine2d.XYWH(0, 0, float32(w), float32(h)))
	return img
}

// The whole look is one rule: a control is the background pushed out of
// the surface or pressed into it, and only the shadows say which. So a
// raised control and a pressed one must differ — and differ *the other
// way up*, light where the other is dark.
func TestNeumorphismPressedIsTheRaisedOneInverted(t *testing.T) {
	lk := neuLook(t, "neumorphism")
	const w, h = 90, 40
	raised := neuPaint(lk, w, h, func(c *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawButton(c, b, StateNone, "")
	})
	pressed := neuPaint(lk, w, h, func(c *paintengine2d.Context, b paintengine2d.Rect) {
		lk.DrawButton(c, b, StatePressed, "")
	})

	// Where the shadows are differs between the two, and that is the
	// point of the look rather than an awkwardness of the test: a
	// raised control throws its shadows *outside* the face, into the
	// margin the face was inset to leave; a pressed one casts them
	// *inside*, against its own edge. So each is measured where its
	// own shadows fall.
	corner := func(img *paintengine2d.Image, x0, y0, x1, y1 int) float64 {
		var sum float64
		var n int
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				r, g, b, _ := img.PremulAt(x, y)
				sum += RelLuminance(paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255))
				n++
			}
		}
		return sum / float64(n)
	}
	const m = 5 // the margin the face is inset by at 1x

	// Raised: a box straddling the face's corner, since the shadow is
	// thrown clear of the edge and falls off across the margin rather
	// than hugging the very outside of it.
	rTL := corner(raised, 0, 0, 2*m, 2*m)
	rBR := corner(raised, w-2*m, h-2*m, w, h)
	if rTL <= rBR {
		t.Errorf("raised: the margin is %.4f at the top left and %.4f at the bottom right; the light comes from the top left", rTL, rBR)
	}
	if rTL-rBR < 0.005 {
		t.Errorf("raised: the shadows are too faint to read (%.4f vs %.4f)", rTL, rBR)
	}

	// Pressed: just inside the face, against its edge.
	pTL := corner(pressed, m+1, m+1, m+9, m+9)
	pBR := corner(pressed, w-m-9, h-m-9, w-m-1, h-m-1)
	if pTL >= pBR {
		t.Errorf("pressed: inside the face it is %.4f at the top left and %.4f at the bottom right; pressed inverts the light", pTL, pBR)
	}
	if pBR-pTL < 0.005 {
		t.Errorf("pressed: the shadows are too faint to read (%.4f vs %.4f)", pTL, pBR)
	}
}

// Everything is one colour: the face of a control is the window's own
// background, not a lighter or darker plate laid on it. That is what
// makes it "soft UI" rather than a flat theme with rounded corners.
func TestNeumorphismFaceIsTheBackground(t *testing.T) {
	for _, name := range []string{"neumorphism", "neumorphism-night"} {
		lk := neuLook(t, name)
		bg := lk.Palette().Background
		img := neuPaint(lk, 90, 40, func(c *paintengine2d.Context, b paintengine2d.Rect) {
			lk.DrawButton(c, b, StateNone, "")
		})
		r, g, b, _ := img.PremulAt(45, 20) // dead centre, the face
		got := paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
		near := func(a, b float32) bool { d := a - b; return d < 0.02 && d > -0.02 }
		if !near(got.R, bg.R) || !near(got.G, bg.G) || !near(got.B, bg.B) {
			t.Errorf("%s: the face is %v, the surface is %v", name, got, bg)
		}
	}
}

// A control paints inside the rect it was given, shadows included. The
// look wants its shadows outside the face, so the face is inset to make
// room — a soft control that bled over its box could not be laid out
// beside anything.
func TestNeumorphismPaintsInsideItsRect(t *testing.T) {
	lk := neuLook(t, "neumorphism")
	cases := []struct {
		name string
		w, h int
		draw func(*paintengine2d.Context, paintengine2d.Rect)
	}{
		{"button", 90, 40, func(c *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawButton(c, b, StateNone, "") }},
		{"button pressed", 90, 40, func(c *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawButton(c, b, StatePressed, "") }},
		{"button focused", 90, 40, func(c *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawButton(c, b, StateFocused, "") }},
		{"check", 20, 20, func(c *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawCheckbox(c, b, StateNone, true, "") }},
		{"radio", 20, 20, func(c *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawRadio(c, b, StateNone, true, "") }},
		// A control smaller than the shadows want: it must shrink its
		// inset, not paint outside.
		{"tiny", 10, 10, func(c *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawButton(c, b, StateNone, "") }},
	}
	for _, c := range cases {
		pad := 6
		img := paintengine2d.NewImage(c.w+pad*2, c.h+pad*2)
		ctx := paintengine2d.NewContext(img)
		c.draw(ctx, paintengine2d.XYWH(float32(pad), float32(pad), float32(c.w), float32(c.h)))
		if x, y, ok := outsideRect(img, paintengine2d.XYWH(float32(pad), float32(pad), float32(c.w), float32(c.h))); ok {
			t.Errorf("%s: painted outside its rect at (%d,%d)", c.name, x, y)
		}
	}
}

// Both packs are registered, named and in the same era, so Settings
// groups them together.
func TestNeumorphismPacksAreRegistered(t *testing.T) {
	for _, name := range []string{"neumorphism", "neumorphism-night"} {
		p, ok := LoadTheme(name)
		if !ok {
			t.Fatalf("no %q pack", name)
		}
		if p.Era != EraNeumorph {
			t.Errorf("%s is in era %q", name, p.Era)
		}
		if p.Tokens.Engine != "neumorphism" {
			t.Errorf("%s names engine %q", name, p.Tokens.Engine)
		}
		if p.Summary == "" {
			t.Errorf("%s has no summary", name)
		}
	}
}
