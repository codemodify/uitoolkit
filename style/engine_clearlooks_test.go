package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var clPackNames = []string{"clearlooks", "human", "cleanlooks"}

func clLook(t *testing.T, name string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("pack %q not found", name)
	}
	var lk LookAndFeel = p.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	c, ok := lk.(*Classic)
	if !ok {
		t.Fatalf("%s: look is %T", name, lk)
	}
	return c
}

func TestClearlooksPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	want := map[string]struct {
		label, lineage string
		year           int
	}{
		"clearlooks": {"Clearlooks", "GNOME", 2005},
		"human":      {"Human", "Ubuntu", 2006},
		"cleanlooks": {"Cleanlooks", "Qt", 2007},
	}
	for _, n := range clPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "clearlooks" || p.Label != w.label || p.Lineage != w.lineage || p.Year != w.year {
			t.Fatalf("%s: engine %q label %q lineage %q year %d", n, p.Tokens.Engine, p.Label, p.Lineage, p.Year)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != "clearlooks" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
		if p.Tokens.Family != ThemeLight {
			t.Fatalf("%s family %q", n, p.Tokens.Family)
		}
	}
	order := []string{"bluecurve", "clearlooks", "human", "cleanlooks"}
	for i := 1; i < len(order); i++ {
		if pos[order[i-1]] > pos[order[i]] {
			t.Fatalf("packs not ordered by year: %s after %s", order[i-1], order[i])
		}
	}
}

// hexNear reports whether c is within tol of the 8-bit colour s per
// channel.
func hexNear(c paintengine2d.Color, s string, tol int) bool {
	w := Hex(s)
	r, g, b, _ := c.Premul8()
	wr, wg, wb, _ := w.Premul8()
	d := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(r, wr) <= tol && d(g, wg) <= tol && d(b, wb) <= tol
}

// The shading maths: HLS lightness and saturation × k. The expected values
// are the 2005 engine's shade table on GNOME 2.12's #efebe7 and #628cb2.
func TestClearlooksShadeMaths(t *testing.T) {
	bg, sel := Hex("#efebe7"), Hex("#628cb2")
	for _, tc := range []struct {
		c    paintengine2d.Color
		k    float32
		want string
	}{
		{bg, 1.065, "#fcfbfa"}, {bg, 0.93, "#e2dbd4"}, {bg, 0.896, "#dbd3cb"}, {bg, 0.85, "#d1c8bf"},
		{bg, 0.768, "#c0b5a9"}, {bg, 0.665, "#aa9c8f"}, {bg, 0.4, "#655e56"},
		{bg, 0.5, "#81756a"}, {bg, 0.62, "#9f9284"},
		{sel, 1.42, "#a7c6e1"}, {sel, 1.05, "#6993b9"}, {sel, 0.65, "#465b6e"},
		{bg, 1, "#efebe7"},
	} {
		if got := clShade(tc.c, tc.k); !hexNear(got, tc.want, 1) {
			t.Errorf("shade(%s, %v) = %s, want %s", colorHexPadded(tc.c), tc.k, colorHexPadded(got), tc.want)
		}
	}
	// HLS round-trips.
	for _, s := range []string{"#efebe7", "#628cb2", "#ff6d0c", "#000000", "#ffffff", "#4464ac"} {
		if got := clShade(Hex(s), 1); !hexNear(got, s, 0) {
			t.Errorf("shade(%s, 1) = %s", s, colorHexPadded(got))
		}
	}
	// The pack's colour set is built from that table.
	c := clColors(clLook(t, "clearlooks", 1))
	for i, want := range []string{"#fcfbfa", "#e2dbd4", "#dbd3cb", "#d1c8bf", "#c0b5a9", "#aa9c8f", "#655e56"} {
		if !hexNear(c.s[i], want, 1) {
			t.Errorf("shade[%d] = %s, want %s", i, colorHexPadded(c.s[i]), want)
		}
	}
	if !hexNear(c.spot[2], "#465b6e", 1) || !hexNear(c.upper, "#81756a", 1) {
		t.Errorf("spot3 %s / border %s", colorHexPadded(c.spot[2]), colorHexPadded(c.upper))
	}
}

func TestClearlooksPaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		for _, scale := range []float32{1, 2} {
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), clLook(t, n, scale))
		}
	}
}

// Degenerate rects (empty, a pixel, slivers) paint without panicking.
func TestClearlooksPaintsDegenerateRects(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		for _, sc := range []float32{1, 2} {
			lk := clLook(t, n, sc)
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

// GTK puts the affirmative button last ("Cancel OK"), and the close button
// is painted where hit-testing looks for it: a white × on the title
// colour at the right of the title bar.
func TestClearlooksStyleHintAndCloseRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		for _, sc := range []float32{1, 2} {
			lk := clLook(t, n, sc)
			if LookHint(lk, HintDialogPrimaryFirst) != 0 {
				t.Fatalf("%s: GTK dialogs put the default button last", n)
			}
			if LookHint(lk, HintTabsCentered) != 0 {
				t.Fatalf("%s: GTK tabs start at the left", n)
			}
			b := paintengine2d.XYWH(10*sc, 10*sc, 280*sc, 160*sc)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Max.X > b.Max.X || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close %v not at the right of the title bar of %v (top %v)", n, sc, cr, b, in.Top)
			}
			if cr.Dx() != cr.Dy() {
				t.Fatalf("%s: close button %vx%v is not square", n, cr.Dx(), cr.Dy())
			}
			img := paintengine2d.NewImage(int(310*sc), int(190*sc))
			ctx := paintengine2d.NewContext(img)
			lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
			ctr := cr.Center()
			r, g, bl, _ := img.PremulAt(int(ctr.X), int(ctr.Y))
			if r < 200 || g < 200 || bl < 200 {
				t.Fatalf("%s@%gx: close centre rgb(%d,%d,%d), want the white ×", n, sc, r, g, bl)
			}
			// Between the border and the ×, the button shows the title colour.
			x, y := int(cr.Center().X), int(cr.Min.Y+cr.Dy()*0.18)
			if px := clPixel(img, x, y); clDist(px, clColors(lk).caption) > clDist(px, Hex("#ffffff")) {
				t.Fatalf("%s@%gx: close button face at (%d,%d) is %s, not the title colour", n, sc, x, y, colorHexPadded(px))
			}
			if r := lk.WindowCloseRect(paintengine2d.XYWH(0, 0, 20, 20)); !r.Empty() {
				t.Fatalf("%s: a frame too small for a title bar reports close rect %v", n, r)
			}
		}
	}
}

// clPixel reads an opaque pixel as a colour.
func clPixel(img *paintengine2d.Image, x, y int) paintengine2d.Color {
	r, g, b, _ := img.PremulAt(x, y)
	return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
}

// clDist is the squared RGB distance of two colours.
func clDist(a, b paintengine2d.Color) float32 {
	dr, dg, db := a.R-b.R, a.G-b.G, a.B-b.B
	return dr*dr + dg*dg + db*db
}

func clRaster(lk *Classic, draw func(ctx *paintengine2d.Context, b paintengine2d.Rect)) *paintengine2d.Image {
	img := paintengine2d.NewImage(200, 30)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 200, 30), paintengine2d.Fill(lk.Palette().Field))
	draw(ctx, paintengine2d.XYWH(0, 3, 200, 24))
	return img
}

// GTK keeps a selection when its view loses focus and subdues it only in
// the backdrop (an inactive window).
func TestClearlooksSelectionKeptUntilBackdrop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range append([]string{"bluecurve"}, clPackNames...) {
		lk := clLook(t, n, 1)
		row := func(st ControlState) []byte {
			return clRaster(lk, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawListRow(ctx, b, st, "Row") }).Pix
		}
		sel, unfocused, backdrop := row(StateChecked), row(StateChecked|StateInactive), row(StateChecked|StateBackdrop)
		if !bytesEqual(sel, unfocused) {
			t.Errorf("%s: an unfocused view greys its selection", n)
		}
		if bytesEqual(sel, backdrop) {
			t.Errorf("%s: the backdrop does not subdue the selection", n)
		}
		if bytesEqual(sel, row(StateNone)) {
			t.Errorf("%s: a selected row paints like an idle one", n)
		}
	}
}

// Labels read on the fills the engine paints them on.
func TestClearlooksLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		lk := clLook(t, n, 1)
		c := clColors(lk)
		e := lk.Engine()
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.fg, c.bg, 4.5},
			{"text on field", c.text, c.base, 4.5},
			{"button label", c.fg, c.bands[1].Color, 4.5},
			{"selected row", c.rowSelFg, c.rowSel[0].Color, 3},
			{"hot menu item", e.MenuTextColor(lk, true), c.menuItem[0].Color, 3},
			{"tooltip", c.tipText, c.tip, 4.5},
			{"title", c.captionText, c.capGrad[1].Color, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// The candy bar is striped: across the filled part the colour alternates
// (Human: cells with light edges; Clearlooks and Cleanlooks: stripes).
func TestClearlooksProgressIsStriped(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		lk := clLook(t, n, 1)
		img := paintengine2d.NewImage(220, 24)
		ctx := paintengine2d.NewContext(img)
		lk.DrawProgressBar(ctx, paintengine2d.XYWH(2, 2, 216, 20), StateNone, 1, false, 0)
		changes := 0
		var prev [3]uint8
		for x := 8; x < 210; x++ {
			r, g, b, _ := img.PremulAt(x, 16)
			cur := [3]uint8{r, g, b}
			if x > 8 && cur != prev {
				changes++
			}
			prev = cur
		}
		if changes < 12 {
			t.Errorf("%s: progress fill looks flat (%d colour changes along a row)", n, changes)
		}
	}
}

// The selected tab wears the spot-colour stripe along its top edge.
func TestClearlooksSelectedTabStripe(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		lk := clLook(t, n, 1)
		c := clColors(lk)
		tab := func(selected bool) *paintengine2d.Image {
			img := paintengine2d.NewImage(100, 34)
			ctx := paintengine2d.NewContext(img)
			lk.DrawTab(ctx, paintengine2d.XYWH(2, 2, 96, 30), StateNone, "Tab", selected)
			return img
		}
		want := c.spot[1]
		switch c.flavour {
		case clHuman:
			want = c.spotBase
		case clCleanlooks:
			want = c.qTabMid
		}
		if !nxNear(tab(true), 50, 4, want) {
			r, g, b, _ := tab(true).PremulAt(50, 4)
			t.Errorf("%s: selected tab top is rgb(%d,%d,%d), want the stripe %s", n, r, g, b, colorHexPadded(want))
		}
		if nxNear(tab(false), 50, 6, want) {
			t.Errorf("%s: an unselected tab wears the stripe", n)
		}
	}
}

func TestClearlooksScrollBars(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range clPackNames {
		s := ScrollBarStyleOf(clLook(t, n, 1))
		if s.Overlay || s.Arrows != ArrowsEnds || s.Thickness != 15 {
			t.Fatalf("%s scroll bar %+v: want GTK 2's 15px gutter with a stepper at each end", n, s)
		}
		if clLook(t, n, 1).eng().FieldFocusRing(clLook(t, n, 1)) {
			t.Fatalf("%s: GTK 2 entries show focus with their border, not a ring", n)
		}
	}
}
