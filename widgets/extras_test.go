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
	g.Buttons()[2].MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 8)})
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
	r.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !r.Selected {
		t.Fatal("should select")
	}
	r.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	r.MouseRelease(widget.MouseEvent{Pos: paintengine2d.Pt(8, 16)})
	if !r.Selected {
		t.Fatal("radio stays selected")
	}
}

func TestComboBoxMeasuresToolbarHeight(t *testing.T) {
	c := NewComboBox([]string{"Inbox", "Sent"}, 0, nil)
	c.SetHost(&host{})
	sz := c.Measure(layout.Loose(400, 200))
	m := style.DarkLook().Metrics()
	want := style.ComboHeight(m)
	if sz.Y != want {
		t.Fatalf("combo height %v want ComboHeight %v (ComboH %v ControlH %v)", sz.Y, want, m.ComboH, m.ControlH)
	}
	if want >= m.ControlH {
		t.Fatalf("ComboHeight %v should be shorter than ControlH %v", want, m.ControlH)
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

func TestToolBarIconLabelItemsDoNotOverlap(t *testing.T) {
	tb := NewToolBar(
		ToolIconBtn(style.IconOpen, "Get Messages", nil),
		ToolIconBtn(style.IconNew, "Write", nil),
		ToolDivider(),
		ToolToggle("Cards", false, nil),
		ToolToggle("Classic", true, nil),
	)
	tb.SetHost(&host{})
	sz := tb.Measure(layout.Unbounded())
	tb.Arrange(paintengine2d.XYWH(0, 0, sz.X, 36))
	rects := tb.itemRects()
	if len(rects) != 5 {
		t.Fatalf("rects %d", len(rects))
	}
	for i := 1; i < len(rects); i++ {
		if tb.items[i] == nil || tb.items[i].Sep || tb.items[i-1] == nil || tb.items[i-1].Sep {
			continue
		}
		gap := rects[i].Min.X - rects[i-1].Max.X
		if gap < style.ToolItemGap-0.51 {
			t.Fatalf("items %d/%d overlap or cramped (gap=%v)", i-1, i, gap)
		}
	}
	img := paintengine2d.NewImage(int(sz.X)+4, 40)
	ctx := paintengine2d.NewContext(img)
	tb.Paint(ctx)
	// Gap between Get Messages and Write must stay clear of the next icon.
	x0 := int(rects[0].Max.X)
	x1 := int(rects[1].Min.X)
	if x1 <= x0 {
		t.Fatal("no gap to inspect")
	}
	br, bg, bb := styleRGB8(tb.Look().Palette().SurfaceAlt)
	ink := 0
	for y := 8; y < 28; y++ {
		for x := x0; x < x1; x++ {
			r, g, b, _ := img.PremulAt(x, y)
			if chanDelta(r, br)+chanDelta(g, bg)+chanDelta(b, bb) > 80 {
				ink++
			}
		}
	}
	if ink > 12 {
		t.Fatalf("label spilled into inter-tool gap (ink=%d)", ink)
	}
}

func TestToolBarMouseClickClearsFocusPaint(t *testing.T) {
	tb := NewToolBar(ToolIconBtn(style.IconCut, "Delete", nil), ToolText("Next", nil))
	h := &host{}
	tb.SetHost(h)
	tb.Arrange(paintengine2d.XYWH(0, 0, 280, 36))
	pt := tb.ItemCenter(0)
	tb.MousePress(widget.MouseEvent{Pos: pt, Button: platform.ButtonLeft})
	tb.MouseRelease(widget.MouseEvent{Pos: pt, Button: platform.ButtonLeft})
	if tb.keyNav {
		t.Fatal("mouse click must not leave keyboard focus chrome")
	}
	tb.KeyPress(widget.KeyEvent{Key: platform.KeyRight})
	if !tb.keyNav || tb.focus != 1 {
		t.Fatalf("keyboard focus keyNav=%v focus=%d", tb.keyNav, tb.focus)
	}
	tb.FocusLost()
	if tb.keyNav || tb.focus != -1 {
		t.Fatalf("FocusLost keyNav=%v focus=%d", tb.keyNav, tb.focus)
	}
}

func TestToolToggleChromeDistinct(t *testing.T) {
	action := NewToolBar(ToolText("Cards", nil))
	off := NewToolBar(ToolToggle("Cards", false, nil))
	on := NewToolBar(ToolToggle("Cards", true, nil))
	h := &host{}
	action.SetHost(h)
	off.SetHost(h)
	on.SetHost(h)
	box := paintengine2d.XYWH(0, 0, 120, 36)
	action.Arrange(box)
	off.Arrange(box)
	on.Arrange(box)
	pa, po, pn := rasterTool(action), rasterTool(off), rasterTool(on)
	if colorDiff(pa, po) < 40 {
		t.Fatal("toggle-off should not match a flat action button")
	}
	if colorDiff(po, pn) < 40 {
		t.Fatal("toggle-on should not match toggle-off")
	}
}

func rasterTool(tb *ToolBar) *paintengine2d.Image {
	img := paintengine2d.NewImage(120, 36)
	tb.Paint(paintengine2d.NewContext(img))
	return img
}

func colorDiff(a, b *paintengine2d.Image) int {
	diff := 0
	h, w := a.Height, a.Width
	if b.Height < h {
		h = b.Height
	}
	if b.Width < w {
		w = b.Width
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ar, ag, ab, aa := a.PremulAt(x, y)
			br, bg, bb, ba := b.PremulAt(x, y)
			if chanDelta(ar, br)+chanDelta(ag, bg)+chanDelta(ab, bb)+chanDelta(aa, ba) > 40 {
				diff++
			}
		}
	}
	return diff
}

func chanDelta(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func styleRGB8(c paintengine2d.Color) (r, g, b uint8) {
	return uint8(c.R*255 + 0.5), uint8(c.G*255 + 0.5), uint8(c.B*255 + 0.5)
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
	if sz.X < 80 || sz.Y < 80 {
		t.Fatalf("card %v", sz)
	}
}
