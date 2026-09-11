package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

type scaledHost struct {
	focus widget.Component
	look  style.LookAndFeel
}

func (h *scaledHost) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *scaledHost) RequestFocus(c widget.Component)                 { h.focus = c }
func (h *scaledHost) Focus() widget.Component                         { return h.focus }
func (h *scaledHost) Scale() float32 {
	if h.look != nil {
		return style.LookScale(h.look)
	}
	return 1
}
func (h *scaledHost) Look() style.LookAndFeel {
	if h.look != nil {
		return h.look
	}
	return style.DarkLook()
}
func (h *scaledHost) RequestLayout() {}

func TestTableTreeRowsFitScaledFont(t *testing.T) {
	look := style.WithScale(style.DarkLook(), 2)
	h := &scaledHost{look: look}

	tv := NewTableView([]TableColumn{
		{Title: "Subject", Width: 120},
		{Title: "Date", Width: 80},
	}, 8, func(row, col int) string {
		if col == 0 {
			return "Hello World"
		}
		return "Yesterday"
	}, nil)
	tv.CellBold = func(row, col int) bool { return row == 0 }
	tv.SetHost(h)
	tv.Arrange(paintengine2d.XYWH(0, 0, 400, 320))
	if tv.rowH() < look.Font().Height()+4 {
		t.Fatalf("table row %v < font %v", tv.rowH(), look.Font().Height())
	}
	if tv.headerH() < tv.rowH() {
		t.Fatalf("header %v < row %v", tv.headerH(), tv.rowH())
	}
	widths := tv.colWidths()
	if widths[0] < 200 || widths[1] < 140 {
		t.Fatalf("columns not scaled: %v", widths)
	}

	tree := NewTreeView(
		NewTreeNode("ada@example.com",
			NewTreeNode("Inbox (36)"),
			NewTreeNode("Drafts"),
			NewTreeNode("Sent"),
		),
	)
	tree.Roots[0].Children[0].Bold = true
	tree.SetHost(h)
	tree.Arrange(paintengine2d.XYWH(0, 0, 220, 280))
	if tree.rowH() < look.Font().Height()+4 {
		t.Fatalf("tree row %v < font %v", tree.rowH(), look.Font().Height())
	}

	img := paintengine2d.NewImage(400, 320)
	ctx := paintengine2d.NewContext(img)
	tv.Paint(ctx)
	tree.Paint(ctx)
}

func TestTableFlexSubjectGetsLeftover(t *testing.T) {
	tv := NewTableView([]TableColumn{
		{Title: "★", Width: 28, MinWidth: 24},
		{Title: "📎", Width: 28, MinWidth: 24},
		{Title: "Subject", MinWidth: 180},
		{Title: "Correspondents", Width: 148, MinWidth: 110},
		{Title: "Date", Width: 108, MinWidth: 88},
		{Title: "Size", Width: 72, MinWidth: 60},
	}, 4, func(row, col int) string { return "x" }, nil)
	tv.SetHost(&host{})
	tv.Arrange(paintengine2d.XYWH(0, 0, 640, 200))
	w := tv.ColumnWidths()
	if len(w) != 6 {
		t.Fatalf("cols %d", len(w))
	}
	sum := float32(0)
	for _, x := range w {
		sum += x
	}
	if sum < 639 || sum > 641 {
		t.Fatalf("widths must fill row: %v sum=%v", w, sum)
	}
	if w[2] < 220 {
		t.Fatalf("subject should flex, got %v (all %v)", w[2], w)
	}
	if w[3] < 110 || w[3] > 160 {
		t.Fatalf("correspondents %v", w[3])
	}

	tv.Arrange(paintengine2d.XYWH(0, 0, 480, 200))
	w = tv.ColumnWidths()
	if w[2] < 160 {
		t.Fatalf("narrow subject starved: %v", w)
	}
	sum = 0
	for _, x := range w {
		sum += x
	}
	if sum < 479 || sum > 481 {
		t.Fatalf("narrow sum %v %v", sum, w)
	}
}

func TestTableBoldUsesBodySizeNotTitle(t *testing.T) {
	look := style.DarkLook()
	if look.BoldFont().Size != look.Font().Size {
		t.Fatalf("bold %v body %v", look.BoldFont().Size, look.Font().Size)
	}
	if look.TitleFont().Size <= look.Font().Size+2 {
		t.Fatal("title must stay a display size, not a row face")
	}
}
