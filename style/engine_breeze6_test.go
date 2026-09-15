package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var breeze6PackNames = []string{"breeze6", "breeze6-night"}

func TestBreeze6PacksRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	kdeCheckPacks(t, "breeze", "KDE", 2024, map[string]string{"breeze6": "Breeze (Plasma 6)", "breeze6-night": "Breeze Dark (Plasma 6)"})
	pos := kdePositions()
	if pos["breeze"] > pos["breeze6"] {
		t.Fatal("Plasma 5's Breeze (2014) sorts after Plasma 6's (2024)")
	}
	for _, n := range breeze6PackNames {
		lk := mustLook(t, n)
		if _, ok := lk.Engine().(breeze6Engine); !ok {
			t.Fatalf("%s paints with %T, want the Plasma 6 era", n, lk.Engine())
		}
	}
	// The Plasma 5 packs keep the Plasma 5 engine.
	for _, n := range breezePackNames {
		if _, ok := mustLook(t, n).Engine().(breezeEngine); !ok {
			t.Fatalf("%s left Plasma 5", n)
		}
	}
}

// The dark pack is Plasma 6.4's darker Breeze Dark; the light one Breeze
// Light, unchanged since 5.27.
func TestBreeze6Schemes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, c := range []struct{ pack, window, text, view, sel string }{
		{"breeze6", "#eff0f1", "#232629", "#ffffff", "#3daee9"},
		{"breeze6-night", "#202326", "#fcfcfc", "#141618", "#3daee9"},
	} {
		p := mustLook(t, c.pack).Palette()
		for _, v := range []struct {
			name string
			got  paintengine2d.Color
			want string
		}{{"window", p.Background, c.window}, {"text", p.Text, c.text}, {"view", p.Field, c.view}, {"selection", p.Selection, c.sel}} {
			if colorHexPadded(v.got) != v.want {
				t.Fatalf("%s %s = %s, want %s", c.pack, v.name, colorHexPadded(v.got), v.want)
			}
		}
	}
}

// Plasma 6's mixes: outlines at the 0.2 frame contrast, pressed and checked
// at the highlight's 0.3, values at its 0.7.
func TestBreeze6Mixes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for n, want := range map[string]struct{ outline, btnOutline, pressed string }{
		"breeze6":       {"#c6c8c9", "#d1d1d2", "#c3e5f6"},
		"breeze6-night": {"#4c4e51", "#535659", "#2f5368"},
	} {
		c := breezeColors(mustLook(t, n))
		for _, v := range []struct {
			name string
			got  paintengine2d.Color
			want string
		}{{"frame outline", c.outline, want.outline}, {"button outline", c.btnOutline, want.btnOutline}, {"pressed button", c.btnDown, want.pressed}} {
			if !accentSame(v.got, Hex(v.want)) {
				t.Errorf("%s: %s %s, want %s", n, v.name, colorHexPadded(v.got), v.want)
			}
		}
		if c.frameR != 4.5 {
			t.Errorf("%s: frame radius %v for a 1px pen, want 4.5 (5px frames)", n, c.frameR)
		}
	}
	if c := breezeColors(mustLook(t, "breeze")); c.frameR != 2.5 || !accentSame(c.outline, Mix(c.win, c.text, 0.25)) {
		t.Errorf("Plasma 5's Breeze changed: radius %v, outline %s", c.frameR, colorHexPadded(c.outline))
	}
}

func TestBreeze6CloseButtonAgreesWithPaint(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, n := range breeze6PackNames {
		kdeCheckClose(t, n)
	}
}

func TestBreeze6PaintsEveryControlInsideItsRect(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	winPaintsEverything(t, breeze6PackNames)
	winPaintsInsideRect(t, breeze6PackNames)
	for _, n := range breeze6PackNames {
		for _, scale := range []float32{1, 1.75, 2} {
			lk := WithScale(mustLook(t, n), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}

// b6Paint paints f into a w×h image on the look's window colour.
func b6Paint(lk *Classic, w, h int, f func(ctx *paintengine2d.Context)) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(w), float32(h)), paintengine2d.Fill(lk.Palette().Background))
	f(ctx)
	return img
}

// Buttons round their 5px corners where Plasma 5 rounded 3, and keyboard
// focus lays a band of the highlight at 0.3 round the face.
func TestBreeze6ButtonCornersAndFocusBand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "breeze6")
	c := breezeColors(lk)
	b := paintengine2d.XYWH(10, 10, 120, 36)
	plain := b6Paint(lk, 140, 56, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, StateNone, "") })
	// The face is 2px in (12,12); the pixels by its corner stay the window
	// colour, where Plasma 5's 3px frame would have covered them.
	if !winNear(plain, 12, 13, c.win) || !winNear(plain, 13, 12, c.win) {
		t.Errorf("the button's corner is not 5px round: %v", pixelColor(plain, 12, 13))
	}
	if !winNear(plain, 70, 12, c.btnOutline) {
		t.Errorf("the button's top edge is not its 0.2 outline: %v", pixelColor(plain, 70, 12))
	}
	focused := b6Paint(lk, 140, 56, func(ctx *paintengine2d.Context) { lk.DrawButton(ctx, b, StateFocused, "") })
	if want := webOver(c.win, c.hl.WithAlpha(0.3)); !winNear(focused, 70, 11, want) {
		t.Errorf("no focus band in the margin: %v, want %s", pixelColor(focused, 70, 11), colorHexPadded(want))
	}
	if !winNear(focused, 70, 12, c.hl) {
		t.Errorf("a focused button's outline is not the highlight")
	}
	if winNear(plain, 70, 11, c.hl.WithAlpha(0.3)) {
		t.Errorf("an unfocused button carries the band")
	}
}

// Views are frameless; a selected row is a rounded box in an outline, inset
// from the row's sides.
func TestBreeze6FramelessViewsAndRoundedSelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "breeze6")
	c := breezeColors(lk)
	if !lk.ViewFrameInsets().Zero() {
		t.Fatalf("views keep a frame: %+v", lk.ViewFrameInsets())
	}
	vb := paintengine2d.XYWH(10, 10, 100, 60)
	view := b6Paint(lk, 120, 80, func(ctx *paintengine2d.Context) { lk.DrawViewFrame(ctx, vb, StateFocused) })
	if !winNear(view, 10, 40, c.base) || !winNear(view, 60, 10, c.base) {
		t.Errorf("a view's edge is not its fill: %v", pixelColor(view, 10, 40))
	}
	row := paintengine2d.XYWH(0, 0, 200, 28)
	img := paintengine2d.NewImage(200, 28)
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(row, paintengine2d.Fill(c.base))
	lk.DrawListRow(ctx, row, StateChecked, "")
	if !winNear(img, 100, 14, c.hl) {
		t.Errorf("the selection is not the highlight: %v", pixelColor(img, 100, 14))
	}
	if !winNear(img, 1, 14, c.base) {
		t.Errorf("the selection is not inset from the view's side")
	}
	if !winNear(img, 2, 1, c.base) {
		t.Errorf("the selection's corner is square")
	}
	if winNear(img, 100, 1, c.hl) {
		t.Errorf("the selection has no outline: %v", pixelColor(img, 100, 1))
	}
}

// A pressed menu item is the solid focus colour; a hovered one the focus
// at 0.3.
func TestBreeze6PressedMenuItem(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := mustLook(t, "breeze6")
	c := breezeColors(lk)
	ch := MenuChromeFor(lk)
	paint := func(st ControlState) *paintengine2d.Image {
		return b6Paint(lk, 240, 40, func(ctx *paintengine2d.Context) {
			lk.DrawMenuItem(ctx, paintengine2d.XYWH(10+ch.PadL, 5, 200, 30), st, MenuRow{Label: ""})
		})
	}
	if !winNear(paint(StatePressed|StateHovered), 120, 20, c.focus) {
		t.Errorf("a pressed menu item is not the solid focus colour")
	}
	if hot := paint(StateHovered); winNear(hot, 120, 20, c.focus) || winNear(hot, 120, 20, c.win) {
		t.Errorf("a hovered menu item is not the focus at 0.3: %v", pixelColor(hot, 120, 20))
	}
}

// Follow the desktop moves between the Plasma 6 twins.
func TestBreeze6SchemeSiblings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := SchemeVariant("breeze6", SchemeDark); got != "breeze6-night" {
		t.Errorf("breeze6 in the dark: %q", got)
	}
	if got := SchemeVariant("breeze6-night", SchemeLight); got != "breeze6" {
		t.Errorf("breeze6-night in the light: %q", got)
	}
}
