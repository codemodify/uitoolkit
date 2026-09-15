package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

var breezePackNames = []string{"breeze", "breeze-night"}

func TestBreezePacksRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "breeze", "KDE", 2014, map[string]string{"breeze": "Breeze", "breeze-night": "Breeze Dark"})
	// The dark pack replaces the legacy base-engine "breeze-night"; there is
	// no second dark Breeze under another name.
	if _, ok := kdePositions()["breeze-dark"]; ok {
		t.Fatal("no pack may be named breeze-dark: the dark pack is breeze-night")
	}
}

// The packs carry the Plasma 5.27 colour schemes.
func TestBreezeSchemes(t *testing.T) {
	for _, c := range []struct{ pack, window, text, view, sel string }{
		{"breeze", "#eff0f1", "#232629", "#ffffff", "#3daee9"},
		{"breeze-night", "#2a2e32", "#fcfcfc", "#1b1e20", "#3daee9"},
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

func TestBreezeStyleHintsAndScrollBars(t *testing.T) {
	for _, n := range breezePackNames {
		lk := mustLook(t, n)
		if LookHint(lk, HintDialogPrimaryFirst) != 1 || LookHint(lk, HintTabsCentered) != 0 || LookHint(lk, HintFormLabelsRight) != 1 {
			t.Fatalf("%s: want KDE's OK-before-Cancel order, left-aligned tabs and right-aligned form labels", n)
		}
		s := ScrollBarStyleOf(lk)
		if s.Arrows != ArrowsNone || !s.Overlay || s.Thickness != 12 {
			t.Fatalf("%s scroll bar %+v: want a 12px overlay bar without arrows", n, s)
		}
	}
}

// The scroll handle is thin at rest and widens (showing its groove) while
// the pointer is over the bar.
func TestBreezeScrollHandleWidensOnHover(t *testing.T) {
	lk := mustLook(t, "breeze")
	width := func(ss ScrollState) (int, int) {
		img := paintengine2d.NewImage(40, 200)
		ctx := paintengine2d.NewContext(img)
		bar := paintengine2d.XYWH(10, 10, 12, 180)
		parts := ScrollParts{Bar: bar, Track: bar, Thumb: paintengine2d.XYWH(10, 60, 12, 50)}
		DrawScrollBarParts(lk, ctx, parts, true, ss)
		w := func(y int) int {
			n := 0
			for x := 0; x < img.Width; x++ {
				if _, _, _, a := img.PremulAt(x, y); a > 40 {
					n++
				}
			}
			return n
		}
		return w(85), w(30) // across the handle, across the bare track
	}
	idle, idleTrack := width(ScrollState{})
	hot, hotTrack := width(ScrollState{Hovered: true, Hot: ScrollThumbPart})
	if idle == 0 || hot <= idle {
		t.Fatalf("handle %dpx at rest and %dpx hovered: want it to widen", idle, hot)
	}
	if idleTrack != 0 || hotTrack == 0 {
		t.Fatalf("groove %dpx at rest, %dpx hovered: want it shown only on hover", idleTrack, hotTrack)
	}
}

// Keyboard focus turns a field's outline to the focus colour.
func TestBreezeFocusOutline(t *testing.T) {
	lk := mustLook(t, "breeze")
	img := paintengine2d.NewImage(200, 40)
	ctx := paintengine2d.NewContext(img)
	b := paintengine2d.XYWH(4, 4, 190, 32)
	lk.DrawTextField(ctx, b, StateFocused, "", "", 0, 0, 0, false, 0, nil)
	// The outline sits one pixel in (a 1px margin, then the 1px line).
	if !nxNear(img, 100, 5, Hex("#3daee9")) {
		r, g, bl, _ := img.PremulAt(100, 5)
		t.Fatalf("focused field outline rgb(%d,%d,%d), want #3daee9", r, g, bl)
	}
}

func TestBreezeCloseButtonAgreesWithPaint(t *testing.T) {
	for _, n := range breezePackNames {
		kdeCheckClose(t, n)
	}
}

func TestBreezePaintsEveryControlInsideItsRect(t *testing.T) {
	for _, n := range breezePackNames {
		p, _ := LoadTheme(n)
		for _, scale := range []float32{1, 2} {
			lk := WithScale(p.Look(), scale).(*Classic)
			aquaExercise(t, fmt.Sprintf("%s@%gx", n, scale), lk)
			kdeExtras(t, fmt.Sprintf("%s@%gx", n, scale), lk)
		}
	}
}
