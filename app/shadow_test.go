package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// Menus float on the look's drop shadow: the window paints it just outside
// the popup, and taking the popup down repaints that band.
func TestPopupDropsLookShadow(t *testing.T) {
	for _, lk := range []*style.Classic{style.LightLook(), style.DarkLook()} {
		t.Run(lk.Name(), func(t *testing.T) { testPopupShadow(t, lk) })
	}
}

func testPopupShadow(t *testing.T, lk *style.Classic) {
	a := New(Options{Look: lk, Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 320, Height: 240, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel(""))
	a.PumpOnce()
	before := w.Capture()
	pop := widgets.ShowContextMenu(w.Content(), paintengine2d.Pt(40, 40), widgets.Item("Open", nil), widgets.Item("Close", nil))
	a.PumpOnce()
	if pop == nil || w.Popup() == nil {
		t.Fatal("no popup")
	}
	out := style.PopupShadowOf(w.Look(), style.PopupMenu)
	if out.Bottom < 4 {
		t.Fatalf("light look menu shadow reach %+v, want a soft shadow", out)
	}
	b := pop.Bounds()
	x, y := int((b.Min.X+b.Max.X)/2), int(b.Max.Y+2)
	with := w.Capture()
	lum := func(img *paintengine2d.Image, x, y int) float32 {
		r, g, b, _ := img.At(x, y).RGBA()
		return (0.2126*float32(r) + 0.7152*float32(g) + 0.0722*float32(b)) / 65535
	}
	if got, bg := lum(with, x, y), lum(before, x, y); got >= bg-0.02 {
		t.Fatalf("below the menu luma %.3f, background %.3f: no shadow", got, bg)
	}
	w.DismissPopup()
	a.PumpOnce()
	w.frame()
	after := w.surf.Buffer()
	if got, bg := lum(after, x, y), lum(before, x, y); got < bg-0.01 {
		t.Fatalf("shadow left behind after dismiss: %.3f vs %.3f", got, bg)
	}
}
