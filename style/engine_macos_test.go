package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var macosPackNames = []string{"yosemite", "bigsur", "bigsur-night"}

func macLook(t *testing.T, name string, scale float32) *Classic {
	t.Helper()
	p, ok := LoadTheme(name)
	if !ok {
		t.Fatalf("pack %q not found", name)
	}
	var lk LookAndFeel = p.Look()
	if scale != 1 {
		lk = WithScale(lk, scale)
	}
	return lk.(*Classic)
}

func TestMacOSPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	count := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
		count[n]++
	}
	want := map[string]struct {
		label string
		year  int
		fam   ThemeName
		era   float32
	}{
		"yosemite":     {"OS X Yosemite", 2014, ThemeLight, 0},
		"bigsur":       {"macOS Big Sur", 2020, ThemeLight, 1},
		"bigsur-night": {"macOS Big Sur Dark", 2020, ThemeDark, 1},
	}
	for _, n := range macosPackNames {
		if count[n] != 1 {
			t.Fatalf("pack %q listed %d times", n, count[n])
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "macos" || p.Lineage != "Mac OS" || p.Label != w.label || p.Year != w.year || p.Tokens.Family != w.fam {
			t.Fatalf("%s: engine %q lineage %q label %q year %d family %q", n, p.Tokens.Engine, p.Lineage, p.Label, p.Year, p.Tokens.Family)
		}
		if p.Tokens.Params["era"] != w.era {
			t.Fatalf("%s: era %v", n, p.Tokens.Params["era"])
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != "macos" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	// Aqua (2001) comes before Yosemite (2014), Yosemite before Big Sur.
	if pos["aqua"] > pos["yosemite"] || pos["yosemite"] > pos["bigsur"] || pos["bigsur"] > pos["bigsur-night"] {
		t.Fatalf("packs not ordered by year: %v", pos)
	}
}

// Every scheme has every key, and every key the engine reads is in them.
func TestMacOSColourTables(t *testing.T) {
	for name, sc := range map[string]macScheme{"bigsur": macBigSur, "bigsur-night": macBigSurDark} {
		for k := range macYosemite {
			if _, ok := sc[k]; !ok {
				t.Errorf("%s lacks %q", name, k)
			}
		}
		for k := range sc {
			if _, ok := macYosemite[k]; !ok {
				t.Errorf("%s has %q, which macYosemite lacks", name, k)
			}
		}
	}
	for _, n := range macosPackNames {
		c := macColors(macLook(t, n, 1))
		for _, col := range []paintengine2d.Color{c.win, c.accent, c.header, c.washPress, c.lights[2], c.glyph} {
			if colorHexPadded(col) == "#ff00ff" {
				t.Fatalf("%s: a colour key is missing", n)
			}
		}
	}
}

func TestMacOSPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		for _, sc := range []float32{1, 2} {
			lk := macLook(t, n, sc)
			img := paintengine2d.NewImage(int(420*sc), int(200*sc))
			ctx := paintengine2d.NewContext(img)
			for _, st := range lunaStates {
				for _, r := range []paintengine2d.Rect{
					paintengine2d.XYWH(4*sc, 4*sc, 120*sc, 28*sc),
					paintengine2d.XYWH(4*sc, 4*sc, 17*sc, 100*sc),
					paintengine2d.XYWH(4.5*sc, 3.25*sc, 96.5*sc, 22.75*sc),
				} {
					lunaCalls(lk, ctx, r, st)
				}
			}
			lunaStateless(lk, ctx, paintengine2d.XYWH(4*sc, 4*sc, 400*sc, 180*sc))
			for _, r := range []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(2, 2, 3, 3),
				paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5)} {
				for _, st := range lunaStates {
					lunaCalls(lk, ctx, r, st)
				}
				lunaStateless(lk, ctx, r)
			}
			if ctx.SaveCount() != 0 {
				t.Fatalf("%s@%gx left %d saved states", n, sc, ctx.SaveCount())
			}
		}
	}
}

func TestMacOSPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		for _, sc := range []float32{1, 2} {
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, sc), macLook(t, n, sc))
		}
	}
}

// The close button is the red traffic light, first on the left of the
// title bar, and it is painted where hit-testing looks for it.
func TestMacOSCloseIsTheRedLightOnTheLeft(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		for _, sc := range []float32{1, 2} {
			lk := macLook(t, n, sc)
			b := paintengine2d.XYWH(4*sc, 4*sc, 300*sc, 180*sc)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close rect %v not the left light of frame %v (top inset %v)", n, sc, cr, b, in.Top)
			}
			img := winFrame(lk, b, WindowState{Active: true, CanClose: true})
			c := cr.Center()
			r, g, bl, _ := img.PremulAt(int(c.X), int(c.Y))
			if int(r) < int(g)+40 || int(r) < int(bl)+40 {
				t.Fatalf("%s@%gx: close light centre is rgb(%d,%d,%d), want red", n, sc, r, g, bl)
			}
			// An inactive window greys its lights.
			off := winFrame(lk, b, WindowState{CanClose: true})
			if r, g, _, _ := off.PremulAt(int(c.X), int(c.Y)); int(r) > int(g)+20 {
				t.Fatalf("%s@%gx: the inactive window's close light is still red", n, sc)
			}
		}
	}
}

// Mac rules: the default button last, centred segmented tabs, labels right
// of their fields' left edge, no pulsing default button since Yosemite.
func TestMacOSStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		lk := macLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 0 {
			t.Fatalf("%s: Mac dialogs put the default button last", n)
		}
		if LookHint(lk, HintTabsCentered) != 1 || LookHint(lk, HintFormLabelsRight) != 1 {
			t.Fatalf("%s: centred tabs and right-aligned form labels", n)
		}
		if LookHint(lk, HintDefaultPulseMs) != 0 {
			t.Fatalf("%s: the default button stopped pulsing with Yosemite", n)
		}
		s := ScrollBarStyleOf(lk)
		if !s.Transient || !s.Overlay || s.Arrows != ArrowsNone {
			t.Fatalf("%s: overlay scrollers, got %+v", n, s)
		}
		if SpinBoxStyleOf(lk) != (SpinBoxStyle{}) {
			t.Fatalf("%s: the stepper stands beside the field", n)
		}
	}
}

// The overlay scroller is a thin knob at rest and a wider one on a track
// under the pointer.
func TestMacOSOverlayScrollers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := macLook(t, "yosemite", 1)
	view := paintengine2d.XYWH(0, 0, 60, 200)
	parts := ScrollGeometry(lk, view, true, 800, 200, 100, false)
	ink := func(st ScrollState) (n, w int) {
		img := paintengine2d.NewImage(60, 200)
		DrawScrollBarParts(lk, paintengine2d.NewContext(img), parts, true, st)
		y := int(parts.Thumb.Center().Y)
		for x := 0; x < 60; x++ {
			if _, _, _, a := img.PremulAt(x, y); a > 40 {
				w++
			}
		}
		for i := 3; i < len(img.Pix); i += 4 {
			if img.Pix[i] > 0 {
				n++
			}
		}
		return n, w
	}
	idleN, idleW := ink(ScrollState{})
	hotN, hotW := ink(ScrollState{Hovered: true, Hot: ScrollThumbPart})
	if idleW == 0 || hotW <= idleW || hotN <= idleN*2 {
		t.Fatalf("idle knob %dpx wide (%d px inked), hovered %dpx (%d px): want a wider knob on a track", idleW, idleN, hotW, hotN)
	}
}

// List selections are the accent in a focused list and grey otherwise;
// Big Sur insets and rounds them.
func TestMacOSRowSelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		lk := macLook(t, n, 1)
		c := macColors(lk)
		b := paintengine2d.XYWH(0, 0, 200, 24)
		row := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(200, 24)
			lk.DrawListRow(paintengine2d.NewContext(img), b, st, "")
			return img
		}
		on, off := row(StateChecked), row(StateChecked|StateInactive)
		if !nxNear(on, 100, 12, c.sel) || !nxNear(off, 100, 12, c.selOff) {
			t.Fatalf("%s: focused selection must be the accent, unfocused grey", n)
		}
		_, _, _, a := on.PremulAt(2, 12)
		if c.bigSur && a != 0 {
			t.Fatalf("%s: Big Sur's selection is inset from the view's side", n)
		}
		if !c.bigSur && a == 0 {
			t.Fatalf("%s: Yosemite's selection spans the row", n)
		}
	}
}

// A table row's selection is one box across its cells: the middle cells
// are filled edge to edge, the ends rounded and inset on Big Sur.
func TestMacOSTableSelectionJoinsCells(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := macLook(t, "bigsur", 1)
	c := macColors(lk)
	img := paintengine2d.NewImage(300, 28)
	ctx := paintengine2d.NewContext(img)
	for i, st := range []ControlState{StateFirst, StateNone, StateLast} {
		lk.DrawTableCell(ctx, paintengine2d.XYWH(float32(i)*100, 0, 100, 28), st|StateChecked, "", AlignStart, nil)
	}
	for _, x := range []int{99, 100, 199, 200} {
		if !nxNear(img, x, 14, c.sel) {
			t.Fatalf("the selection breaks at the cell seam x=%d", x)
		}
	}
	for _, x := range []int{1, 298} {
		if _, _, _, a := img.PremulAt(x, 14); a != 0 {
			t.Fatalf("the row box reaches the view's side at x=%d", x)
		}
	}
}

// Keyboard focus is the halo: a focused push button paints a ring of the
// focus colour around its face, inside its rect.
func TestMacOSFocusHalo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		lk := macLook(t, n, 1)
		c := macColors(lk)
		b := paintengine2d.XYWH(0, 0, 100, 30)
		img := paintengine2d.NewImage(100, 30)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(b, paintengine2d.Fill(c.win))
		lk.DrawButton(ctx, b, StateFocused, "")
		want := Mix(c.win, c.focus, c.focus.A)
		if !nxNear(img, 50, 1, want) && !nxNear(img, 50, 2, want) {
			r, g, bl, _ := img.PremulAt(50, 1)
			t.Fatalf("%s: no halo above the focused button: rgb(%d,%d,%d)", n, r, g, bl)
		}
	}
}

// Text reads on every fill the engine paints it on.
func TestMacOSLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range macosPackNames {
		c := macColors(macLook(t, n, 1))
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.win, 4.5},
			{"text on content", c.text, c.content, 4.5},
			{"button label", c.btnText, Mix(c.btnStops[0].Color, c.btnStops[1].Color, 0.5), 4.5},
			// White on Yosemite's measured default-button blue is 2.95:1.
			{"default label", c.onAccent, Mix(c.accentStops[0].Color, c.accentStops[1].Color, 0.5), 2.9},
			{"list selection", c.onAccent, c.sel, 4.5},
			{"unfocused selection", c.selOffText, c.selOff, 4.5},
			// The vibrant highlight is the selection lightened (3.8:1).
			{"menu highlight", c.onAccent, c.menuHi, 3.5},
			{"menu text", c.text, Mix(c.win, c.menu, c.menu.A), 4.5},
			{"tooltip", c.tipText, Mix(c.win, c.tip, c.tip.A), 4.5},
			{"title", c.titleText, c.title, 4.5},
			{"secondary text", c.text2, c.content, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}
