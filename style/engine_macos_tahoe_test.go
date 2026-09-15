package style

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var tahoePackNames = []string{"tahoe", "tahoe-night"}

func TestTahoePacksRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		if _, dup := pos[n]; dup {
			t.Fatalf("pack %q listed twice", n)
		}
		pos[n] = i
	}
	for n, w := range map[string]struct {
		label string
		fam   ThemeName
	}{"tahoe": {"macOS Tahoe", ThemeLight}, "tahoe-night": {"macOS Tahoe Dark", ThemeDark}} {
		p, ok := LoadTheme(n)
		if !ok {
			t.Fatalf("pack %q not registered", n)
		}
		if p.Tokens.Engine != "macos" || p.Lineage != "Mac OS" || p.Label != w.label || p.Year != 2025 || p.Tokens.Family != w.fam || p.Tokens.Params["era"] != 2 {
			t.Fatalf("%s: engine %q lineage %q label %q year %d family %q era %v", n, p.Tokens.Engine, p.Lineage, p.Label, p.Year, p.Tokens.Family, p.Tokens.Params["era"])
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		lk := p.Look()
		if _, ok := lk.Engine().(tahoeEngine); !ok || lk.Engine().ID() != "macos" {
			t.Fatalf("%s paints with %T (%s), want Tahoe's era of macos", n, lk.Engine(), lk.Engine().ID())
		}
	}
	if pos["bigsur"] > pos["tahoe"] {
		t.Fatal("Big Sur (2020) sorts after Tahoe (2025)")
	}
	for _, n := range macosPackNames {
		if _, ok := macLook(t, n, 1).Engine().(macosEngine); !ok {
			t.Fatalf("%s left its era", n)
		}
	}
}

// Both appearances have every key, and no resolved colour is the magenta
// sentinel of a missing one.
func TestTahoeColourTables(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for k := range tahoeLight {
		if _, ok := tahoeDark[k]; !ok {
			t.Errorf("tahoeDark lacks %q", k)
		}
	}
	for k := range tahoeDark {
		if _, ok := tahoeLight[k]; !ok {
			t.Errorf("tahoeLight lacks %q", k)
		}
	}
	for _, n := range tahoePackNames {
		c := tahoeColors(macLook(t, n, 1))
		winNoSentinel(t, n, reflect.ValueOf(*c))
	}
	// The 2025 system colours.
	for n, want := range map[string]string{"tahoe": "#0088ff", "tahoe-night": "#0091ff"} {
		if got := colorHexPadded(tahoeColors(macLook(t, n, 1)).accent); got != want {
			t.Errorf("%s: accent %s, want Blue %s", n, got, want)
		}
	}
}

func TestTahoePaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, tahoePackNames)
	winPaintsInsideRect(t, tahoePackNames)
	for _, n := range tahoePackNames {
		for _, sc := range []float32{1, 1.75, 2} {
			lk := macLook(t, n, sc)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, sc), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, sc), lk)
			for _, st := range []ControlState{StateNone, StateSidebar, StateSidebar | StateBackdrop, StateFocused} {
				img := paintengine2d.NewImage(int(260*sc), int(180*sc))
				b := paintengine2d.XYWH(20*sc, 20*sc, 220*sc, 140*sc)
				lk.DrawViewFrame(paintengine2d.NewContext(img), b, st)
				if k := winOutside(img, b); k > 0 {
					t.Errorf("%s@%gx view frame %#x: %d pixels outside", n, sc, uint32(st), k)
				}
			}
		}
	}
}

// The close button is the red light, first on the left; an inactive window
// greys it.
func TestTahoeCloseIsTheRedLightOnTheLeft(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range tahoePackNames {
		for _, sc := range []float32{1, 2} {
			lk := macLook(t, n, sc)
			b := paintengine2d.XYWH(4*sc, 4*sc, 300*sc, 180*sc)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Min.X || cr.Max.X > b.Min.X+b.Dx()*0.2 || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close rect %v not the left light of frame %v (top inset %v)", n, sc, cr, b, in.Top)
			}
			c := cr.Center()
			if r, g, bl, _ := winFrame(lk, b, WindowState{Active: true, CanClose: true}).PremulAt(int(c.X), int(c.Y)); int(r) < int(g)+40 || int(r) < int(bl)+40 {
				t.Fatalf("%s@%gx: close light centre is rgb(%d,%d,%d), want red", n, sc, r, g, bl)
			}
			if r, g, _, _ := winFrame(lk, b, WindowState{CanClose: true}).PremulAt(int(c.X), int(c.Y)); int(r) > int(g)+20 {
				t.Fatalf("%s@%gx: the inactive window's close light is still red", n, sc)
			}
		}
	}
}

// The Mac's rules hold in Tahoe.
func TestTahoeStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range tahoePackNames {
		lk := macLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 0 || LookHint(lk, HintTabsCentered) != 1 || LookHint(lk, HintFormLabelsRight) != 1 || LookHint(lk, HintMnemonics) != MnemonicsNever {
			t.Fatalf("%s: the Mac's dialog order, centred tabs, right-aligned labels and no mnemonics", n)
		}
		if s := ScrollBarStyleOf(lk); !s.Transient || !s.Overlay || s.Arrows != ArrowsNone {
			t.Fatalf("%s: overlay scrollers, got %+v", n, s)
		}
		if !ToolGroupsOf(lk) {
			t.Fatalf("%s: tool bar buttons share glass groups", n)
		}
	}
	if ToolGroupsOf(macLook(t, "bigsur", 1)) {
		t.Fatal("Big Sur's tool bar has no glass groups")
	}
}

// tahoePaint paints f into a w×h image on the window colour.
func tahoePaint(lk *Classic, w, h int, f func(ctx *paintengine2d.Context)) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(w), float32(h)), paintengine2d.Fill(tahoeColors(lk).win))
	f(ctx)
	return img
}

// Push buttons are flat grey faces rounded by about a quarter of their
// height, capsules from the large size up; the default button is the accent.
func TestTahoeButtons(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := macLook(t, "tahoe", 1)
	c := tahoeColors(lk)
	medium := paintengine2d.XYWH(10, 10, 120, 34) // a 28px face
	img := tahoePaint(lk, 140, 60, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, medium, StateNone, "") })
	face := c.flat(c.btn)
	if !winNear(img, 70, 30, face) {
		t.Errorf("the button face is %v, want the grey %s", pixelColor(img, 70, 30), colorHexPadded(face))
	}
	if !winNear(img, 13, 13, c.win) || !winNear(img, 20, 27, face) {
		t.Errorf("the medium button is not a 7px-rounded rectangle")
	}
	large := paintengine2d.XYWH(10, 10, 120, 42) // a 36px face: a capsule
	cap := tahoePaint(lk, 140, 60, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, large, StateNone, "") })
	if !winNear(cap, 17, 16, c.win) || !winNear(cap, 31, 31, face) {
		t.Errorf("the large button is not a capsule")
	}
	def := tahoePaint(lk, 140, 60, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, medium, StatePrimary, "") })
	if !winNear(def, 70, 27, c.accent) {
		t.Errorf("the default button is not the accent: %v", pixelColor(def, 70, 27))
	}
	// Plain again in an inactive window.
	if back := tahoePaint(lk, 140, 60, func(ctx *paintengine2d.Context) {
		lk.DrawButton(ctx, medium, StatePrimary|StateBackdrop, "")
	}); !winNear(back, 70, 27, face) {
		t.Errorf("the default button keeps the accent in an inactive window")
	}
}

// Glass: a tool bar group is a capsule lighter than the window with a rim;
// the floating sidebar a rounded panel inset from the view's edges; a knob
// held turns to glass.
func TestTahoeGlass(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range tahoePackNames {
		lk := macLook(t, n, 1)
		c := tahoeColors(lk)
		g := paintengine2d.XYWH(10, 6, 140, 40)
		img := tahoePaint(lk, 160, 52, func(ctx *paintengine2d.Context) { DrawToolGroupOf(lk, ctx, g, 3) })
		if !winNear(img, 80, 26, c.flat(c.glass)) {
			t.Errorf("%s: the group's glass is %v, want %s", n, pixelColor(img, 80, 26), colorHexPadded(c.flat(c.glass)))
		}
		if !winNear(img, 12, 8, c.win) {
			t.Errorf("%s: the group is not a capsule", n)
		}
		side := paintengine2d.XYWH(0, 0, 200, 300)
		sb := tahoePaint(lk, 200, 300, func(ctx *paintengine2d.Context) { lk.DrawViewFrame(ctx, side, StateSidebar) })
		if !winNear(sb, 2, 150, c.win) || !winNear(sb, 100, 150, c.flat(c.glass)) {
			t.Errorf("%s: the sidebar is not a panel inset from the view's edge: %v, %v", n, pixelColor(sb, 2, 150), pixelColor(sb, 100, 150))
		}
		if !winNear(sb, 8, 8, c.win) {
			t.Errorf("%s: the sidebar panel's corner is not round", n)
		}
		sw := paintengine2d.XYWH(10, 10, 120, 30)
		rest := tahoePaint(lk, 140, 50, func(ctx *paintengine2d.Context) { lk.DrawSwitch(ctx, sw, StateNone, true, "") })
		held := tahoePaint(lk, 140, 50, func(ctx *paintengine2d.Context) { lk.DrawSwitch(ctx, sw, StatePressed, true, "") })
		k := int(10 + 3 + 54 - 2 - 17) // the on knob's centre
		if !winNear(rest, k, 25, c.knob) || winNear(held, k, 25, c.knob) {
			t.Errorf("%s: the knob is not white at rest and glass while held: %v, %v", n, pixelColor(rest, k, 25), pixelColor(held, k, 25))
		}
	}
}

// Lists are rounded cards; the focused selection is the accent, inset and
// rounded; a sidebar's selection is grey with its label in the accent.
func TestTahoeSelections(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := macLook(t, "tahoe", 1)
	c := tahoeColors(lk)
	b := paintengine2d.XYWH(0, 0, 200, 32)
	row := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(200, 32)
		lk.DrawListRow(paintengine2d.NewContext(img), b, st, "")
		return img
	}
	on, off := row(StateChecked), row(StateChecked|StateInactive)
	if !winNear(on, 100, 16, c.sel) {
		t.Errorf("the focused selection is %v, want %s", pixelColor(on, 100, 16), colorHexPadded(c.sel))
	}
	if _, _, _, a := off.PremulAt(100, 16); a == 0 || winNear(off, 100, 16, c.sel) {
		t.Errorf("the unfocused selection is not the grey")
	}
	if _, _, _, a := on.PremulAt(2, 16); a != 0 {
		t.Errorf("the selection is not inset from the card's side")
	}
	if _, _, _, a := on.PremulAt(4, 1); a != 0 {
		t.Errorf("the selection's corner is square")
	}
	side := tahoePaint(lk, 200, 32, func(ctx *paintengine2d.Context) {
		ctx.DrawRect(b, paintengine2d.Fill(c.side))
		lk.DrawListRow(ctx, b, StateChecked|StateSidebar, "")
	})
	if !winNear(side, 100, 16, webOver(c.side, c.sideSel)) {
		t.Errorf("the sidebar selection is %v, want the grey row", pixelColor(side, 100, 16))
	}
	if fg := c.sideRow(lk, paintengine2d.NewContext(paintengine2d.NewImage(200, 32)), b, StateChecked|StateSidebar); fg != c.sideText {
		t.Errorf("the selected sidebar label is %s, want the accent %s", colorHexPadded(fg), colorHexPadded(c.sideText))
	}
	// A table's alternate rows are rounded stripes inset from the card.
	stripe := paintengine2d.NewImage(300, 32)
	for i, st := range []ControlState{StateFirst, StateNone, StateLast} {
		lk.DrawTableCell(paintengine2d.NewContext(stripe), paintengine2d.XYWH(float32(i)*100, 0, 100, 32), st|StateAlternate, "", AlignStart, nil)
	}
	if _, _, _, a := stripe.PremulAt(150, 16); a == 0 {
		t.Errorf("no stripe across the middle cell")
	}
	if _, _, _, a := stripe.PremulAt(1, 16); a != 0 {
		t.Errorf("the stripe reaches the card's side")
	}
}

// Text reads on every fill the engine paints it on.
func TestTahoeLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range tahoePackNames {
		c := tahoeColors(macLook(t, n, 1))
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.win, 4.5},
			{"text on content", c.text, c.content, 4.5},
			{"button label", c.btnText, c.flat(c.btn), 4.5},
			{"default label", c.onAccent, c.accent, 3},
			{"list selection", c.onAccent, c.sel, 4.5},
			{"unfocused selection", c.selOffText, webOver(c.content, c.selOff), 4.5},
			{"menu highlight", c.onAccent, c.menuHi, 3},
			{"menu text", c.text, c.flat(c.menu), 4.5},
			{"tooltip", c.tipText, c.flat(c.tip), 4.5},
			{"sidebar label", c.text, c.side, 4.5},
			{"selected sidebar label", c.sideText, webOver(c.side, c.sideSel), 3},
			{"secondary text", c.text2, c.content, 3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s %s on %s is %.2f:1 < %.1f", n, chk.what, colorHexPadded(chk.fg), colorHexPadded(chk.bg), r, chk.min)
			}
		}
	}
}

// Follow the desktop moves between the Tahoe twins.
func TestTahoeSchemeSiblings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := SchemeVariant("tahoe", SchemeDark); got != "tahoe-night" {
		t.Errorf("tahoe in the dark: %q", got)
	}
	if got := SchemeVariant("tahoe-night", SchemeLight); got != "tahoe" {
		t.Errorf("tahoe-night in the light: %q", got)
	}
}
