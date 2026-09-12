package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

type countHost struct {
	host
	n int
}

func (h *countHost) Invalidate(widget.Component, paintengine2d.Rect) { h.n++ }

func mountCount(c widget.Component, box paintengine2d.Rect) *countHost {
	h := &countHost{}
	c.SetHost(h)
	c.SetLook(style.LightLook())
	_ = c.Measure(layout.Tight(box.Dx(), box.Dy()))
	c.Arrange(box)
	return h
}

func TestStaticHoverDoesNotInvalidate(t *testing.T) {
	title := NewTitle("YOU NEED TO SEE THESE NEW PRICES")
	body := NewTextView("Generation 6 — body preview that must stay bold/regular.", "")
	col := NewColumn(title, NewLabel("From: shop@example.com"), NewPad(8, body))
	h := mountCount(col, paintengine2d.XYWH(0, 0, 420, 280))
	lk := style.LightLook()
	title.SetLook(lk)
	if title.font() != lk.TitleFont() {
		t.Fatal("subject must measure with TitleFont")
	}
	n := h.n
	title.MouseEnter()
	body.MouseEnter()
	col.MouseEnter()
	if h.n != n {
		t.Fatalf("static/pane hover dirtied %d times", h.n-n)
	}
	if title.font() != title.Look().TitleFont() {
		t.Fatal("hover changed title font")
	}
}

func TestButtonHoverStillInvalidates(t *testing.T) {
	btn := NewButton("Save All", nil)
	h := mountCount(btn, paintengine2d.XYWH(0, 0, 120, 36))
	n := h.n
	btn.MouseEnter()
	if h.n <= n {
		t.Fatal("button hover chrome must dirty")
	}
	n = h.n
	btn.MouseExit()
	if h.n <= n {
		t.Fatal("button leave must dirty")
	}
}

func TestScrollViewBarHoverDirtiesTrackOnly(t *testing.T) {
	inner := NewColumn()
	for i := 0; i < 30; i++ {
		inner.Add(NewLabel("row content for scroll hover"))
	}
	sv := NewScrollView(inner)
	h := mountCount(sv, paintengine2d.XYWH(0, 0, 200, 120))
	track, _ := sv.thumb()
	if track.Empty() {
		t.Fatal("need overflow track")
	}
	n := h.n
	sv.MouseEnter()
	if h.n != n {
		t.Fatal("entering the pane must not dirty the content")
	}
	mid := paintengine2d.Pt((track.Min.X+track.Max.X)*0.5, (track.Min.Y+track.Max.Y)*0.5)
	sv.MouseMove(widget.MouseEvent{Pos: mid})
	if h.n <= n {
		t.Fatal("bar hover should dirty the track")
	}
	if h.n-n > 2 {
		t.Fatalf("bar hover should not invalidate the whole view (%d)", h.n-n)
	}
}

func TestTitleStaysTitleFontAfterHover(t *testing.T) {
	look := style.LightLook()
	title := NewTitle("Subject line")
	title.SetLook(look)
	title.SetHost(&host{})
	title.Arrange(paintengine2d.XYWH(0, 0, 360, 28))
	if title.font() != look.TitleFont() {
		t.Fatal("TitleFont")
	}
	title.MouseEnter()
	title.MouseExit()
	if title.font() != look.TitleFont() {
		t.Fatal("hover restyled the subject")
	}
}
