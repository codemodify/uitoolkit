package widgets

import (
	"strings"
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

func TestLabelHonorsNewlinesAndWraps(t *testing.T) {
	hard := NewLabel("one\ntwo\nthree")
	sz := hard.Measure(layout.Unbounded())
	one := NewLabel("one").Measure(layout.Unbounded())
	if sz.Y < one.Y*2.5 {
		t.Fatalf("newline height %v want > %v", sz.Y, one.Y*2.5)
	}
	lines := hard.VisualLines(400)
	if len(lines) != 3 || lines[0].Text != "one" || lines[2].Text != "three" {
		t.Fatalf("hard lines %+v", lines)
	}

	long := "There are no accounts, want to add one? Type the IMAP/SMTP password in the form until a secret store exists and this sentence must wrap."
	body := NewLabel(long).WithWrap(180)
	ws := body.Measure(layout.Loose(180, 800))
	if ws.X > 182 {
		t.Fatalf("wrapped width %v", ws.X)
	}
	if ws.Y <= one.Y+4 {
		t.Fatalf("expected wrap height, got %v", ws.Y)
	}
	wl := body.VisualLines(180)
	if len(wl) < 3 {
		t.Fatalf("expected several wrap lines, got %+v", wl)
	}
	f := body.font()
	for _, ln := range wl {
		if strings.Contains(ln.Text, "\n") {
			t.Fatalf("newline leaked into visual line %q", ln.Text)
		}
		if adv := f.Advance(ln.Text); adv > 181 {
			t.Fatalf("line overflows %q adv=%v", ln.Text, adv)
		}
	}
}

func TestMessageBoxBodyWrapsAndHonorsNewlines(t *testing.T) {
	msg := "There are no accounts, want to add one?\n\nType the IMAP/SMTP password in the form. It is saved in mail.json (mode 0600) until a secret store exists."
	mb := NewMessageBox(MessageBoxOptions{
		Title: "Mail", Message: msg, Kind: MessageQuestion, Buttons: ButtonsYesNo,
	})
	ov := mb.Overlay()
	if ov == nil {
		t.Fatal("overlay")
	}
	ov.Arrange(paintengine2d.XYWH(0, 0, 480, 400))
	var body *Label
	widget.Walk(ov, func(c widget.Component) {
		if l, ok := c.(*Label); ok && strings.Contains(l.Text, "mail.json") {
			body = l
		}
	})
	if body == nil || !body.Wrap {
		t.Fatal("body label")
	}
	bw := body.Bounds().Dx()
	if bw < 80 {
		t.Fatalf("body width %v", body.Bounds())
	}
	lines := body.VisualLines(0)
	if len(lines) < 4 {
		t.Fatalf("expected wrap + blank line, got %d %+v", len(lines), lines)
	}
	var blank bool
	f := body.font()
	joined := ""
	for _, ln := range lines {
		if ln.Text == "" {
			blank = true
		}
		if strings.Contains(ln.Text, "\n") {
			t.Fatalf("newline in visual line %q", ln.Text)
		}
		if adv := f.Advance(ln.Text); adv > bw+1 {
			t.Fatalf("clipped line %q adv=%v width=%v", ln.Text, adv, bw)
		}
		joined += ln.Text
	}
	if !blank {
		t.Fatal("\\n\\n should yield an empty visual line")
	}
	if !strings.Contains(joined, "There are no accounts") || !strings.Contains(joined, "mail.json") {
		t.Fatalf("lost text %q", joined)
	}
	if body.Bounds().Dy() <= f.Height()+6 {
		t.Fatalf("dialog body still one line: %v", body.Bounds())
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
