package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// A panel that *is* a window — a borderless toplevel drawing its chrome
// with the look's in-app window frame — needs its caption to do what a
// caption does. Without this the window has a title bar that cannot move
// it: the caption is paint.
func TestAWindowPanelsCaptionTakesAPress(t *testing.T) {
	pack, ok := style.LoadTheme("win95")
	if !ok {
		t.Skipf("the %q pack is not in this build", "win95")
	}
	h := &host{}
	p := NewPanel("", NewLabel("body"))
	p.Window = true
	p.Title = "Flip"
	p.SetLook(pack.Look())
	p.SetHost(h)
	p.Arrange(paintengine2d.XYWH(0, 0, 400, 300))

	bar := p.captionRect()
	if bar.Empty() {
		t.Fatal("an in-app window panel has no caption band")
	}
	moves := 0
	p.OnCaptionPress = func() bool { moves++; return true }

	mid := paintengine2d.Pt(bar.Dx()/2, bar.Dy()/2)
	if !p.MousePress(widget.MouseEvent{Pos: mid, Button: platform.ButtonLeft}) {
		t.Error("a press on the caption was not taken")
	}
	if moves != 1 {
		t.Errorf("the caption was pressed %d times, want 1", moves)
	}

	// Below the caption is the body, which is not the caption's to take.
	below := paintengine2d.Pt(bar.Dx()/2, bar.Max.Y+20)
	if p.MousePress(widget.MouseEvent{Pos: below, Button: platform.ButtonLeft}) {
		t.Error("a press below the caption was taken as a caption press")
	}
	if moves != 1 {
		t.Errorf("a press in the body moved the window (%d moves)", moves)
	}
}

// The close button is part of the caption band, and pressing it must
// close rather than move: a window you cannot close without dragging it
// is the bug this orders against.
func TestTheCloseButtonWinsOverTheCaption(t *testing.T) {
	pack, ok := style.LoadTheme("win95")
	if !ok {
		t.Skipf("the %q pack is not in this build", "win95")
	}
	closed, moves := 0, 0
	p := NewPanel("", NewLabel("body"))
	p.Window = true
	p.Title = "Flip"
	p.OnClose = func() { closed++ }
	p.SetLook(pack.Look())
	p.SetHost(&host{})
	p.Arrange(paintengine2d.XYWH(0, 0, 400, 300))
	p.OnCaptionPress = func() bool { moves++; return true }

	cr := p.closeRect()
	if cr.Empty() {
		t.Fatal("an in-app window panel with OnClose has no close button")
	}
	if !p.captionRect().Contains(cr.Center()) {
		t.Fatal("the close button is not in the caption band, so this proves nothing")
	}
	e := widget.MouseEvent{Pos: cr.Center(), Button: platform.ButtonLeft}
	p.MousePress(e)
	p.MouseRelease(e)
	if moves != 0 {
		t.Errorf("pressing the close button moved the window (%d moves)", moves)
	}
	if closed != 1 {
		t.Errorf("the close button closed %d times, want 1", closed)
	}
}
