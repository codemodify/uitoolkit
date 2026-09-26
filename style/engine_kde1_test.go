package style

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestKDE1PackRegisteredInYearOrder(t *testing.T) {
	kdeCheckPacks(t, "kde1", "KDE", 1998, map[string]string{"kde1": "KDE 1"})
	pos := kdePositions()
	// KDE 1 (1998) opens the KDE lane, before KDE 2 (2000) and Keramik.
	for _, later := range []string{"kde2", "keramik", "oxygen"} {
		if p, ok := pos[later]; ok && pos["kde1"] > p {
			t.Errorf("KDE 1 (1998) sorts after %s: %v %v", later, pos["kde1"], p)
		}
	}
}

// The pack carries KDE 1's own colour scheme, the one KDE shipped again in
// KDE 2 as "KDEOne" — and not the warm "Desert red" of the KDE 1.0
// screenshot everyone reproduces.
func TestKDE1CarriesTheKDE1Scheme(t *testing.T) {
	lk := mustLook(t, "kde1")
	p := lk.Palette()
	for _, c := range []struct {
		name string
		got  paintengine2d.Color
		want string
	}{
		{"background", p.Background, "#c0c0c0"},
		{"base", p.Field, "#ffffff"},
		{"selection", p.Selection, "#000080"},
		{"selected text", p.TextOnAccent, "#ffffff"},
		{"caption", lk.X("caption", paintengine2d.Color{}), "#000080"},
		{"caption blend", lk.X("caption2", paintengine2d.Color{}), "#000000"},
		{"inactive caption", lk.X("captionOff", paintengine2d.Color{}), "#808080"},
		{"inactive blend", lk.X("captionOff2", paintengine2d.Color{}), "#c0c0c0"},
		// Qt's shades, not Windows' #dfdfdf and #808080.
		{"inner light", lk.X("light", paintengine2d.Color{}), "#d8d8d8"},
		{"inner shadow", lk.X("shadow", paintengine2d.Color{}), "#606060"},
	} {
		if got := colorHexPadded(c.got); got != c.want {
			t.Errorf("%s = %s, want %s", c.name, got, c.want)
		}
	}
	if got := colorHexPadded(Hex("#d6ceb9")); got == colorHexPadded(p.Background) {
		t.Error("the pack took the Desert red scheme of the KDE 1.0 screenshot")
	}
}

// KDE 1 kept Qt 1's Windows 95 widgets: an arrow at each end of a scroll
// bar, OK before Cancel the KDE way.
func TestKDE1KeepsTheWindowsWidgets(t *testing.T) {
	lk := mustLook(t, "kde1")
	if s := ScrollBarStyleOf(lk); s.Arrows != ArrowsEnds || s.Overlay {
		t.Fatalf("KDE 1 scroll bar %+v: want one arrow at each end", s)
	}
	if LookHint(lk, HintDialogPrimaryFirst) != 1 {
		t.Error("KDE 1: want OK before Cancel")
	}
	if got := lk.Engine().ID(); got != "kde1" {
		t.Fatalf("engine %q", got)
	}
	// The in-app frame is kwm's, not Windows', so its insets differ from
	// the win95 engine's.
	w95 := mustLook(t, "win95")
	if lk.WindowFrameInsets() == w95.WindowFrameInsets() {
		t.Error("KDE 1's window frame is Windows 95's")
	}
}

// kwm shaded a title bar across its width: navy at the left, black at the
// right. The two ends must differ, and the left end must be the caption
// colour.
func TestKDE1TitleBarBlendsAcrossItsWidth(t *testing.T) {
	lk := mustLook(t, "kde1")
	b := paintengine2d.XYWH(0, 0, 360, 200)
	img := paintengine2d.NewImage(int(b.Max.X), int(b.Max.Y))
	lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "", WindowState{Active: true})
	in := lk.WindowFrameInsets()
	y := int(in.Top * 0.5)
	at := func(x int) paintengine2d.Color {
		r, g, bl, a := img.PremulAt(x, y)
		return paintengine2d.RGBA(float32(r)/255, float32(g)/255, float32(bl)/255, float32(a)/255)
	}
	left, right := at(20), at(int(b.Max.X)-40)
	if !kdeClose(left, Hex("#000080"), 24) {
		t.Errorf("the left of the strip is %s, want about #000080", colorHexPadded(left))
	}
	if Luma(left) <= Luma(right) {
		t.Errorf("the strip does not darken to the right: %s then %s", colorHexPadded(left), colorHexPadded(right))
	}
	if !kdeClose(right, Hex("#000000"), 24) {
		t.Errorf("the right of the strip is %s, want about black", colorHexPadded(right))
	}
}

// The strip is sunk into the frame, so the row above it is darker than the
// window colour and the row below it lighter.
func TestKDE1TitleStripIsSunkIntoTheFrame(t *testing.T) {
	lk := mustLook(t, "kde1")
	b := paintengine2d.XYWH(0, 0, 360, 200)
	img := paintengine2d.NewImage(int(b.Max.X), int(b.Max.Y))
	lk.DrawWindowFrame(paintengine2d.NewContext(img), b, "", WindowState{Active: true})
	lum := func(y int) int {
		r, g, bl, _ := img.PremulAt(100, y)
		return int(r) + int(g) + int(bl)
	}
	face := lum(2)
	fr, h, sep := int(kw1Border(lk)), int(kw1CaptionH(lk)), int(kw1Sep(lk))
	top, bottom := lum(fr+sep), lum(fr+h-sep-1)
	if top >= face {
		t.Errorf("the strip's top line (%d) is not darker than the frame (%d)", top, face)
	}
	if bottom <= face {
		t.Errorf("the strip's bottom line (%d) is not lighter than the frame (%d)", bottom, face)
	}
}

// KToolBar's drag handle: diagonal lines at the left of the bar, in both
// the highlight and the shadow.
func TestKDE1ToolBarCarriesTheDragHandle(t *testing.T) {
	lk := mustLook(t, "kde1")
	img := paintengine2d.NewImage(200, 40)
	lk.DrawToolBar(paintengine2d.NewContext(img), paintengine2d.XYWH(0, 0, 200, 40))
	c := w95colors(lk)
	// Below the bar's own etched top line, so only the hatch is counted.
	grip := paintengine2d.XYWH(0, 4, 14, 34)
	rest := paintengine2d.XYWH(40, 4, 14, 34)
	for _, c := range []struct {
		name string
		col  paintengine2d.Color
	}{{"highlight", c.hi}, {"shadow", c.shadow}} {
		n, m := countNearIn(img, grip, c.col), countNearIn(img, rest, c.col)
		if n < 12 {
			t.Errorf("the handle has %d %s pixels, want a hatch", n, c.name)
		}
		if m > 0 {
			t.Errorf("the handle bleeds into the bar: %d %s pixels past it", m, c.name)
		}
	}
}

func TestKDE1CloseButtonAgreesWithPaint(t *testing.T) {
	kdeCheckClose(t, "kde1")
}

func TestKDE1PaintsEveryControlInsideItsRect(t *testing.T) {
	p, _ := LoadTheme("kde1")
	for _, scale := range []float32{1, 2} {
		lk := WithScale(p.Look(), scale).(*Classic)
		aquaExercise(t, fmt.Sprintf("kde1@%gx", scale), lk)
		kdeExtras(t, fmt.Sprintf("kde1@%gx", scale), lk)
	}
}
