package style

import (
	"reflect"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var fluentPackNames = []string{"fluent", "fluent-night"}

func TestFluentPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winCheckPacks(t, "fluent", []winPackWant{
		{"fluent", "Fluent", 2021, ThemeLight},
		{"fluent-night", "Fluent Dark", 2021, ThemeDark},
	})
	// The engine packs replace the legacy base-engine packs (2017) and keep
	// their bevel language; the old alias still resolves.
	for _, n := range fluentPackNames {
		p, _ := LoadTheme(n)
		if p.Tokens.Bevel != BevelFluentAccent {
			t.Fatalf("%s bevel %q", n, p.Tokens.Bevel)
		}
	}
	if p, _ := LoadTheme("fluent-dark"); p.Name != "fluent-night" {
		t.Fatalf("fluent-dark alias resolves to %q", p.Name)
	}
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	if pos["aero"] > pos["win8"] || pos["win8"] > pos["win10"] || pos["win10-night"] > pos["fluent"] {
		t.Fatalf("Windows packs out of year order: %v", pos)
	}
}

func TestFluentColourTables(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for k := range fluentDark {
		if _, ok := fluentLight[k]; !ok {
			t.Errorf("fluentDark key %q is not in fluentLight", k)
		}
	}
	for k := range fluentLight {
		if _, ok := fluentDark[k]; !ok {
			t.Errorf("fluentLight key %q is not in fluentDark", k)
		}
	}
	for _, n := range fluentPackNames {
		lk := winLook(t, n, 1)
		c := fluentColors(lk)
		if c != fluentColors(lk) {
			t.Fatalf("%s: colours are rebuilt per paint", n)
		}
		winNoSentinel(t, n, reflect.ValueOf(*c))
	}
}

func TestFluentPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, fluentPackNames)
}

func TestFluentPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsInsideRect(t, fluentPackNames)
}

func TestFluentCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range fluentPackNames {
		winCheckClose(t, n)
	}
}

func TestFluentStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range fluentPackNames {
		winCheckHints(t, n)
	}
}

// Labels read on the (translucent) fills they sit on, flattened over Mica.
func TestFluentLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range fluentPackNames {
		lk := winLook(t, n, 1)
		c := fluentColors(lk)
		menu := Mix(c.flyout, c.subtle, c.subtle.A)
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on Mica", c.text, c.bg, 4.5},
			{"secondary text", c.text2, c.bg, 4.5},
			{"button label", c.text, c.over(c.ctl[0]), 4.5},
			{"hot button label", c.text, c.over(c.ctl[1]), 4.5},
			{"accent button label", c.onAccent[0], c.accent[0], 4.5},
			{"text on field", lk.fieldText(), c.field, 4.5},
			{"hot menu label", lk.Engine().MenuTextColor(lk, true), menu, 4.5},
			{"tool tip", c.text, c.flyout, 4.5},
			{"page", c.text, c.page, 4.5},
			{"selected table row", c.text, Mix(c.field, c.tint, c.tint.A), 4.5},
			{"disabled label", c.textDis, c.bg, 2},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// The elevation border: a button's bottom edge is darker than its sides in
// the light theme; in the dark theme its top edge is lighter.
func TestFluentElevationBorder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range fluentPackNames {
		lk := winLook(t, n, 1)
		c := fluentColors(lk)
		img := paintengine2d.NewImage(100, 44)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 100, 44), paintengine2d.Fill(c.bg))
		lk.DrawButton(ctx, paintengine2d.XYWH(5, 5, 90, 32), StateNone, "")
		top, _, _, _ := img.PremulAt(50, 5)
		side, _, _, _ := img.PremulAt(5, 20)
		bot, _, _, _ := img.PremulAt(50, 36)
		if c.dark && top <= side+2 {
			t.Errorf("%s: top edge %d not lighter than the side %d", n, top, side)
		}
		if !c.dark && bot+8 >= side {
			t.Errorf("%s: bottom edge %d not darker than the side %d", n, bot, side)
		}
	}
}

// List selection is the accent pill at the left of a subtle row; WinUI
// keeps it while the view is unfocused and greys the pill in an inactive
// window.
func TestFluentListSelectionPill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range fluentPackNames {
		lk := winLook(t, n, 1)
		c := fluentColors(lk)
		paint := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(160, 32)
			ctx := paintengine2d.NewContext(img)
			ctx.DrawRect(paintengine2d.XYWH(0, 0, 160, 32), paintengine2d.Fill(c.field))
			lk.DrawListRow(ctx, paintengine2d.XYWH(0, 0, 160, 32), st, "")
			return img
		}
		on := paint(StateChecked)
		item := fluentItemRect(lk, paintengine2d.XYWH(0, 0, 160, 32))
		px := int(item.Min.X + 1)
		if !winNear(on, px, 16, c.accent[0]) {
			t.Errorf("%s: no accent pill at the left of a selected row", n)
		}
		if winNear(paint(StateNone), px, 16, c.accent[0]) {
			t.Errorf("%s: an unselected row carries the pill", n)
		}
		if !winSame(on, paint(StateChecked|StateInactive)) {
			t.Errorf("%s: an unfocused view must keep its selection", n)
		}
		if winNear(paint(StateChecked|StateInactive|StateBackdrop), px, 16, c.accent[0]) {
			t.Errorf("%s: an inactive window's selection keeps the accent pill", n)
		}
	}
}

// Keyboard focus is the two-line focus visual: the outer line in
// FocusStrokeColorOuter, the inner in FocusStrokeColorInner.
func TestFluentFocusVisual(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range fluentPackNames {
		lk := winLook(t, n, 1)
		c := fluentColors(lk)
		img := paintengine2d.NewImage(100, 44)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 100, 44), paintengine2d.Fill(c.bg))
		lk.DrawButton(ctx, paintengine2d.XYWH(5, 5, 90, 32), StateFocused, "")
		outer, inner := c.over(c.focusOuter), Mix(c.over(c.ctl[0]), c.focusInner, c.focusInner.A)
		if !winNear(img, 50, 5, outer) || !winNear(img, 50, 6, outer) || !winNear(img, 50, 7, inner) {
			t.Errorf("%s: focus visual not a 2px %s line over a 1px %s one", n, colorHexPadded(outer), colorHexPadded(inner))
		}
	}
}

// The progress bar is a thin accent bar centred in its rect.
func TestFluentProgressIsThinAccentBar(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := winLook(t, "fluent", 1)
	c := fluentColors(lk)
	img := paintengine2d.NewImage(120, 20)
	lk.DrawProgressBar(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 120, 20), StateNone, 0.5, false, 0)
	if !winNear(img, 30, 10, c.accent[0]) {
		t.Fatal("no accent indicator in the middle")
	}
	if _, _, _, a := img.PremulAt(30, 5); a != 0 {
		t.Fatal("the indicator is taller than WinUI's 3px")
	}
}

func winSame(a, b *paintengine2d.Image) bool {
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}

// A user pack that names the engine and a dark palette but no scheme gets
// the dark tables.
func TestWindowsEnginesInferDarkScheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, engine := range []string{"metro", "fluent"} {
		raw := []byte(`{"label": "Mine", "engine": "` + engine + `", "family": "dark", "colors": {"background": "#1e1e1e", "text": "#ffffff"}}`)
		p, err := parseThemeFile("zz-mine-"+engine, raw, ThemeSourceUser)
		if err != nil {
			t.Fatal(err)
		}
		tok := p.Tokens
		lk := newClassic(string(tok.Family), tok.Palette, packMetrics(DensityDefault, tok, CornersTheme, IconSizeMedium),
			CornersTheme, IconSetClassic, IconSizeMedium, tok)
		if lk.Engine().ID() != engine {
			t.Fatalf("%s: user pack paints with %q", engine, lk.Engine().ID())
		}
		switch engine {
		case "metro":
			if metroSchemeIndex(lk) != 2 {
				t.Errorf("metro: dark user pack got scheme %d", metroSchemeIndex(lk))
			}
		case "fluent":
			if !fluentColors(lk).dark {
				t.Error("fluent: dark user pack got the light tokens")
			}
		}
	}
}
