package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var fusionPackNames = []string{"fusion", "fusion-night"}

func TestFusionPacksRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "fusion", "Qt", 2012, map[string]string{"fusion": "Fusion", "fusion-night": "Fusion Dark"})
	// The dark pack replaces the legacy base-engine "fusion-night"; there
	// is no second dark Fusion under another name.
	if _, ok := kdePositions()["fusion-dark"]; ok {
		t.Fatal("no pack may be named fusion-dark: the dark pack is fusion-night")
	}
	if p, _ := LoadTheme("fusion-night"); !strings.Contains(p.Summary, "Not historical") {
		t.Fatalf("fusion-night must say Qt's 2012 Fusion had no dark palette: %q", p.Summary)
	}
	pos := kdePositions()
	if !(pos["oxygen"] < pos["fusion"] && pos["fusion"] < pos["breeze"]) {
		t.Fatalf("Oxygen (2008), Fusion (2012) and Breeze (2014) out of year order: %v %v %v", pos["oxygen"], pos["fusion"], pos["breeze"])
	}
}

func TestFusionStyleHintsAndScrollBars(t *testing.T) {
	for _, n := range fusionPackNames {
		lk := mustLook(t, n)
		// Qt on Linux: KDE button-box layout, accept before reject.
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 0 {
			t.Fatalf("%s: want OK before Cancel, left-aligned tabs and left-aligned form labels", n)
		}
		s := ScrollBarStyleOf(lk)
		if s.Arrows != ArrowsEnds || s.Overlay || s.Thickness != 14 {
			t.Fatalf("%s scroll bar %+v: want a 14px bar with an arrow button at each end", n, s)
		}
		if o := lk.TabOutset(); o.Right <= 0 {
			t.Fatalf("%s: the selected tab overlaps its neighbour, got outset %+v", n, o)
		}
	}
}

// Fusion's shades follow Qt's documented HSV-value arithmetic.
func TestFusionColourArithmetic(t *testing.T) {
	for _, c := range []struct {
		got  paintengine2d.Color
		want string
	}{
		{darkerPct(Hex("#efefef"), 140), "#ababab"}, // the outline of the light palette
		{lighterPct(Hex("#efefef"), 150), "#ffffff"},
		{lighterPct(Hex("#308cc6"), 100), "#308cc6"},
		{withLightness(Hex("#ff0000"), 0.25), "#800000"},
	} {
		if colorHexPadded(c.got) != c.want {
			t.Fatalf("got %s want %s", colorHexPadded(c.got), c.want)
		}
	}
	for _, s := range []string{"#308cc6", "#2a82da", "#353535", "#ffffdc", "#7f7f7f"} {
		c := Hex(s)
		h, sat, v := hsvOf(c)
		if got := colorHexPadded(hsvColor(h, sat, v, 1)); got != s {
			t.Fatalf("HSV round trip of %s gave %s", s, got)
		}
	}
}

func TestFusionCloseButtonAgreesWithPaint(t *testing.T) {
	for _, n := range fusionPackNames {
		kdeCheckClose(t, n)
	}
}

func TestFusionPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range fusionPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}

// ---- helpers shared by the Fusion, Oxygen and Breeze tests -----------------------------

func kdePositions() map[string]int {
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		if _, dup := pos[n]; dup {
			pos[n] = -1 // flagged below
			continue
		}
		pos[n] = i
	}
	return pos
}

// kdeCheckPacks checks that every pack in labels is registered once, paints
// with engine, and carries its label, year and lineage.
func kdeCheckPacks(t *testing.T, engine, lineage string, year int, labels map[string]string) {
	t.Helper()
	pos := kdePositions()
	for n, label := range labels {
		i, ok := pos[n]
		if !ok {
			t.Fatalf("pack %q not registered", n)
		}
		if i < 0 {
			t.Fatalf("pack %q listed twice", n)
		}
		p, _ := LoadTheme(n)
		if p.Tokens.Engine != engine || p.Lineage != lineage || p.Year != year || p.Label != label {
			t.Fatalf("%s: engine %q lineage %q year %d label %q", n, p.Tokens.Engine, p.Lineage, p.Year, p.Label)
		}
		if lk := p.Look(); lk.Engine().ID() != engine {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
}

// kdeCheckClose checks that the close button sits at the right of the
// title bar and that WindowCloseRect covers exactly what the button paints:
// a frame with the button and one without differ only inside the rect.
func kdeCheckClose(t *testing.T, n string) {
	t.Helper()
	for _, scale := range []float32{1, 2} {
		lk := WithScale(mustLook(t, n), scale).(*Classic)
		b := paintengine2d.XYWH(4*scale, 4*scale, 300*scale, 180*scale)
		cr := lk.WindowCloseRect(b)
		in := lk.WindowFrameInsets()
		if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.2 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
			t.Fatalf("%s@%gx: close rect %v not at the right of the title bar of %v (top inset %v)", n, scale, cr, b, in.Top)
		}
		for _, ws := range []WindowState{{Active: true}, {Active: false}} {
			with, without := ws, ws
			with.CanClose = true
			a := kdeFrame(lk, b, with)
			z := kdeFrame(lk, b, without)
			x0, y0, x1, y1 := a.Width, a.Height, -1, -1
			for y := 0; y < a.Height; y++ {
				for x := 0; x < a.Width; x++ {
					if kdeDiffers(a, z, x, y) {
						x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
					}
				}
			}
			if x1 < 0 {
				t.Fatalf("%s@%gx %+v: no close button painted", n, scale, with)
			}
			slack := int(scale)
			if x0 < int(cr.Min.X)-slack || y0 < int(cr.Min.Y)-slack || x1 > int(cr.Max.X)+slack || y1 > int(cr.Max.Y)+slack {
				t.Fatalf("%s@%gx %+v: close button painted over (%d,%d)-(%d,%d), hit rect %v", n, scale, with, x0, y0, x1, y1, cr)
			}
		}
	}
}

func kdeFrame(lk *Classic, b paintengine2d.Rect, ws WindowState) *paintengine2d.Image {
	img := paintengine2d.NewImage(int(b.Max.X+b.Min.X), int(b.Max.Y+b.Min.Y))
	ctx := paintengine2d.NewContext(img)
	lk.DrawWindowFrame(ctx, b, "", ws)
	return img
}

func kdeDiffers(a, b *paintengine2d.Image, x, y int) bool {
	r0, g0, b0, a0 := a.PremulAt(x, y)
	r1, g1, b1, a1 := b.PremulAt(x, y)
	d := func(p, q uint8) bool { return int(p) > int(q)+2 || int(q) > int(p)+2 }
	return d(r0, r1) || d(g0, g1) || d(b0, b1) || d(a0, a1)
}

// kdeExtras paints what aquaExercise does not reach — the current item's
// focus mark, item views' frames, rows in every item state, the tab pane
// and the window background — and fails when one panics, leaves saved
// states or paints outside its rect.
func kdeExtras(t *testing.T, name string, lk *Classic) {
	t.Helper()
	type cell struct {
		n    string
		w, h float32
		draw func(ctx *paintengine2d.Context, b paintengine2d.Rect)
	}
	var cells []cell
	for _, st := range []ControlState{
		StateNone, StateHovered, StateFocused, StateChecked, StateChecked | StateFocused,
		StateChecked | StateInactive, StateChecked | StateFocused | StateInactive, StateChecked | StateBackdrop,
		StateChecked | StateHovered, StateDisabled, StateDisabled | StateChecked | StateFocused,
	} {
		st := st
		sn := fmt.Sprintf("%#x", uint32(st))
		cells = append(cells,
			cell{"item focus " + sn, 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawItemFocus(ctx, b, st) }},
			cell{"view frame " + sn, 200, 120, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawViewFrame(ctx, b, st) }},
			cell{"list row " + sn, 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawListRow(ctx, b, st, "List row") }},
			cell{"tree row " + sn, 170, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTreeRow(ctx, b, st, true, false, 1, "Archives", false)
			}},
			cell{"table cell " + sn, 70, 24, func(ctx *paintengine2d.Context, b paintengine2d.Rect) {
				lk.DrawTableCell(ctx, b, st, "Cell", AlignEnd, nil)
			}},
		)
	}
	cells = append(cells,
		cell{"tab pane", 300, 160, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawTabPane(ctx, b) }},
		cell{"window background", 320, 240, func(ctx *paintengine2d.Context, b paintengine2d.Rect) { lk.DrawWindowBackground(ctx, b) }},
	)
	sc := lk.Scale()
	m := 6 * sc
	for _, c := range cells {
		w, h := c.w*sc, c.h*sc
		img := paintengine2d.NewImage(int(w+2*m), int(h+2*m))
		ctx := paintengine2d.NewContext(img)
		b := paintengine2d.XYWH(m, m, w, h)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%s: %s panicked: %v", name, c.n, r)
				}
			}()
			c.draw(ctx, b)
		}()
		if ctx.SaveCount() != 0 {
			t.Fatalf("%s: %s left %d saved states", name, c.n, ctx.SaveCount())
		}
		if x, y, ok := aquaOutside(img, b); ok {
			t.Errorf("%s: %s painted outside its rect at (%d,%d) of %v", name, c.n, x, y, b)
		}
	}
}
