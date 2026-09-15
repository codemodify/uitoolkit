package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var md3xPackNames = []string{"material3x", "material3x-night"}

func TestMaterial3xPacksRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	for n, w := range map[string]struct {
		label string
		fam   ThemeName
	}{"material3x": {"Material 3 Expressive", ThemeLight}, "material3x-night": {"Material 3 Expressive Dark", ThemeDark}} {
		p, ok := LoadTheme(n)
		if !ok {
			t.Fatalf("pack %q not registered", n)
		}
		if p.Tokens.Engine != "material" || p.Lineage != "Google" || p.Label != w.label || p.Year != 2025 || p.Tokens.Family != w.fam {
			t.Fatalf("%s: engine %q lineage %q label %q year %d family %q", n, p.Tokens.Engine, p.Lineage, p.Label, p.Year, p.Tokens.Family)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if _, ok := p.Look().Engine().(material3xEngine); !ok {
			t.Fatalf("%s paints with %T, want the Expressive era", n, p.Look().Engine())
		}
	}
	if pos["material3"] > pos["material3x"] {
		t.Fatal("Material You (2021) sorts after Expressive (2025)")
	}
	for _, n := range materialPackNames {
		if _, ok := materialLook(t, n, 1).Engine().(materialEngine); !ok {
			t.Fatalf("%s left its generation", n)
		}
	}
}

// The baseline scheme's roles as the 2023–24 spec gives them: tone-based
// containers, 10% focus and pressed layers, tone-30 light on-containers.
func TestMaterial3xRoles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for n, want := range map[string]struct{ surface, menu, dialog, drawer, onPC string }{
		"material3x":       {"#fef7ff", "#f3edf7", "#ece6f0", "#f7f2fa", "#4f378b"},
		"material3x-night": {"#141218", "#211f26", "#2b2930", "#1d1b20", "#eaddff"},
	} {
		c := mdColors(materialLook(t, n, 1))
		for _, v := range []struct {
			name string
			got  paintengine2d.Color
			want string
		}{{"surface", c.surface, want.surface}, {"menu (surface-container)", c.menu, want.menu}, {"dialog (surface-container-high)", c.dialog, want.dialog},
			{"drawer (surface-container-low)", c.drawer(), want.drawer}, {"on-primary-container", c.onPrimaryC, want.onPC}} {
			if colorHexPadded(v.got) != v.want {
				t.Errorf("%s: %s %s, want %s", n, v.name, colorHexPadded(v.got), v.want)
			}
		}
		if c.aFocus != 0.10 || c.aPress != 0.10 || c.aHover != 0.08 {
			t.Errorf("%s: state layers %v/%v/%v, want 8%%/10%%/10%%", n, c.aHover, c.aFocus, c.aPress)
		}
	}
	// Material You keeps its 2021 tinted surfaces and 12% layers.
	if c := mdColors(materialLook(t, "material3", 1)); c.aFocus != 0.12 {
		t.Errorf("material3 changed: focus layer %v", c.aFocus)
	}
}

func TestMaterial3xPaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, md3xPackNames)
	winPaintsInsideRect(t, md3xPackNames)
	for _, n := range md3xPackNames {
		for _, sc := range []float32{1, 1.75, 2} {
			lk := materialLook(t, n, sc)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, sc), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, sc), lk)
		}
	}
}

func TestMaterial3xCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range md3xPackNames {
		winCheckClose(t, n)
	}
}

// md3xPaint paints f into a w×h image on the surface.
func md3xPaint(lk *Classic, w, h int, f func(ctx *paintengine2d.Context)) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(w), float32(h)), paintengine2d.Fill(mdColors(lk).surface))
	f(ctx)
	return img
}

// Buttons are round at rest and square off to 8dp corners while pressed;
// keyboard focus is the secondary ring round the face.
func TestMaterial3xButtonShapesAndFocus(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := materialLook(t, "material3x", 1)
	c := mdColors(lk)
	b := paintengine2d.XYWH(10, 10, 140, 50) // a 40px face 5px in
	paint := func(st ControlState) *paintengine2d.Image {
		return md3xPaint(lk, 160, 70, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, st|StatePrimary, "") })
	}
	rest, held := paint(StateNone), paint(StatePressed|StateHovered)
	// (18,17) lies outside the round end and inside an 8dp corner.
	if !winNear(rest, 18, 17, c.surface) || winNear(held, 18, 17, c.surface) {
		t.Errorf("the button does not morph from round to 8dp corners while pressed: %v, %v", pixelColor(rest, 18, 17), pixelColor(held, 18, 17))
	}
	if !winNear(rest, 80, 35, c.primary) {
		t.Errorf("the filled button is not the primary: %v", pixelColor(rest, 80, 35))
	}
	foc := paint(StateFocused)
	if !winNear(foc, 80, 12, md3xColors(lk).secondary) {
		t.Errorf("no secondary focus ring 2dp above the face: %v", pixelColor(foc, 80, 12))
	}
	if !winNear(foc, 80, 14, c.surface) {
		t.Errorf("the ring touches the face: %v", pixelColor(foc, 80, 14))
	}
}

// The slider's track splits round its bar handle, which narrows when
// focused; the progress value is a wave, not a line.
func TestMaterial3xSliderAndWavyProgress(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := materialLook(t, "material3x", 1)
	c := mdColors(lk)
	b := paintengine2d.XYWH(10, 10, 200, 44)
	slider := func(st ControlState) *paintengine2d.Image {
		return md3xPaint(lk, 220, 64, func(ctx *paintengine2d.Context) { lk.DrawSlider(ctx, b, st, 0.5) })
	}
	img := slider(StateNone)
	x0, x1 := SliderTravelOf(lk, b)
	hx := int(x0 + (x1-x0)*0.5)
	if !winNear(img, hx, 32, c.primary) {
		t.Errorf("no handle at the value: %v", pixelColor(img, hx, 32))
	}
	if !winNear(img, hx+5, 32, c.surface) || !winNear(img, hx-5, 32, c.surface) {
		t.Errorf("no gap between the handle and the track")
	}
	if !winNear(img, hx+30, 32, c.secondaryC) || !winNear(img, hx-30, 32, c.primary) {
		t.Errorf("the tracks are not primary (active) and secondary-container (inactive)")
	}
	if winSame(img, slider(StateFocused)) {
		t.Errorf("the handle does not narrow when focused")
	}
	// The wave: the value's stroke leaves the centre line.
	p := md3xPaint(lk, 220, 34, func(ctx *paintengine2d.Context) {
		lk.DrawProgressBar(ctx, paintengine2d.XYWH(10, 10, 200, 14), StateNone, 0.8, false, 0)
	})
	rows := map[int]bool{}
	for x := 20; x < 150; x++ {
		for y := 10; y < 24; y++ {
			if winNear(p, x, y, c.primary) {
				rows[y] = true
			}
		}
	}
	if len(rows) < 7 {
		t.Errorf("the progress value spans %d rows: a line, not a wave", len(rows))
	}
}

// A list is a rounded group of segments: 2dp gaps between rows, the
// selected row in the secondary container.
func TestMaterial3xSegmentedList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := materialLook(t, "material3x", 1)
	c := mdColors(lk)
	img := paintengine2d.NewImage(200, 84)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 200, 84), paintengine2d.Fill(c.menu))
	lk.DrawListRow(ctx, paintengine2d.XYWH(0, 0, 200, 42), StateNone, "")
	lk.DrawListRow(ctx, paintengine2d.XYWH(0, 42, 200, 42), StateChecked, "")
	if !winNear(img, 100, 42, c.menu) {
		t.Errorf("no gap between two rows: %v", pixelColor(img, 100, 42))
	}
	if !winNear(img, 100, 20, md3xColors(lk).segment) || !winNear(img, 100, 62, c.secondaryC) {
		t.Errorf("rows are not segments, the selected one secondary-container")
	}
}

// Tabs are primary tabs: a 3dp indicator under the selected label only.
func TestMaterial3xPrimaryTabs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := materialLook(t, "material3x", 1)
	c := mdColors(lk)
	b := paintengine2d.XYWH(0, 0, 200, 48)
	sel := md3xPaint(lk, 200, 48, func(ctx *paintengine2d.Context) { lk.DrawTab(ctx, b, StateNone, "Tab", true) })
	if !winNear(sel, 100, 46, c.primary) || winNear(sel, 100, 44, c.primary) {
		t.Errorf("no 3dp indicator under the label")
	}
	if winNear(sel, 8, 46, c.primary) {
		t.Errorf("the indicator spans the tab")
	}
	if off := md3xPaint(lk, 200, 48, func(ctx *paintengine2d.Context) { lk.DrawTab(ctx, b, StateNone, "Tab", false) }); winNear(off, 100, 46, c.primary) {
		t.Errorf("an unselected tab has the indicator")
	}
}

// The seed drives the scheme when the desktop's accent takes its place; the
// pack's own seed gives the pack back.
func TestMaterial3xAccentIsTheSeed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	own := accLook(t, "material3x", Hex("#6750a4"), 1)
	if got := colorHexPadded(mdColors(own).primary); got != "#6750a4" {
		t.Errorf("the own seed moved the primary to %s", got)
	}
	lk := accLook(t, "material3x", Hex("#26a269"), 1)
	want := md3xScheme(Hex("#26a269"), false)
	if c := mdColors(lk); !accentSame(c.primary, want["primary"]) || !accentSame(c.menu, want["surfaceContainer"]) {
		t.Errorf("green seed: primary %s menu %s, want %s %s", colorHexPadded(c.primary), colorHexPadded(c.menu),
			colorHexPadded(want["primary"]), colorHexPadded(want["surfaceContainer"]))
	}
}

// Text reads on every fill the engine paints it on.
func TestMaterial3xLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range md3xPackNames {
		lk := materialLook(t, n, 1)
		c, x := mdColors(lk), md3xColors(lk)
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
		}{
			{"text on surface", c.text, c.surface},
			{"filled button", c.onPrimary, c.primary},
			{"tonal button", c.onSecondaryC, c.secondaryC},
			{"outlined button", c.onSurfaceVar, c.surface},
			{"menu", c.text, c.menu},
			{"selected choice", x.onTertiaryC, x.tertiaryC},
			{"list segment", c.text, x.segment},
			{"selected row", c.onSecondaryC, c.secondaryC},
			{"dialog", c.text, c.dialog},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < 4.5 {
				t.Errorf("%s: %s %s on %s is %.2f:1", n, chk.what, colorHexPadded(chk.fg), colorHexPadded(chk.bg), r)
			}
		}
	}
}

// Follow the desktop moves between the Expressive twins.
func TestMaterial3xSchemeSiblings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := SchemeVariant("material3x", SchemeDark); got != "material3x-night" {
		t.Errorf("material3x in the dark: %q", got)
	}
	if got := SchemeVariant("material3x-night", SchemeLight); got != "material3x" {
		t.Errorf("material3x-night in the light: %q", got)
	}
}
