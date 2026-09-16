package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Dragging out of the collection widgets and the text editors, and the
// rows and tabs that take a drop.
//
// A widget says what a press drags with [widget.DragSource]; the window
// asks once the press has moved past the drag threshold, so nothing here
// tracks the pointer itself. The drop half is [widget.DropTarget] as it
// always was, with a highlight on the row (or tab) under the drag rather
// than around the whole view.

// ---- text: drag the selection out ----------------------------------------------

// A press inside the selection does not move the caret: it may be the
// start of a drag of that text. The selection collapses on release
// instead, if no drag took it.

// pressInSelection reports whether index i is inside [a, b).
func pressInSelection(i, a, b int) bool {
	if a > b {
		a, b = b, a
	}
	return a != b && i >= a && i < b
}

// DragAt drags the selected text out of the field.
func (t *TextField) DragAt(paintengine2d.Point) *widget.Drag {
	if !t.dragSel {
		return nil
	}
	return textDrag(t, t.SelectedText(), func(a, b int) {
		t.selA, t.selB = a, b
		t.replaceSel("")
	}, t.Enabled(), t.selA, t.selB)
}

// DragAt drags the selected text out of the area.
func (t *TextArea) DragAt(paintengine2d.Point) *widget.Drag {
	if !t.dragSel {
		return nil
	}
	return textDrag(t, t.SelectedText(), func(a, b int) {
		t.selA, t.selB = a, b
		t.replaceSel("")
	}, t.Enabled() && !t.ReadOnly, t.selA, t.selB)
}

// textDrag is a drag of selected text, with the picture the look would
// draw for it. A move takes the text away here once the target says it
// performed one — unless the drop landed in this same widget, where the
// text was already inserted and taking the original away too would be a
// surprise rather than a move.
func textDrag(c widget.Component, text string, cut func(a, b int), editable bool, a, b int) *widget.Drag {
	if text == "" {
		return nil
	}
	d := widget.DragText(text)
	if d == nil {
		return nil
	}
	if !editable {
		d.Actions, d.Preferred = platform.DragCopy, platform.DragCopy
	}
	d.Source = c
	d.Image, d.Hotspot = widget.DragLabel(c.Look(), dragLabelFor(text), dragScale(c))
	if a > b {
		a, b = b, a
	}
	d.Done = func(action platform.DragAction) {
		if action == platform.DragMove && editable && !droppedOnSelf(c) {
			cut(a, b)
		}
		clearDragSel(c)
	}
	return d
}

// droppedOnSelf reports that the drag's own widget took the drop, which
// the text widgets record while handling it.
func droppedOnSelf(c widget.Component) bool {
	if s, ok := c.(interface{ tookOwnDrop() bool }); ok {
		return s.tookOwnDrop()
	}
	return false
}

func clearDragSel(c widget.Component) {
	if s, ok := c.(interface{ endDragSel() }); ok {
		s.endDragSel()
	}
}

func (t *TextField) tookOwnDrop() bool { return t.selfDrop }
func (t *TextArea) tookOwnDrop() bool  { return t.selfDrop }

func (t *TextField) endDragSel() { t.dragSel, t.selfDrop = false, false }
func (t *TextArea) endDragSel()  { t.dragSel, t.selfDrop = false, false }

// dragLabelFor is the short text a drag's picture shows: the text
// dragged, cut to a line's worth.
func dragLabelFor(text string) string {
	const max = 32
	r := []rune(text)
	for i, ch := range r {
		if ch == '\n' || ch == '\r' {
			r = r[:i]
			break
		}
	}
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return string(r)
}

// dragScale is the display scale a drag's picture is drawn at, so it is
// as sharp as the window it came from.
func dragScale(c widget.Component) float32 {
	if h := c.Host(); h != nil {
		if s := h.Scale(); s > 0 {
			return s
		}
	}
	return 1
}

// ---- lists, tables and trees ---------------------------------------------------

// DragAt drags the selected rows out of the table. It is the OnDrag hook
// that decides what they are worth dragging as — the table knows about
// rows, not about what they hold.
func (t *TableView) DragAt(local paintengine2d.Point) *widget.Drag {
	if t.OnDrag == nil || !t.Enabled() {
		return nil
	}
	p := toView(local, t.frame())
	if i := t.indexAt(p.Y); i < 0 || p.Y >= t.inner().Dy() {
		return nil
	}
	rows := t.SelectedRows()
	if len(rows) == 0 && t.Selected >= 0 {
		rows = []int{t.Selected}
	}
	if len(rows) == 0 {
		return nil
	}
	return t.OnDrag(rows)
}

// DragAt drags the node under the press out of the tree.
func (t *TreeView) DragAt(local paintengine2d.Point) *widget.Drag {
	if t.OnDrag == nil || !t.Enabled() {
		return nil
	}
	n := t.nodeAt(toView(local, t.frame()).Y)
	if n == nil {
		return nil
	}
	return t.OnDrag(n)
}

// DropTypes: what OnDropNode takes, if anything.
func (t *TreeView) DropTypes() []string {
	if t.OnDropNode == nil {
		return nil
	}
	if len(t.DropMimes) > 0 {
		return t.DropMimes
	}
	return []string{"text/uri-list"}
}

// Drop hands the drop to the node under it.
func (t *TreeView) Drop(e widget.DropEvent) bool {
	n := t.nodeAt(toView(e.Pos, t.frame()).Y)
	t.clearDropRow()
	if n == nil || t.OnDropNode == nil {
		return false
	}
	return t.OnDropNode(n, e)
}

// DragOver highlights the row a drag is over, the way a file manager
// lights up the folder a file would land in.
func (t *TreeView) DragOver(pos paintengine2d.Point) {
	row := t.rowAt(toView(pos, t.frame()).Y)
	if row != t.dropRow {
		t.dropRow = row
		t.Invalidate()
	}
}

func (t *TreeView) DragLeave() { t.clearDropRow() }

func (t *TreeView) clearDropRow() {
	if t.dropRow != -1 {
		t.dropRow = -1
		t.Invalidate()
	}
}

// The document tabs' own halves of this — what a tab dragged out of the
// strip offers, and what the strip takes from one dragged in — are in
// tearoff.go.

// ---- the highlight -------------------------------------------------------------

// paintDropRow tints the row (or tab) a drag is over in the look's accent,
// so what a drop would land on is never in doubt.
func paintDropRow(ctx *paintengine2d.Context, lk style.LookAndFeel, b paintengine2d.Rect) {
	if b.Empty() {
		return
	}
	acc := lk.Palette().Accent
	r := lk.Metrics().Radius
	ctx.DrawRoundRect(b, r, r, paintengine2d.Fill(acc.WithAlpha(0.18)))
	ctx.DrawRoundRect(b.Inset(0.5), r, r, paintengine2d.StrokePaint(acc, max(1, style.Dip(lk, 1))))
}

// DropActionFor is what a drop on the tree would do: what DropActions
// allows of the drag's own set, and a copy when it says nothing.
func (t *TreeView) DropActionFor(offered platform.DragAction) platform.DragAction {
	return dropActions(t.DropActions, offered)
}

// DropActionFor is what a drop on the tab strip would do.
func (t *BrowserTabs) DropActionFor(offered platform.DragAction) platform.DragAction {
	return dropActions(t.DropActions, offered)
}

// dropActions narrows what a target allows to what the drag offers. A
// target that allows nothing in particular copies, which every drop
// target can manage.
func dropActions(allowed, offered platform.DragAction) platform.DragAction {
	if allowed == platform.DragNone {
		return platform.DragCopy
	}
	if both := allowed & offered; both != platform.DragNone {
		return both
	}
	return allowed
}
