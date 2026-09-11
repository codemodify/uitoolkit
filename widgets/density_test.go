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

func TestTableBoldUsesBodySizeNotTitle(t *testing.T) {
	look := style.DarkLook()
	if look.BoldFont().Size != look.Font().Size {
		t.Fatalf("bold %v body %v", look.BoldFont().Size, look.Font().Size)
	}
	if look.TitleFont().Size <= look.Font().Size+2 {
		t.Fatal("title must stay a display size, not a row face")
	}
}
