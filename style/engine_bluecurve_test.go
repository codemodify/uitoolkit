package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestBluecurvePackRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	p, ok := LoadTheme("bluecurve")
	if !ok {
		t.Fatal("bluecurve not registered")
	}
	if p.Tokens.Engine != "bluecurve" || p.Label != "Bluecurve" || p.Lineage != "Red Hat" || p.Year != 2002 || p.Summary == "" {
		t.Fatalf("bluecurve: engine %q label %q lineage %q year %d", p.Tokens.Engine, p.Label, p.Lineage, p.Year)
	}
	if lk := p.Look(); lk.Engine().ID() != "bluecurve" {
		t.Fatalf("bluecurve look paints with %q", lk.Engine().ID())
	}
	pos := map[string]int{}
	for i, n := range AllBuiltinThemeNames() {
		pos[n] = i
	}
	if pos["bluecurve"] > pos["clearlooks"] {
		t.Fatal("Bluecurve (2002) sorts after Clearlooks (2005)")
	}
}

// The grey table and spot colours of Bluecurve's gtkrc colours.
func TestBluecurveShades(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := blColors(clLook(t, "bluecurve", 1))
	for i, want := range []string{"#f5f5f5", "#dedede", "#cecece", "#c4c4c4", "#b1b1b1", "#999999", "#5c5c5c", "#2f2f2f"} {
		if !hexNear(c.g[i], want, 1) {
			t.Errorf("gray%d = %s, want %s", i, colorHexPadded(c.g[i]), want)
		}
	}
	for i, want := range []string{"#98b2ed", "#4468b8", "#3b4c71"} {
		got := [...]paintengine2d.Color{c.s1, c.s2, c.s3}[i]
		if !hexNear(got, want, 1) {
			t.Errorf("spot%d = %s, want %s", i+1, colorHexPadded(got), want)
		}
	}
}

func TestBluecurvePaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, scale := range []float32{1, 2} {
		lk := clLook(t, "bluecurve", scale)
		aquaExercise(t, "bluecurve", lk)
		img := paintengine2d.NewImage(64, 64)
		ctx := paintengine2d.NewContext(img)
		for _, r := range []paintengine2d.Rect{{}, paintengine2d.XYWH(0, 0, 1, 1), paintengine2d.XYWH(0, 0, 5, 40), paintengine2d.XYWH(0, 0, 40, 5)} {
			for _, st := range lunaStates {
				lunaCalls(lk, ctx, r, st)
			}
			lunaStateless(lk, ctx, r)
		}
	}
}

// Square corners: a button's corner pixel is its #5c5c5c outline, and the
// check box's tick is the blue spot colour.
func TestBluecurveShapes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := clLook(t, "bluecurve", 1)
	c := blColors(lk)
	img := paintengine2d.NewImage(100, 40)
	ctx := paintengine2d.NewContext(img)
	lk.DrawButton(ctx, paintengine2d.XYWH(4, 4, 90, 30), StateNone, "")
	if !nxNear(img, 4, 4, c.g[6]) || !nxNear(img, 93, 33, c.g[6]) {
		t.Fatal("Bluecurve buttons are square in a grey-6 outline")
	}
	if !nxNear(img, 5, 5, c.white) || !nxNear(img, 92, 32, c.g[2]) {
		t.Fatal("Bluecurve buttons are lit white top-left and grey-2 bottom-right")
	}
	img = paintengine2d.NewImage(20, 20)
	ctx = paintengine2d.NewContext(img)
	lk.Engine().CheckIndicator(lk, ctx, paintengine2d.XYWH(3, 3, 13, 13), StateNone, true)
	if countNear(img, c.spot) < 6 {
		t.Fatal("the Bluecurve tick is not the spot colour")
	}
	// Menu items highlight in the blue gradient with white text.
	if ContrastRatio(lk.Engine().MenuTextColor(lk, true), c.menuGrad[1].Color) < 3 {
		t.Fatal("hot menu text does not read on the menu gradient")
	}
}

func TestBluecurveStyleHintAndCloseRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := clLook(t, "bluecurve", 1)
	if LookHint(lk, HintDialogPrimaryFirst) != 0 {
		t.Fatal("GTK dialogs put the default button last")
	}
	b := paintengine2d.XYWH(10, 10, 280, 160)
	cr := lk.WindowCloseRect(b)
	if cr.Empty() || cr.Max.X > b.Max.X || cr.Max.Y > b.Min.Y+lk.WindowFrameInsets().Top {
		t.Fatalf("close %v outside the title bar of %v", cr, b)
	}
	img := paintengine2d.NewImage(310, 190)
	ctx := paintengine2d.NewContext(img)
	lk.DrawWindowFrame(ctx, b, "Dialog", WindowState{Active: true, CanClose: true})
	if r, g, bl, _ := img.PremulAt(int(cr.Center().X), int(cr.Center().Y)); r < 200 || g < 200 || bl < 200 {
		t.Fatalf("close centre rgb(%d,%d,%d), want the white ×", r, g, bl)
	}
}
