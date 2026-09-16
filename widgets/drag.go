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

// DragAt drags the selected rows out of the list, as the table does.
func (l *ListView) DragAt(local paintengine2d.Point) *widget.Drag {
	if l.OnDrag == nil || !l.Enabled() {
		return nil
	}
	if i := l.indexAt(toView(local, l.frame()).Y); i < 0 {
		return nil
	}
	rows := l.SelectedRows()
	if len(rows) == 0 && l.Selected >= 0 {
		rows = []int{l.Selected}
	}
	if len(rows) == 0 {
		return nil
	}
	return l.OnDrag(rows)
}

// DropTypes: what OnDropAt and OnDropRow take, if anything.
func (l *ListView) DropTypes() []string {
	return rowDropTypes(l.OnDropAt != nil || l.OnDropRow != nil, l.DropMimes)
}

// Drop hands the drop to the row under it, or to the gap it landed in.
func (l *ListView) Drop(e widget.DropEvent) bool {
	row, at := l.dropSpotAt(toView(e.Pos, l.frame()).Y)
	l.markDrop(-1, -1)
	switch {
	case at >= 0 && l.OnDropAt != nil:
		return l.OnDropAt(at, e)
	case row >= 0 && l.OnDropRow != nil:
		return l.OnDropRow(row, e)
	}
	return false
}

// DragOver marks the row a drag is over, or the gap between two rows a
// reorderable list would insert it into.
func (l *ListView) DragOver(pos paintengine2d.Point) {
	l.markDrop(l.dropSpotAt(toView(pos, l.frame()).Y))
}

func (l *ListView) DragLeave() { l.markDrop(-1, -1) }

// DropActionFor is what a drop on the list would do.
func (l *ListView) DropActionFor(offered platform.DragAction) platform.DragAction {
	return dropActions(l.DropActions, offered)
}

func (l *ListView) dropSpotAt(y float32) (row, at int) {
	if l.OnDropAt == nil {
		return rowIndexAt(y, l.OffsetY, l.rowH(), l.Count), -1
	}
	return rowDropSpot(y, l.OffsetY, l.rowH(), l.Count, l.OnDropRow != nil)
}

func (l *ListView) markDrop(row, at int) {
	if l.dropRow == row && l.dropAt == at {
		return
	}
	l.dropRow, l.dropAt = row, at
	l.Invalidate()
}

// DropTypes: what OnDropAt and OnDropRow take, if anything.
func (t *TableView) DropTypes() []string {
	return rowDropTypes(t.OnDropAt != nil || t.OnDropRow != nil, t.DropMimes)
}

// Drop hands the drop to the row under it, or to the gap it landed in.
func (t *TableView) Drop(e widget.DropEvent) bool {
	row, at := t.dropSpotAt(toView(e.Pos, t.frame()).Y)
	t.markDrop(-1, -1)
	switch {
	case at >= 0 && t.OnDropAt != nil:
		return t.OnDropAt(at, e)
	case row >= 0 && t.OnDropRow != nil:
		return t.OnDropRow(row, e)
	}
	return false
}

// DragOver marks the row a drag is over, or the gap between two rows a
// reorderable table would insert it into.
func (t *TableView) DragOver(pos paintengine2d.Point) {
	t.markDrop(t.dropSpotAt(toView(pos, t.frame()).Y))
}

func (t *TableView) DragLeave() { t.markDrop(-1, -1) }

// DropActionFor is what a drop on the table would do.
func (t *TableView) DropActionFor(offered platform.DragAction) platform.DragAction {
	return dropActions(t.DropActions, offered)
}

// dropSpotAt works in the body's own space: the table's rows start under
// the sticky header, which is no part of any row's gap.
func (t *TableView) dropSpotAt(y float32) (row, at int) {
	hh := t.headerH()
	if y < hh {
		// Over the header: the gap above the first row, which is where a
		// drop aimed at the top of the table means to go.
		if t.OnDropAt == nil {
			return -1, -1
		}
		return -1, 0
	}
	if t.OnDropAt == nil {
		return t.indexAt(y), -1
	}
	return rowDropSpot(y-hh, t.OffsetY, t.rowH(), t.RowCount, t.OnDropRow != nil)
}

func (t *TableView) markDrop(row, at int) {
	if t.dropRow == row && t.dropAt == at {
		return
	}
	t.dropRow, t.dropAt = row, at
	t.Invalidate()
}

// rowDropTypes is what a row view takes: nothing when it has no hook for
// a drop at all, the types it names, or a uri-list — the type a file lands
// as, which is what a view that names none is nearly always after.
func rowDropTypes(takes bool, mimes []string) []string {
	if !takes {
		return nil
	}
	if len(mimes) > 0 {
		return mimes
	}
	return []string{"text/uri-list"}
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

// DropTypes: what OnDropNode and OnDropAt take, if anything.
func (t *TreeView) DropTypes() []string {
	return rowDropTypes(t.OnDropNode != nil || t.OnDropAt != nil, t.DropMimes)
}

// Drop hands the drop to the node under it, or — when it landed in a gap
// between two rows — to the place in the tree that gap stands for.
func (t *TreeView) Drop(e widget.DropEvent) bool {
	p := toView(e.Pos, t.frame())
	row, at := t.dropSpotAt(p)
	t.clearDropMarks()
	if at >= 0 {
		if t.OnDropAt == nil {
			return false
		}
		parent, i, _ := t.gapTarget(at, p.X)
		return t.OnDropAt(parent, i, e)
	}
	if row < 0 || t.OnDropNode == nil {
		return false
	}
	rows := t.flatten()
	if row >= len(rows) {
		return false
	}
	return t.OnDropNode(rows[row].node, e)
}

// DragOver marks where a drop would land: the row itself, the way a file
// manager lights up the folder a file would go into, or the gap between
// two rows when the tree can be reordered and the pointer is in one.
func (t *TreeView) DragOver(pos paintengine2d.Point) {
	p := toView(pos, t.frame())
	row, at := t.dropSpotAt(p)
	depth := 0
	if at >= 0 {
		_, _, depth = t.gapTarget(at, p.X)
	}
	t.markDrop(row, at, depth)
}

func (t *TreeView) DragLeave() { t.clearDropMarks() }

// dropSpotAt is where a drop at view-space p would land: onto row row, or
// in the gap before row at. A tree with no OnDropAt has no gaps at all, so
// it behaves exactly as it did before there were any.
func (t *TreeView) dropSpotAt(p paintengine2d.Point) (row, at int) {
	rows := len(t.flatten())
	if t.OnDropAt == nil {
		return rowIndexAt(p.Y, t.OffsetY, t.rowH(), rows), -1
	}
	return rowDropSpot(p.Y, t.OffsetY, t.rowH(), rows, t.OnDropNode != nil)
}

// markDrop sets the tree's two drag marks at once: the row a drop would
// land on, and the gap it would be inserted into.
func (t *TreeView) markDrop(row, at, depth int) {
	if t.dropRow == row && t.dropAt == at && t.dropDepth == depth {
		return
	}
	t.dropRow, t.dropAt, t.dropDepth = row, at, depth
	t.Invalidate()
}

func (t *TreeView) clearDropMarks() { t.markDrop(-1, -1, 0) }

// gapTarget is the place in the tree the gap before flat row at stands
// for: the node a drop there becomes a child of (nil for a root), the
// place it takes among that node's children, and the depth the gap is
// drawn at.
//
// A gap is ambiguous wherever the row above it is deeper than the row
// below — the gap after a folder's last child is also the gap before the
// folder's next sibling, and both are somewhere a user means to drop. The
// pointer's own x settles it, which is how every tree that can be
// reordered resolves the same ambiguity: aim at the indent of the level
// you mean.
func (t *TreeView) gapTarget(at int, px float32) (parent *TreeNode, index, depth int) {
	rows := t.flatten()
	if at < 0 {
		at = 0
	}
	if at > len(rows) {
		at = len(rows)
	}
	above := at - 1
	if above < 0 {
		// Above the first row: the head of the roots, and nothing else
		// it could mean.
		return nil, 0, 0
	}
	// The deepest level this gap can stand for is one inside the row
	// above when that row is an open folder, and the row above's own
	// level otherwise; the shallowest is the level of the row below it,
	// which the gap is already drawn against.
	deepest := rows[above].depth
	if n := rows[above].node; !n.Leaf() && n.Expanded {
		deepest++
	}
	shallowest := 0
	if at < len(rows) {
		shallowest = rows[at].depth
	}
	depth = t.depthAtX(px)
	if depth > deepest {
		depth = deepest
	}
	if depth < shallowest {
		depth = shallowest
	}
	if depth > rows[above].depth {
		// Into the open folder above, before everything already in it.
		return rows[above].node, 0, depth
	}
	// After the row above's ancestor at this depth: the nearest row at or
	// before it whose level is the one the pointer picked.
	i := above
	for i > 0 && rows[i].depth > depth {
		i--
	}
	parent = t.rowParent(i)
	return parent, t.childIndex(parent, rows[i].node) + 1, depth
}

// rowParent is the node flat row i hangs under: the nearest row before it
// one level shallower. nil for a root.
func (t *TreeView) rowParent(i int) *TreeNode {
	rows := t.flatten()
	if i < 0 || i >= len(rows) || rows[i].depth == 0 {
		return nil
	}
	for j := i - 1; j >= 0; j-- {
		if rows[j].depth == rows[i].depth-1 {
			return rows[j].node
		}
	}
	return nil
}

// childIndex is n's place among parent's children (among the roots when
// parent is nil), -1 when it is not there at all.
func (t *TreeView) childIndex(parent, n *TreeNode) int {
	list := t.Roots
	if parent != nil {
		list = parent.Children
	}
	for i, c := range list {
		if c == n {
			return i
		}
	}
	return -1
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

// DropActionFor is what a drop on the tab strip would do: a tab dragged
// out of another window moves — it is in one window or the other, and a
// copy would leave two of it — while anything else does what DropActions
// allows. The strip knows which it is looking at because the caret is up
// (DragOverMime, tearoff.go); Wayland needs the answer that way round,
// since there a target's preference is all the compositor has to go on.
func (t *BrowserTabs) DropActionFor(offered platform.DragAction) platform.DragAction {
	if t.dropAt >= 0 && offered.Has(platform.DragMove) {
		return platform.DragMove
	}
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
