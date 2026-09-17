package style

import "testing"

// The one switch: which looks ask for real glass, when they get it, and
// what happens on a desktop that cannot blur.

func lookNamed(t *testing.T, pack string) LookAndFeel {
	t.Helper()
	p, ok := LoadTheme(pack)
	if !ok {
		t.Skipf("no %s pack", pack)
	}
	return p.Look()
}

func TestOnlyTheGlassLooksAskForIt(t *testing.T) {
	// The looks that have faked glass since they were written ask for the
	// real thing; the ones that never had any do not, and nothing about
	// them changes.
	for _, pack := range []string{"fluent", "bigsur", "tahoe"} {
		lk := lookNamed(t, pack)
		if !WantsGlass(lk) {
			t.Errorf("%s does not ask for glass", pack)
		}
	}
	for _, pack := range []string{"win95", "motif", "redmond"} {
		p, ok := LoadTheme(pack)
		if !ok {
			continue
		}
		if WantsGlass(p.Look()) {
			t.Errorf("%s asks for glass and never faked any", pack)
		}
	}
}

func TestGlassNeedsBothTheLookAndTheDesktop(t *testing.T) {
	lk := lookNamed(t, "bigsur")
	SetGlassAvailable(false)
	t.Cleanup(func() { SetGlassAvailable(false) })
	if GlassBehind(lk) {
		t.Fatal("a look got glass on a desktop that cannot blur")
	}
	SetGlassAvailable(true)
	if !GlassBehind(lk) {
		t.Fatal("a look that asks did not get glass on a desktop that can blur")
	}
	// A look that never asked never gets it, however capable the desktop.
	if p, ok := LoadTheme("win95"); ok && GlassBehind(p.Look()) {
		t.Fatal("a look that never asked got glass")
	}
	if GlassBehind(nil) {
		t.Fatal("no look at all got glass")
	}
}

func TestAnUncompositedScreenHasNoGlass(t *testing.T) {
	// Nothing composites the window: there is no desktop behind it to
	// blur, and alpha counts for nothing there — the same clause that
	// squares the corners and drops the shadow.
	lk := lookNamed(t, "bigsur")
	if !DecorationOf(lk, DecorationState{Active: true}).Glass {
		t.Fatal("no glass to drop")
	}
	if DecorationOf(lk, DecorationState{Active: true, Solid: true}).Glass {
		t.Fatal("an uncomposited screen kept its glass")
	}
	// A maximized window keeps it: Mica and vibrancy do not stop at the
	// screen's edge, unlike a corner radius.
	if !DecorationOf(lk, DecorationState{Active: true, Maximized: true}).Glass {
		t.Fatal("a maximized window lost its glass")
	}
}

func TestGlassTintIsAlwaysTranslucent(t *testing.T) {
	// An opaque tint over a blurred desktop shows none of the blur.
	lk := lookNamed(t, "bigsur")
	c := GlassTint(lk, 0.78)
	if c.A >= 1 {
		t.Fatalf("glass tint %v is opaque", c)
	}
	if c.A <= 0 {
		t.Fatalf("glass tint %v is invisible", c)
	}
	// And it is the look's own colour, not a generic grey.
	bg := lk.Palette().Background
	if c.R != bg.R || c.G != bg.G || c.B != bg.B {
		t.Fatalf("glass tint %v is not the look's background %v", c, bg)
	}
}
