package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestOxygenPackRegistered(t *testing.T) {
	kdeCheckPacks(t, "oxygen", "KDE", 2008, map[string]string{"oxygen": "Oxygen"})
}

func TestOxygenStyleHintsAndScrollBars(t *testing.T) {
	lk := mustLook(t, "oxygen")
	if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 1 {
		t.Fatal("Oxygen: want KDE's OK-before-Cancel order, left-aligned tabs and right-aligned form labels")
	}
	s := ScrollBarStyleOf(lk)
	if s.Arrows != ArrowsNone || s.Overlay || s.Thickness != 16 {
		t.Fatalf("Oxygen scroll bar %+v: want a 16px bar with no arrows", s)
	}
}

// The tone model lands on the shades Oxygen shows for its own colour scheme.
func TestOxygenTonesMatchTheScheme(t *testing.T) {
	h := oxTones{k: 1}
	win := Hex("#d6d2d0")
	for _, c := range []struct {
		name string
		got  paintengine2d.Color
		want string
	}{
		{"light", h.light(win), "#fdfdfc"},
		{"dark", h.dark(win), "#b1aba8"},
		{"mid", h.mid(win), "#c3bdba"},
		{"shadow", h.shadow(win), "#3d3b3a"},
		{"top", h.top(win), "#e2dfde"},
		{"bottom", h.bottom(win), "#c1bbb7"},
	} {
		if !kdeClose(c.got, Hex(c.want), 8) {
			t.Fatalf("%s of the Oxygen window colour is %s, want about %s", c.name, colorHexPadded(c.got), c.want)
		}
	}
	// The contrast param stretches the moves.
	if l1, l2 := oxLstar(oxTones{k: 1}.dark(win)), oxLstar(oxTones{k: 1.4}.dark(win)); l2 >= l1 {
		t.Fatalf("a higher contrast must darken the dark shade (%v → %v)", l1, l2)
	}
}

// The window background is lighter at the top than below the gradient and
// brightest under the radial light at the top centre.
func TestOxygenWindowBackground(t *testing.T) {
	lk := mustLook(t, "oxygen")
	img := paintengine2d.NewImage(800, 500)
	ctx := paintengine2d.NewContext(img)
	lk.DrawWindowBackground(ctx, paintengine2d.XYWH(0, 0, 800, 500))
	lum := func(x, y int) int {
		r, g, b, _ := img.PremulAt(x, y)
		return int(r) + int(g) + int(b)
	}
	if top, bottom := lum(40, 70), lum(40, 450); top <= bottom+15 {
		t.Fatalf("gradient: top %d not lighter than bottom %d", top, bottom)
	}
	if centre, side := lum(400, 4), lum(40, 4); centre <= side {
		t.Fatalf("radial light: top centre %d not lighter than the top left %d", centre, side)
	}
	if a, b := lum(40, 400), lum(40, 480); a != b {
		t.Fatalf("below the 300px gradient the window is flat (%d vs %d)", a, b)
	}
}

// Keyboard focus shows as the blue glow around a field.
func TestOxygenFocusGlow(t *testing.T) {
	lk := mustLook(t, "oxygen")
	paint := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(200, 40)
		ctx := paintengine2d.NewContext(img)
		lk.DrawTextField(ctx, paintengine2d.XYWH(4, 4, 190, 30), st, "", "", 0, 0, 0, false, 0, nil)
		return img
	}
	idle, focused := paint(StateNone), paint(StateFocused)
	blue := 0
	for y := 4; y < 34; y++ {
		for x := 4; x < 194; x++ {
			r, _, b, _ := focused.PremulAt(x, y)
			r0, _, b0, _ := idle.PremulAt(x, y)
			if int(b)-int(r) > 40 && int(b0)-int(r0) < 20 {
				blue++
			}
		}
	}
	if blue < 100 {
		t.Fatalf("focused field shows %d glow pixels, want a blue ring", blue)
	}
}

func TestOxygenCloseButtonAgreesWithPaint(t *testing.T) {
	kdeCheckClose(t, "oxygen")
}

func TestOxygenPaintsEveryControlInsideItsRect(t *testing.T) {
	p, _ := LoadTheme("oxygen")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		aquaExercise(t, fmt.Sprintf("oxygen@%gx", scale), lk)
		kdeExtras(t, fmt.Sprintf("oxygen@%gx", scale), lk)
	}
}

// kdeClose reports whether a and b are within tol (0–255) per channel.
func kdeClose(a, b paintengine2d.Color, tol int) bool {
	ar, ag, ab, _ := a.Premul8()
	br, bg, bb, _ := b.Premul8()
	d := func(p, q uint8) bool { return int(p)-int(q) <= tol && int(q)-int(p) <= tol }
	return d(ar, br) && d(ag, bg) && d(ab, bb)
}
