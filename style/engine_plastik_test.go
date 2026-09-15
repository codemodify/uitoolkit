package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var plastikPackNames = []string{"plastik", "plastique"}

func TestPlastikPacksRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "plastik", "KDE", 2004, map[string]string{"plastik": "Plastik"})
	kdeCheckPacks(t, "plastik", "Qt", 2006, map[string]string{"plastique": "Plastique"})
	pos := kdePositions()
	// Keramik (2002), Plastik (2004), Plastique (2006), Oxygen (2008),
	// Fusion (2012).
	order := []string{"keramik", "plastik", "plastique", "oxygen", "fusion"}
	for i := 1; i < len(order); i++ {
		if pos[order[i-1]] > pos[order[i]] {
			t.Fatalf("%s sorts after %s: %v", order[i-1], order[i], pos)
		}
	}
}

// Plastik carries KDE 3.5's default colour scheme; Plastique Qt 4's
// standard palette (the same window, button and highlight).
func TestPlastikCarriesTheKDE35Scheme(t *testing.T) {
	for _, n := range plastikPackNames {
		p := mustLook(t, n).Palette()
		for _, c := range []struct {
			name string
			got  paintengine2d.Color
			want string
		}{
			{"background", p.Background, "#efefef"},
			{"button", p.SurfaceAlt, "#dddfe4"},
			{"selection", p.Selection, "#678db2"},
			{"selected text", p.TextOnAccent, "#ffffff"},
			{"base", p.Field, "#ffffff"},
		} {
			if got := colorHexPadded(c.got); got != c.want {
				t.Errorf("%s %s = %s, want %s", n, c.name, got, c.want)
			}
		}
	}
	if got := colorHexPadded(mustLook(t, "plastik").X("alternate", paintengine2d.Color{})); got != "#edf4f9" {
		t.Errorf("plastik alternate background %s, want KDE 3.5's #edf4f9", got)
	}
}

// The contour: KDE 3.5 derives it from the button colour (#8c8d90 on
// #dddfe4, as measured), Plastique from the window, a neutral grey.
func TestPlastikContour(t *testing.T) {
	if c := plastikColors(mustLook(t, "plastik")).contour; !kdeClose(c, Hex("#8c8d90"), 2) {
		t.Fatalf("plastik contour %s, want #8c8d90", colorHexPadded(c))
	}
	c := plastikColors(mustLook(t, "plastique")).contour
	if c.R != c.G || c.G != c.B || !kdeClose(c, Hex("#868686"), 4) {
		t.Fatalf("plastique contour %s, want a neutral grey near #868686", colorHexPadded(c))
	}
}

func TestPlastikStyleHintsAndScrollBars(t *testing.T) {
	for _, c := range []struct {
		name       string
		labelRight int
	}{{"plastik", 0}, {"plastique", 1}} {
		lk := mustLook(t, c.name)
		// KDE's OK-before-Cancel; KDE 3 left-aligned form labels, Qt 4's
		// QFormLayout right-aligns them in Plastique.
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != c.labelRight {
			t.Fatalf("%s: want OK before Cancel, left-aligned tabs and form labels right = %d", c.name, c.labelRight)
		}
		s := ScrollBarStyleOf(lk)
		if s.Arrows != ArrowsTripleEnd || s.Overlay || s.Thickness != 16 {
			t.Fatalf("%s scroll bar %+v: want a 16px bar with KDE 3's three step buttons", c.name, s)
		}
		if o := lk.TabOverlap(); o != 1 {
			t.Fatalf("%s: neighbouring tabs share their contour, got overlap %v", c.name, o)
		}
		if !lk.PopupShadow(PopupMenu).Zero() {
			t.Fatalf("%s: KDE 3 menus cast no shadow", c.name)
		}
	}
}

// plastikCount counts pixels of img within tol of want inside r.
func plastikCount(img *paintengine2d.Image, r paintengine2d.Rect, want paintengine2d.Color, tol int) int {
	n := 0
	wr, wg, wb, _ := want.Premul8()
	for y := int(r.Min.Y); y < int(r.Max.Y); y++ {
		for x := int(r.Min.X); x < int(r.Max.X); x++ {
			cr, cg, cb, _ := img.PremulAt(x, y)
			d := func(p, q uint8) bool { return int(p)-int(q) <= tol && int(q)-int(p) <= tol }
			if d(cr, wr) && d(cg, wg) && d(cb, wb) {
				n++
			}
		}
	}
	return n
}

// Plastik marks the selected tab with a bar in the selection colour along
// its top; Qt 4's Plastique does not.
func TestPlastikSelectedTabBar(t *testing.T) {
	for _, c := range []struct {
		name string
		bar  bool
	}{{"plastik", true}, {"plastique", false}} {
		lk := mustLook(t, c.name)
		img := paintengine2d.NewImage(110, 34)
		ctx := paintengine2d.NewContext(img)
		lk.DrawTab(ctx, paintengine2d.XYWH(2, 2, 100, 28), StateNone, "Tab", true)
		n := plastikCount(img, paintengine2d.XYWH(2, 2, 100, 4), lk.Palette().Selection, 12)
		if got := n > 20; got != c.bar {
			t.Fatalf("%s selected tab: %d selection pixels along the top, want a bar = %v", c.name, n, c.bar)
		}
	}
}

// Plastik's progress bar carries diagonal stripes, Plastique's vertical
// chunks: down one column of Plastique's bar the colour holds, down
// Plastik's it changes as the stripes lean.
func TestPlastikProgressStripesAndChunks(t *testing.T) {
	for _, c := range []struct {
		name     string
		vertical bool
	}{{"plastik", false}, {"plastique", true}} {
		lk := mustLook(t, c.name)
		img := paintengine2d.NewImage(210, 30)
		ctx := paintengine2d.NewContext(img)
		lk.DrawProgressBar(ctx, paintengine2d.XYWH(4, 4, 200, 20), StateNone, 1, false, 0)
		steady := 0
		for x := 20; x < 180; x++ {
			r0, g0, b0, _ := img.PremulAt(x, 10)
			r1, g1, b1, _ := img.PremulAt(x, 18)
			if r0 == r1 && g0 == g1 && b0 == b1 {
				steady++
			}
		}
		if got := steady > 150; got != c.vertical {
			t.Fatalf("%s progress: %d of 160 columns hold their colour, want vertical chunks = %v", c.name, steady, c.vertical)
		}
	}
}

// A focused text field takes the selection-coloured contour (Plastik's
// input focus highlight); an unfocused one keeps the grey contour.
func TestPlastikInputFocusHighlight(t *testing.T) {
	for _, n := range plastikPackNames {
		lk := mustLook(t, n)
		paint := func(st ControlState) int {
			img := paintengine2d.NewImage(200, 36)
			ctx := paintengine2d.NewContext(img)
			lk.DrawTextField(ctx, paintengine2d.XYWH(4, 4, 190, 28), st, "", "", 0, 0, 0, false, 0, nil)
			return plastikCount(img, paintengine2d.XYWH(4, 4, 190, 28), lk.Palette().Selection, 10)
		}
		if idle, focused := paint(StateNone), paint(StateFocused); idle > 0 || focused < 300 {
			t.Fatalf("%s field contour: %d selection pixels idle, %d focused", n, idle, focused)
		}
	}
}

// KDE 3 kept a list's selection when it lost focus and when its window
// went inactive.
func TestKDE3KeepsSelectionUnfocused(t *testing.T) {
	for _, n := range []string{"keramik", "plastik", "plastique"} {
		lk := mustLook(t, n)
		fill := func(st ControlState) paintengine2d.Color {
			img := paintengine2d.NewImage(180, 30)
			ctx := paintengine2d.NewContext(img)
			lk.DrawListRow(ctx, paintengine2d.XYWH(2, 2, 170, 24), st, "")
			r, g, b, _ := img.PremulAt(150, 14)
			return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
		}
		sel := fill(StateChecked)
		if !kdeClose(sel, lk.Palette().Selection, 2) {
			t.Fatalf("%s: selected row %s, want the selection %s", n, colorHexPadded(sel), colorHexPadded(lk.Palette().Selection))
		}
		for _, st := range []ControlState{StateChecked | StateInactive, StateChecked | StateBackdrop} {
			if got := fill(st); !kdeClose(got, sel, 2) {
				t.Fatalf("%s: selection with state %#x turns %s", n, uint32(st), colorHexPadded(got))
			}
		}
	}
}

func TestPlastikCloseButtonAgreesWithPaint(t *testing.T) {
	for _, n := range plastikPackNames {
		kdeCheckClose(t, n)
	}
}

func TestPlastikPaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range plastikPackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}

// KDE 3 labelled tool buttons in its tool bar font, a point smaller than
// the general font; Qt 4's Plastique used the application font.
func TestKDE3ToolBarFont(t *testing.T) {
	for _, c := range []struct {
		name    string
		smaller bool
	}{{"keramik", true}, {"plastik", true}, {"plastique", false}} {
		lk := mustLook(t, c.name)
		tool, body := ControlFontOf(lk, RoleTool), lk.Font()
		if got := tool.Height() < body.Height(); got != c.smaller {
			t.Fatalf("%s tool font %.1fpx against %.1fpx: want smaller = %v", c.name, tool.Height(), body.Height(), c.smaller)
		}
		if ControlFontOf(lk, RoleButton) != body || ControlFontOf(lk, RoleMenu) != body {
			t.Fatalf("%s: buttons and menus keep the general font", c.name)
		}
	}
}
