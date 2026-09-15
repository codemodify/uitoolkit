package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var adwPackNames = []string{"adwaita-gtk3", "adwaita", "adwaita-night"}

func TestAdwaitaPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	want := map[string]struct {
		label string
		year  int
		fam   ThemeName
	}{
		"adwaita-gtk3":  {"Adwaita (GTK 3)", 2014, ThemeLight},
		"adwaita":       {"Adwaita", 2020, ThemeLight},
		"adwaita-night": {"Adwaita Dark", 2020, ThemeDark},
	}
	for _, n := range adwPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "adwaita" || p.Label != w.label || p.Lineage != "GNOME" || p.Year != w.year || p.Tokens.Family != w.fam {
			t.Fatalf("%s: engine %q label %q lineage %q year %d family %q", n, p.Tokens.Engine, p.Label, p.Lineage, p.Year, p.Tokens.Family)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != "adwaita" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	order := []string{"clearlooks", "adwaita-gtk3", "adwaita"}
	for i := 1; i < len(order); i++ {
		if pos[order[i-1]] > pos[order[i]] {
			t.Fatalf("packs not ordered by year: %s after %s", order[i-1], order[i])
		}
	}
	// "adwaita" used to alias Breeze; the real pack wins now.
	if p, _ := LoadTheme("adwaita"); p.Tokens.Engine != "adwaita" {
		t.Fatalf("adwaita resolves to %q", p.Name)
	}
}

func TestAdwaitaPaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range adwPackNames {
		for _, scale := range []float32{1, 2} {
			lk := clLook(t, n, scale)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			img := paintengine2d.NewImage(64, 64)
			ctx := paintengine2d.NewContext(img)
			for _, r := range []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(2, 2, 3, 3),
				paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5)} {
				for _, st := range lunaStates {
					lunaCalls(lk, ctx, r, st)
				}
				lunaStateless(lk, ctx, r)
			}
			if ctx.SaveCount() != 0 {
				t.Fatalf("%s: %d saved states left", n, ctx.SaveCount())
			}
		}
	}
}

// GNOME puts the affirmative button at the right ("Cancel OK"), and the
// circular close button is painted where hit-testing looks for it.
func TestAdwaitaStyleHintAndCloseRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range adwPackNames {
		for _, sc := range []float32{1, 2} {
			lk := clLook(t, n, sc)
			if LookHint(lk, HintDialogPrimaryFirst) != 0 {
				t.Fatalf("%s: GNOME dialogs put the default button last", n)
			}
			if LookHint(lk, HintTabsCentered) != 0 {
				t.Fatalf("%s: notebook tabs start at the left", n)
			}
			b := paintengine2d.XYWH(10*sc, 10*sc, 300*sc, 170*sc)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Max.X > b.Max.X || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close %v not at the right of the header bar of %v (top %v)", n, sc, cr, b, in.Top)
			}
			if cr.Dx() != cr.Dy() {
				t.Fatalf("%s: close button %vx%v is not square", n, cr.Dx(), cr.Dy())
			}
			img := paintengine2d.NewImage(int(330*sc), int(200*sc))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
			c := adwColors(lk)
			// The × crosses at the centre in the title colour; the button's
			// wash shows between its arms.
			ctr := cr.Center()
			glyph := c.text
			if c.gtk3 {
				glyph = c.g3.text
			}
			if px := clPixel(img, int(ctr.X), int(ctr.Y)); clDist(px, glyph) > 0.08 {
				t.Fatalf("%s@%gx: close centre %s, want the glyph %s", n, sc, colorHexPadded(px), colorHexPadded(glyph))
			}
			if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
				t.Fatalf("%s: a frame too small for a header bar reports close rect %v", n, r)
			}
		}
	}
}

// libadwaita's button is currentColor at 10% over whatever is under it:
// #e6e6e7 on the light window, #38383c on the dark one; the suggested
// (default) button is the accent #3584e4.
func TestAdwaitaButtonWashes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for n, want := range map[string]string{"adwaita": "#e6e6e7", "adwaita-night": "#38383c"} {
		lk := clLook(t, n, 1)
		img := paintengine2d.NewImage(120, 40)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, 120, 40), paintengine2d.Fill(adwColors(lk).win))
		lk.DrawButton(ctx, paintengine2d.XYWH(4, 3, 110, 34), StateNone, "")
		if px := clPixel(img, 20, 20); !hexNear(px, want, 2) {
			t.Errorf("%s: button face %s, want %s", n, colorHexPadded(px), want)
		}
		lk.DrawButton(ctx, paintengine2d.XYWH(4, 3, 110, 34), StatePrimary, "")
		if px := clPixel(img, 20, 20); !hexNear(px, "#3584e4", 2) {
			t.Errorf("%s: suggested button %s, want the accent #3584e4", n, colorHexPadded(px))
		}
		// Buttons are flat: no border ring darker than the wash.
		if px := clPixel(img, 4, 20); hexNear(px, "#000000", 60) {
			t.Errorf("%s: the button has a dark border %s", n, colorHexPadded(px))
		}
	}
}

// The focus ring is 2px of the accent at half alpha, inside the control.
func TestAdwaitaFocusRingInside(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range []string{"adwaita", "adwaita-night"} {
		lk := clLook(t, n, 1)
		c := adwColors(lk)
		render := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(120, 40)
			ctx := paintengine2d.NewContext(img)
			ctx.DrawRect(paintengine2d.XYWH(0, 0, 120, 40), paintengine2d.Fill(c.win))
			lk.DrawButton(ctx, paintengine2d.XYWH(4, 3, 110, 34), st, "")
			return img
		}
		focused := render(StateFocused)
		want := adwOver(adwOver(c.win, c.wash(adwWashBtn)), c.focus)
		if px := clPixel(focused, 60, 4); clDist(px, want) > 0.01 {
			t.Errorf("%s: focus ring pixel %s, want the accent at half alpha %s", n, colorHexPadded(px), colorHexPadded(want))
		}
		if px := clPixel(focused, 60, 7); clDist(px, clPixel(render(StateNone), 60, 7)) > 0.001 {
			t.Errorf("%s: the ring is wider than 2px", n)
		}
		if px := clPixel(focused, 60, 1); clDist(px, c.win) > 0.001 {
			t.Errorf("%s: the ring paints outside the button", n)
		}
	}
}

// Overlay scroll bars: no arrows, thin when idle, wide under the pointer.
func TestAdwaitaOverlayScrollBars(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range []string{"adwaita", "adwaita-night"} {
		lk := clLook(t, n, 1)
		s := ScrollBarStyleOf(lk)
		if !s.Overlay || s.Arrows != ArrowsNone {
			t.Fatalf("%s: scroll bar %+v, want an overlay bar without arrows", n, s)
		}
		width := func(hovered bool) int {
			img := paintengine2d.NewImage(40, 200)
			ctx := paintengine2d.NewContext(img)
			p := ScrollGeometry(lk, paintengine2d.XYWH(0, 0, 40, 200), true, 800, 200, 100, false)
			DrawScrollBarParts(lk, ctx, p, true, ScrollState{Hovered: hovered})
			y := int(p.Thumb.Center().Y)
			w := 0
			for x := 0; x < 40; x++ {
				if _, _, _, a := img.PremulAt(x, y); a > 40 {
					w++
				}
			}
			return w
		}
		idle, hot := width(false), width(true)
		if idle == 0 || hot <= idle+3 {
			t.Fatalf("%s: slider %dpx idle, %dpx hovered: want a thin indicator that widens", n, idle, hot)
		}
	}
	if s := ScrollBarStyleOf(clLook(t, "adwaita-gtk3", 1)); s.Overlay || s.Thickness != 13 {
		t.Fatalf("GTK 3.14 scroll bar %+v, want the 13px gutter", s)
	}
}

// Rows: the accent at 25% when selected, kept when the view loses focus,
// washed to a neutral tint in the backdrop; the label keeps its colour.
func TestAdwaitaRowSelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range adwPackNames {
		lk := clLook(t, n, 1)
		c := adwColors(lk)
		row := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(200, 30)
			ctx := paintengine2d.NewContext(img)
			ctx.DrawRect(paintengine2d.XYWH(0, 0, 200, 30), paintengine2d.Fill(c.view))
			lk.DrawListRow(ctx, paintengine2d.XYWH(0, 3, 200, 24), st, "")
			return img
		}
		sel := row(StateChecked)
		if !bytesEqual(sel.Pix, row(StateChecked|StateInactive).Pix) {
			t.Errorf("%s: an unfocused view greys its selection", n)
		}
		if bytesEqual(sel.Pix, row(StateChecked|StateBackdrop).Pix) {
			t.Errorf("%s: the backdrop does not subdue the selection", n)
		}
		want := adwOver(c.view, c.sel)
		if c.gtk3 {
			want = c.g3.sel
		}
		if px := clPixel(sel, 100, 15); clDist(px, want) > 0.002 {
			t.Errorf("%s: selected row %s, want %s", n, colorHexPadded(px), colorHexPadded(want))
		}
	}
}

// Labels read on every fill the engine paints them on.
func TestAdwaitaLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range adwPackNames {
		lk := clLook(t, n, 1)
		c := adwColors(lk)
		checks := []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.win, 4.5},
			{"text on view", c.viewText, c.view, 4.5},
			{"button label", c.text, adwOver(c.win, c.wash(adwWashBtn)), 4.5},
			{"suggested label", c.accentFg, c.accentBg, 3},
			{"tooltip", c.tooltipFg, adwOver(c.win, c.tooltip), 4.5},
			{"selected row", c.viewText, adwOver(c.view, c.sel), 4.5},
			{"dim label", c.dim, c.win, 3},
		}
		if c.gtk3 {
			checks[5] = struct {
				what   string
				fg, bg paintengine2d.Color
				min    float64
			}{"selected row", c.g3.selFg, c.g3.sel, 3}
		}
		for _, chk := range checks {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Check boxes fill with the accent when checked and are empty rings of
// currentColor when not; the dark knob is #d2d2d2 until it is switched on.
func TestAdwaitaIndicators(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := clLook(t, "adwaita", 1)
	img := paintengine2d.NewImage(30, 30)
	ctx := paintengine2d.NewContext(img)
	lk.Engine().CheckIndicator(lk, ctx, paintengine2d.XYWH(5, 5, 20, 20), StateNone, true)
	if px := clPixel(img, 15, 7); !hexNear(px, "#3584e4", 2) {
		t.Fatalf("checked box %s, want the accent", colorHexPadded(px))
	}
	img = paintengine2d.NewImage(30, 30)
	ctx = paintengine2d.NewContext(img)
	lk.Engine().CheckIndicator(lk, ctx, paintengine2d.XYWH(5, 5, 20, 20), StateNone, false)
	if _, _, _, a := img.PremulAt(15, 15); a != 0 {
		t.Fatal("an unchecked box is filled")
	}
	if dark := adwColors(clLook(t, "adwaita-night", 1)); !hexNear(dark.knob, "#d2d2d2", 1) {
		t.Fatalf("dark knob %s", colorHexPadded(dark.knob))
	}
}
