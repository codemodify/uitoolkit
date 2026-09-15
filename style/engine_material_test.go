package style

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var materialPackNames = []string{"material", "material-night", "material3", "material3-night"}

func TestMaterialPacksRegisteredInYearOrder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	want := map[string]struct {
		label string
		year  int
		fam   ThemeName
		gen   float32
	}{
		"material":        {"Material", 2014, ThemeLight, 2},
		"material-night":  {"Material Dark", 2014, ThemeDark, 2},
		"material3":       {"Material 3", 2021, ThemeLight, 3},
		"material3-night": {"Material 3 Dark", 2021, ThemeDark, 3},
	}
	for _, n := range materialPackNames {
		if _, ok := pos[n]; !ok {
			t.Fatalf("pack %q not registered", n)
		}
		p, _ := LoadTheme(n)
		w := want[n]
		if p.Tokens.Engine != "material" || p.Lineage != "Google" || p.Label != w.label || p.Year != w.year {
			t.Fatalf("%s: engine %q lineage %q label %q year %d", n, p.Tokens.Engine, p.Lineage, p.Label, p.Year)
		}
		if p.Tokens.Family != w.fam || p.Tokens.Params["gen"] != w.gen {
			t.Fatalf("%s: family %q gen %v", n, p.Tokens.Family, p.Tokens.Params["gen"])
		}
		if strings.TrimSpace(p.Summary) == "" || strings.Contains(p.Summary, "\n") {
			t.Fatalf("%s: summary must be one line: %q", n, p.Summary)
		}
		if lk := p.Look(); lk.Engine().ID() != "material" {
			t.Fatalf("%s look paints with %q", n, lk.Engine().ID())
		}
		if p.Tokens.Bevel != BevelSoftShadow {
			t.Fatalf("%s bevel %q", n, p.Tokens.Bevel)
		}
	}
	if pos["material"] > pos["material3"] || pos["material-night"] > pos["material3-night"] {
		t.Fatalf("packs not ordered by year: %v", pos)
	}
	// The legacy era packs are replaced, not duplicated.
	count := map[string]int{}
	for _, n := range AllBuiltinThemeNames() {
		count[n]++
	}
	for _, n := range []string{"material", "material-night"} {
		if count[n] != 1 {
			t.Fatalf("%s listed %d times", n, count[n])
		}
	}
	if p, _ := LoadTheme("material-dark"); p.Name != "material-night" {
		t.Fatalf("material-dark alias resolves to %q", p.Name)
	}
}

func materialLook(t *testing.T, name string, scale float32) *Classic {
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

// Every Draw* paints every state without panicking, degenerate rects too.
func TestMaterialPaintsEveryControl(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range materialPackNames {
		for _, sc := range []float32{1, 2} {
			lk := materialLook(t, n, sc)
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

// Every control paints strictly inside its rect, at 1x and 2x.
func TestMaterialPaintsInsideRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range materialPackNames {
		for _, sc := range []float32{1, 2} {
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, sc), materialLook(t, n, sc))
		}
	}
}

// The close button is the icon button at the right of the dialog's title
// band, and it is painted where hit-testing looks for it.
func TestMaterialCloseRectAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range materialPackNames {
		for _, sc := range []float32{1, 2} {
			lk := materialLook(t, n, sc)
			b := paintengine2d.XYWH(10*sc, 10*sc, 300*sc, 180*sc)
			cr := lk.WindowCloseRect(b)
			in := lk.WindowFrameInsets()
			if cr.Empty() || cr.Min.X < b.Max.X-b.Dx()*0.25 || cr.Max.X > b.Max.X || cr.Min.Y < b.Min.Y || cr.Max.Y > b.Min.Y+in.Top {
				t.Fatalf("%s@%gx: close %v not at the right of the title band of %v (top %v)", n, sc, cr, b, in.Top)
			}
			if cr.Dx() != cr.Dy() {
				t.Fatalf("%s@%gx: close %vx%v is not square", n, sc, cr.Dx(), cr.Dy())
			}
			rest := winFrame(lk, b, WindowState{Active: true, CanClose: true})
			hot := winFrame(lk, b, WindowState{Active: true, CanClose: true, CloseHot: true})
			// The × crosses at the centre; the hover layer fills the button.
			ctr := cr.Center()
			if !winDiffers(rest, winFrame(lk, b, WindowState{Active: true}), int(ctr.X), int(ctr.Y)) {
				t.Fatalf("%s@%gx: no close glyph at the close rect's centre", n, sc)
			}
			x, y := int(cr.Min.X+cr.Dx()*0.25), int(ctr.Y)
			if !winDiffers(rest, hot, x, y) {
				t.Fatalf("%s@%gx: hovering the close button changes nothing at (%d,%d)", n, sc, x, y)
			}
		}
	}
}

func TestMaterialStyleHints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range materialPackNames {
		lk := materialLook(t, n, 1)
		if LookHint(lk, HintDialogPrimaryFirst) != 0 {
			t.Fatalf("%s: Material dialogs put the confirming action last", n)
		}
		if LookHint(lk, HintHoverFadeMs) != 150 {
			t.Fatalf("%s: Material state changes fade over about 150ms", n)
		}
		if LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintDefaultPulseMs) != 0 {
			t.Fatalf("%s: tabs start at the left, nothing pulses", n)
		}
		s := ScrollBarStyleOf(lk)
		if !s.Transient || !s.Overlay || s.Arrows != ArrowsNone {
			t.Fatalf("%s: scroll indicators come and go without arrows: %+v", n, s)
		}
		if SpinBoxStyleOf(lk) != (SpinBoxStyle{Inside: true}) {
			t.Fatalf("%s: the step buttons sit inside the field", n)
		}
	}
}

// Text reads on every fill the engine paints it on.
func TestMaterialLabelsReadable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range materialPackNames {
		lk := materialLook(t, n, 1)
		c := mdColors(lk)
		for _, chk := range []struct {
			what   string
			fg, bg paintengine2d.Color
			min    float64
		}{
			{"text on surface", c.text, c.surface, 4.5},
			{"text on background", c.text, c.bg, 4.5},
			{"medium emphasis", c.text2, c.surface, 4.5},
			{"label on primary", c.onPrimary, c.primary, 4.5},
			{"text button label", c.primary, c.surface, 4.5},
			{"tonal label", c.onSecondaryC, c.secondaryC, 4.5},
			{"check mark", c.ctlMark, c.ctlOn, 3},
			{"check on surface", c.ctlOn, c.surface, 3},
			{"unchecked box", c.ctlOff, c.surface, 3},
			{"selected row", c.rowSelText, c.rowSel, 4.5},
			{"menu text", c.text, c.menu, 4.5},
			{"dialog text", c.text, c.dialog, 4.5},
			{"tooltip", c.tipFg, Mix(c.bg, c.tipBg, c.tipBg.A), 4.5},
			{"field outline", c.fieldLine, c.surface, 2.3},
		} {
			if r := ContrastRatio(chk.fg, chk.bg); r < chk.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", n, chk.what, r, chk.min)
			}
		}
	}
}

// Material 3's roles follow the seed colour: the baseline seed gives the
// baseline primary, another seed another hue.
func TestMaterial3SeedDrivesTheScheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := materialLook(t, "material3", 1)
	if d := mdDeltaE(mdColors(lk).primary, Hex("#6750a4")); d > 2 {
		t.Fatalf("baseline primary is %s (ΔE %.1f)", colorHexPadded(mdColors(lk).primary), d)
	}
	p, _ := LoadTheme("material3")
	tok := p.Tokens
	tok.Extra = map[string]paintengine2d.Color{"seed": Hex("#006e1c")}
	green := newClassic("material3", tok.Palette, DefaultMetrics(), CornersRound, IconSetClassic, IconSizeMedium, tok)
	g := mdColors(green)
	if h := mdToLab(g.primary).h; h < 100 || h > 160 {
		t.Fatalf("a green seed gives primary %s (hue %.0f)", colorHexPadded(g.primary), h)
	}
	if l := mdToLab(g.primary).l; l < 38 || l > 42 {
		t.Fatalf("primary is tone 40, got L* %.1f", l)
	}
	if l := mdToLab(g.secondaryC).l; l < 88 || l > 92 {
		t.Fatalf("secondary container is tone 90, got L* %.1f", l)
	}
}

// An outlined field with a label: once it has text the label floats into
// a gap in the top edge.
func TestMaterialFloatingLabel(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range []string{"material", "material3"} {
		lk := materialLook(t, n, 1)
		h := FieldHeight(lk.Metrics())
		render := func(text, placeholder string) *paintengine2d.Image {
			img := paintengine2d.NewImage(220, int(h)+8)
			ctx := paintengine2d.NewContext(img)
			lk.DrawTextField(ctx, paintengine2d.XYWH(4, 4, 200, h), StateNone, text, placeholder, 0, 0, 0, false, 0, nil)
			return img
		}
		floated := render("Ada", "Name")
		resting := render("", "Name")
		c := mdColors(lk)
		box := c.fieldBox(lk, paintengine2d.XYWH(4, 4, 200, h), true)
		if box.Min.Y <= 4 {
			t.Fatalf("%s: a labelled field keeps no room for its floated label", n)
		}
		// The top edge is broken just past the floated label (the notch's
		// trailing margin), and whole there while the label rests inside.
		pad := lk.Metrics().FieldPad
		gapX := int(box.Min.X + pad + c.small.Advance("Name") + lk.S(2))
		edgeX := int(box.Max.X - lk.S(20))
		y := int(box.Min.Y)
		if _, _, _, a := floated.PremulAt(edgeX, y); a < 128 {
			t.Fatalf("%s: the top edge is missing at x=%d", n, edgeX)
		}
		if !winDiffers(floated, resting, gapX, y) {
			t.Fatalf("%s: the top edge is not broken for the floated label at x=%d", n, gapX)
		}
		diff := 0
		for yy := int(box.Min.Y) + 2; yy < int(box.Max.Y)-2; yy++ {
			for xx := int(box.Min.X) + 2; xx < int(box.Max.X)-2; xx++ {
				if winDiffers(floated, resting, xx, yy) {
					diff++
				}
			}
		}
		if diff < 20 {
			t.Fatalf("%s: the resting label and the text paint alike (%d pixels differ)", n, diff)
		}
	}
}
