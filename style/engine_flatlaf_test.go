package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var flatlafPackNames = []string{"flatlaf", "flatlaf-night", "flatlaf-darcula"}

func flatLook(t *testing.T, name string, scale float32) *Classic {
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

func TestFlatLafPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	count := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
		count[n]++
	}
	want := map[string]struct {
		label string
		fam   ThemeName
	}{
		"flatlaf":         {"FlatLaf Light", ThemeLight},
		"flatlaf-night":   {"FlatLaf Dark", ThemeDark},
		"flatlaf-darcula": {"Darcula", ThemeDark},
	}
	for _, n := range flatlafPackNames {
		if count[n] != 1 {
			t.Fatalf("pack %q listed %d times", n, count[n])
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "flatlaf" || p.Lineage != "Java" || p.Label != w.label || p.Year != 2019 || p.Tokens.Family != w.fam {
			t.Fatalf("%s: engine %q lineage %q label %q year %d family %q", n, p.Tokens.Engine, p.Lineage, p.Label, p.Year, p.Tokens.Family)
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if p.Tokens.Bevel != BevelNone {
			t.Fatalf("%s: bevel %q", n, p.Tokens.Bevel)
		}
		if lk := p.Look(); lk.Engine().ID() != "flatlaf" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
	}
	// Registered packs sort by year: FlatLaf (2019) after Material (2014)
	// and before Big Sur (2020).
	if pos["material"] > pos["flatlaf"] || pos["flatlaf-darcula"] > pos["bigsur"] {
		t.Fatalf("packs not ordered by year: %v", pos)
	}
	if p, _ := LoadTheme("flatlaf-dark"); p.Name != "flatlaf-night" {
		t.Fatalf("flatlaf-dark alias resolves to %q", p.Name)
	}
}

// Every scheme has every key the engine reads.
func TestFlatLafColourTables(t *testing.T) {
	for k := range flatLight {
		if _, ok := flatDark[k]; !ok {
			t.Errorf("flatDark lacks %q", k)
		}
	}
	for k := range flatDark {
		if _, ok := flatLight[k]; !ok {
			t.Errorf("flatLight lacks %q", k)
		}
	}
	for _, n := range flatlafPackNames {
		c := flatColors(flatLook(t, n, 1))
		if colorHexPadded(c.accent) == "#ff00ff" || colorHexPadded(c.grip) == "#ff00ff" {
			t.Fatalf("%s: a colour key is missing", n)
		}
	}
}

func TestFlatLafPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range flatlafPackNames {
		for _, sc := range []float32{1, 2} {
			lk := flatLook(t, n, sc)
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

func TestFlatLafPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range flatlafPackNames {
		for _, sc := range []float32{1, 2} {
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, sc), flatLook(t, n, sc))
		}
	}
}

// The close button fills the title bar at the right and turns red under
// the pointer, where hit-testing looks for it.
func TestFlatLafCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range flatlafPackNames {
		for _, sc := range []float32{1, 2} {
			lk := flatLook(t, n, sc)
			b := paintengine2d.XYWH(10*sc, 10*sc, 300*sc, 180*sc)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Max.X > b.Max.X || cr.Min.X < b.Max.X-b.Dx()*0.3 || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close %v not at the right of the title bar of %v (top %v)", n, sc, cr, b, in.Top)
			}
			img := winFrame(lk, b, WindowState{Active: true, CanClose: true, CloseHot: true})
			x, y := int(cr.Min.X+cr.Dx()*0.2), int(cr.Min.Y+cr.Dy()*0.3)
			r, g, bl, _ := img.PremulAt(x, y)
			if int(r) < int(g)+60 || int(r) < int(bl)+60 {
				t.Fatalf("%s@%gx: the hot close button at (%d,%d) is rgb(%d,%d,%d), not red", n, sc, x, y, r, g, bl)
			}
			ctr := cr.Center()
			rest := winFrame(lk, b, WindowState{Active: true, CanClose: true})
			if !winDiffers(rest, winFrame(lk, b, WindowState{Active: true}), int(ctr.X), int(ctr.Y)) {
				t.Fatalf("%s@%gx: no close glyph at the close rect's centre", n, sc)
			}
		}
	}
}

// Swing's option panes put OK first; FlatLaf switches states at once.
func TestFlatLafStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range flatlafPackNames {
		lk := flatLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 {
			t.Fatalf("%s: FlatLaf dialogs put OK first on Windows and Linux", n)
		}
		if LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 0 || LookHint(lk, HintHoverFadeMs) != 0 {
			t.Fatalf("%s: left tabs and labels, no fades", n)
		}
		s := ScrollBarStyleOf(lk)
		if s.Overlay || s.Transient || s.Arrows != ArrowsNone || s.Thickness != 10 {
			t.Fatalf("%s: a 10px arrow-less gutter bar, got %+v", n, s)
		}
	}
}

// The focus border is 2px of accent: Light and Dark draw the focused
// border and the inner focus line; Darcula a 2px ring outside the border.
func TestFlatLafFocusBorder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range flatlafPackNames {
		lk := flatLook(t, n, 1)
		c := flatColors(lk)
		b := paintengine2d.XYWH(4, 4, 100, 30)
		render := func(st ControlState) *paintengine2d.Image {
			img := paintengine2d.NewImage(108, 38)
			ctx := paintengine2d.NewContext(img)
			ctx.DrawRect(paintengine2d.XYWH(0, 0, 108, 38), paintengine2d.Fill(c.bg))
			lk.DrawButton(ctx, b, st, "")
			return img
		}
		img := render(StateFocused)
		// Sample the middle of the left edge, where the corner arc is out of
		// the way.
		y := 19
		near := func(x int, want paintengine2d.Color) bool { return nxNear(img, x, y, want) }
		if c.ring > 0 {
			if !near(4, c.focus) || !near(5, c.focus) || !near(6, c.btnFocusBorder) {
				t.Fatalf("%s: want a 2px ring then the focused border", n)
			}
			if nxNear(render(StateNone), 4, y, c.focus) {
				t.Fatalf("%s: the ring shows without focus", n)
			}
			continue
		}
		if !near(4, c.btnFocusBorder) || !near(5, c.focus) {
			t.Fatalf("%s: want the focused border then the inner focus line", n)
		}
	}
}

// Light marks a selected check box with an accent border and tick on
// white; Dark and Darcula with a grey tick.
func TestFlatLafCheckMarks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	light := flatColors(flatLook(t, "flatlaf", 1))
	if colorHexPadded(light.chkMark) != "#4e9de7" || colorHexPadded(light.chkSel) != "#ffffff" || colorHexPadded(light.chkSelBorder) != "#4e9de7" {
		t.Fatalf("Light check: mark %s fill %s border %s", colorHexPadded(light.chkMark), colorHexPadded(light.chkSel), colorHexPadded(light.chkSelBorder))
	}
	dark := flatColors(flatLook(t, "flatlaf-night", 1))
	if colorHexPadded(dark.chkMark) != "#c7c7c7" {
		t.Fatalf("Dark check mark %s", colorHexPadded(dark.chkMark))
	}
	if d := flatColors(flatLook(t, "flatlaf-darcula", 1)); !d.triangles || d.ring != 2 || d.radioDot != 5 {
		t.Fatalf("Darcula: triangles %v ring %v dot %v", d.triangles, d.ring, d.radioDot)
	}
}

// Text reads on every fill the engine paints it on.
func TestFlatLafLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range flatlafPackNames {
		c := flatColors(flatLook(t, n, 1))
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on window", c.text, c.bg, 4.5},
			{"text on field", c.text, c.field, 4.5},
			{"button label", c.text, c.btn, 4.5},
			{"default label", c.defText, c.def, 4.5},
			// FlatDarkLaf's own #eeeeee on #4b6eaf is 4.36:1.
			{"selection", c.selText, c.sel, 4},
			{"unfocused selection", c.selOffText, c.selOff, 4.5},
			{"menu text", c.text, c.menu, 4.5},
			{"tooltip", c.tipText, c.tip, 4.5},
			{"disabled text", c.textDis, c.bg, 2.5},
			{"check mark", c.chkMark, c.chkSel, 2.5},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}
