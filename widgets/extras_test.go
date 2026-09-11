package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func TestProgressBarClamp(t *testing.T) {
	p := NewProgressBar(1.5)
	if p.Value != 1 {
		t.Fatalf("clamp high %v", p.Value)
	}
	p.SetValue(-2)
	if p.Value != 0 {
		t.Fatalf("clamp low %v", p.Value)
	}
	p.SetBusy(0.4)
	if !p.Indeterminate || p.Phase != 0.4 {
		t.Fatalf("busy %v %v", p.Indeterminate, p.Phase)
	}
	sz := p.Measure(layout.Loose(200, 40))
	if sz.Y < 8 || sz.X < 40 {
		t.Fatalf("size %v", sz)
	}
}

func TestRadioGroupExclusive(t *testing.T) {
	n := -1
	g := NewRadioGroup([]string{"A", "B", "C"}, 0, func(i int) { n = i })
	g.SetHost(&host{})
	sz := g.Measure(layout.Loose(200, 200))
	g.Arrange(paintengine2d.XYWH(0, 0, 200, sz.Y))
	if !g.Buttons()[0].Selected || g.Selected() != 0 {
		t.Fatal("initial")
	}
	g.Buttons()[2].MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
	if g.Selected() != 2 || n != 2 {
		t.Fatalf("sel=%d n=%d", g.Selected(), n)
	}
	if g.Buttons()[0].Selected || !g.Buttons()[2].Selected {
		t.Fatal("exclusive")
	}
	g.Buttons()[2].KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if g.Selected() != 1 {
		t.Fatalf("up %d", g.Selected())
	}
}

func TestRadioStandalone(t *testing.T) {
	r := NewRadio("Solo", false, nil)
	r.SetHost(&host{})
	r.Arrange(paintengine2d.XYWH(0, 0, 120, 32))
	r.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !r.Selected {
		t.Fatal("should select")
	}
	r.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !r.Selected {
		t.Fatal("radio stays selected")
	}
}

func TestComboBoxSelect(t *testing.T) {
	n := -2
	c := NewComboBox([]string{"One", "Two", "Three"}, 0, func(i int) { n = i })
	c.SetHost(&host{})
	c.Arrange(paintengine2d.XYWH(0, 0, 180, 34))
	if c.Text() != "One" {
		t.Fatalf("text %q", c.Text())
	}
	c.Select(2)
	if c.Text() != "Three" || n != 2 {
		t.Fatalf("sel %q n=%d", c.Text(), n)
	}
	c.KeyPress(widget.KeyEvent{Key: platform.KeyHome})
	if c.Selected != 0 {
		t.Fatalf("home %d", c.Selected)
	}
	// no PopupHost on test host — Open should fail closed
	c.Open()
	if c.Opened() {
		t.Fatal("open without host")
	}
}

func TestToolBarClickAndKeys(t *testing.T) {
	n := 0
	tb := NewToolBar(
		ToolIconBtn(style.IconNew, "", func() { n++ }),
		ToolDivider(),
		ToolText("Go", func() { n += 10 }),
		ToolToggle("Snap", false, nil),
	)
	tb.SetHost(&host{})
	tb.Arrange(paintengine2d.XYWH(0, 0, 400, 36))
	rects := tb.itemRects()
	if len(rects) < 4 {
		t.Fatal("rects")
	}
	pt := paintengine2d.Pt((rects[0].Min.X+rects[0].Max.X)*0.5, (rects[0].Min.Y+rects[0].Max.Y)*0.5)
	tb.MousePress(widget.MouseEvent{Pos: pt, Button: platform.ButtonLeft})
	tb.MouseRelease(widget.MouseEvent{Pos: pt, Button: platform.ButtonLeft})
	if n != 1 {
		t.Fatalf("icon click %d", n)
	}
	tb.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	tb.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if tb.focus != 3 {
		t.Fatalf("focus %d", tb.focus)
	}
	tb.KeyPress(widget.KeyEvent{Key: platform.KeySpace})
	if !tb.Items()[3].Down {
		t.Fatal("toggle")
	}
	tb.Hover(2)
	if tb.hover != 2 {
		t.Fatalf("hover %d", tb.hover)
	}
}

func TestTitleBarMeasure(t *testing.T) {
	tb := NewTitleBar("Gallery", "subtitle")
	sz := tb.Measure(layout.Loose(400, 80))
	if sz.Y < 36 {
		t.Fatalf("height %v", sz)
	}
	tb.SetTitle("Other")
	if tb.Title != "Other" {
		t.Fatal(tb.Title)
	}
}

func TestMessageBoxCard(t *testing.T) {
	mb := NewMessageBox(MessageBoxOptions{
		Title:   "Warn",
		Message: "Something happened.",
		Kind:    MessageWarning,
		Buttons: ButtonsYesNoCancel,
	})
	if mb.Overlay() == nil || mb.Overlay().Card == nil {
		t.Fatal("overlay")
	}
	if MessageError.Icon() != style.IconError {
		t.Fatal("icon")
	}
	if MessageQuestion.Title() != "Question" {
		t.Fatal("title")
	}
}

func TestDialogCardStillBuilds(t *testing.T) {
	ok := NewButton("OK", nil)
	card := DialogCard("About", "Body text", ok)
	if card == nil {
		t.Fatal("card")
	}
	sz := card.Measure(layout.Loose(400, 300))
	if sz.X < 140 || sz.Y < 80 {
		t.Fatalf("card %v", sz)
	}
}
