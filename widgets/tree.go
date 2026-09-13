package widgets

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TreeNode is one expandable row in a TreeView.
type TreeNode struct {
	Label    string
	Children []*TreeNode
	Expanded bool
	Bold     bool
	Color    paintengine2d.Color // optional tag swatch
	Data     any
}

// NewTreeNode builds a node.
func NewTreeNode(label string, kids ...*TreeNode) *TreeNode {
	return &TreeNode{Label: label, Children: kids, Expanded: len(kids) > 0}
}

// Leaf reports whether the node has no children.
func (n *TreeNode) Leaf() bool { return n == nil || len(n.Children) == 0 }

type treeRow struct {
	node  *TreeNode
	depth int
}

// TreeView is an expand/collapse hierarchical list with selection.
type TreeView struct {
	widget.Base
	Roots     []*TreeNode
	Selected  *TreeNode
	RowHeight float32
	OffsetY   float32
	OnSelect  func(*TreeNode)
	OnToggle  func(*TreeNode)
	OnContext func(*TreeNode, paintengine2d.Point)
	hover     *TreeNode
	lastClick *TreeNode
	lastAt    time.Time
	vbar      scrollDrag
	rows      rowSceneCache
	flat      []treeRow
	index     map[*TreeNode]int
	flatValid bool
}

// NewTreeView constructs a tree.
func NewTreeView(roots ...*TreeNode) *TreeView {
	t := &TreeView{Roots: roots, RowHeight: 24}
	t.Init(t)
	t.SetWantsFocus(true)
	return t
}

func (t *TreeView) rowH() float32 {
	return style.FittedRowHeight(t.Look(), t.RowHeight)
}

// flatten returns the visible rows. The result is cached: a single MouseMove
// used to walk the whole tree seven times (scroll track, MaxOffset, nodeAt,
// two invalidateNode calls), which allocated megabytes per event on a large
// tree. The cache is dropped by Invalidate / Toggle / SetRoots / Refresh.
func (t *TreeView) flatten() []treeRow {
	if t.flatValid {
		return t.flat
	}
	t.flat = t.flat[:0]
	if t.index == nil {
		t.index = make(map[*TreeNode]int, 64)
	} else {
		clear(t.index)
	}
	var walk func([]*TreeNode, int)
	walk = func(nodes []*TreeNode, depth int) {
		for _, n := range nodes {
			if n == nil {
				continue
			}
			t.index[n] = len(t.flat)
			t.flat = append(t.flat, treeRow{node: n, depth: depth})
			if n.Expanded && len(n.Children) > 0 {
				walk(n.Children, depth+1)
			}
		}
	}
	walk(t.Roots, 0)
	t.flatValid = true
	return t.flat
}

// indexOf is the visible row of n, or -1 when n is absent or collapsed away.
func (t *TreeView) indexOf(n *TreeNode) int {
	if n == nil {
		return -1
	}
	t.flatten()
	if i, ok := t.index[n]; ok {
		return i
	}
	return -1
}

func (t *TreeView) dropFlat() {
	t.flatValid = false
	t.flat = t.flat[:0]
	clear(t.index)
}

// Invalidate drops the cached row layout and the retained row scenes, so node
// label / color / structure changes repaint on the next frame.
func (t *TreeView) Invalidate() {
	t.dropFlat()
	t.rows.reset()
	t.Base.Invalidate()
}

// Refresh re-reads the tree after the caller mutated nodes directly (added
// children, renamed, recolored) and repaints.
func (t *TreeView) Refresh() {
	t.dropFlat()
	t.rows.reset()
	t.clamp()
	t.Base.Invalidate()
}

func (t *TreeView) contentH() float32 { return float32(len(t.flatten())) * t.rowH() }

// MaxOffset is max(0, content − viewport).
func (t *TreeView) MaxOffset() float32 {
	return layout.MaxScroll(t.contentH(), t.LocalBounds().Dy())
}

func (t *TreeView) clamp() {
	t.OffsetY = layout.ClampScroll(t.OffsetY, t.contentH(), t.LocalBounds().Dy())
}

func (t *TreeView) scrollTrack() (track, thumb paintengine2d.Rect) {
	bar, gap := overflowBarSize(t.Look())
	return vScrollThumb(t.LocalBounds(), t.contentH(), t.OffsetY, bar, gap)
}

// VisibleRange is the half-open [lo, hi) window of flattened rows Paint draws.
func (t *TreeView) VisibleRange() (lo, hi int) {
	rows := t.flatten()
	rh := t.rowH()
	if rh <= 0 {
		return 0, 0
	}
	lo = int(t.OffsetY / rh)
	hi = int((t.OffsetY+t.LocalBounds().Dy())/rh) + 1
	if lo < 0 {
		lo = 0
	}
	if hi > len(rows) {
		hi = len(rows)
	}
	return
}

// ScrollTrack is the overflow bar geometry (empty thumb when content fits).
func (t *TreeView) ScrollTrack() (track, thumb paintengine2d.Rect) { return t.scrollTrack() }

// ScrollTo sets OffsetY (clamped) without requiring a wheel event.
func (t *TreeView) ScrollTo(y float32) {
	t.OffsetY = y
	t.clamp()
	t.Invalidate()
}

func (t *TreeView) Measure(c layout.Constraints) paintengine2d.Point {
	h := t.contentH()
	if h <= 0 {
		h = 80
	}
	if c.HasMaxH() && h > c.MaxH {
		h = c.MaxH
	}
	w := float32(200)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (t *TreeView) Arrange(r paintengine2d.Rect) { t.SetBounds(r); t.clamp() }

func (t *TreeView) Paint(ctx *paintengine2d.Context) {
	t.clamp()
	b := t.LocalBounds()
	lk := t.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	rows := t.flatten()
	rh := t.rowH()
	lo := int(t.OffsetY / rh)
	hi := int((t.OffsetY+b.Dy())/rh) + 1
	if lo < 0 {
		lo = 0
	}
	if hi > len(rows) {
		hi = len(rows)
	}
	if rec, ok := ctx.Device().(*paintengine2d.Recorder); ok {
		o := rowOrigin(ctx)
		t.rows.ready(o.X, o.Y, b.Dx(), rh, lookSig(lk))
		recordScrollingRows(rec, ctx, &t.rows, t.ID()^(1<<32), b, b.Dx(), rh, t.OffsetY, 0, lo, hi,
			func(i int) uint64 { return t.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 { return t.rowSig(rows[i]) },
			func(i int) {
				n := rows[i].node
				row := paintengine2d.XYWH(0, 0, b.Dx(), rh)
				lk.DrawTreeRow(ctx, row, n == t.Selected, n == t.hover, n.Expanded, n.Leaf(), rows[i].depth, n.Label, n.Bold)
				paintTreeSwatch(ctx, row, n.Color)
			},
		)
	} else {
		ctx.Save()
		ctx.ClipRect(b)
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - t.OffsetY
			row := paintengine2d.XYWH(0, y, b.Dx(), rh)
			n := rows[i].node
			lk.DrawTreeRow(ctx, row, n == t.Selected, n == t.hover, n.Expanded, n.Leaf(), rows[i].depth, n.Label, n.Bold)
			paintTreeSwatch(ctx, row, n.Color)
		}
		ctx.Restore()
	}
	track, thumb := t.scrollTrack()
	paintOverflowBar(ctx, lk, track, thumb, t.vbar.over, t.vbar.active)
	if t.Focused() {
		lk.DrawFocusRing(ctx, b.Inset(-2))
	}
}

// rowSig is the retained-row cache key for one visible row. Every painted
// attribute must be folded in, or a changed node keeps its cached scene.
func (t *TreeView) rowSig(row treeRow) uint64 {
	n := row.node
	extra := uint64(row.depth+1) << 8
	if n.Expanded {
		extra |= 1
	}
	if n.Leaf() {
		extra |= 2
	}
	if n.Bold {
		extra |= 4
	}
	if c := n.Color; c != (paintengine2d.Color{}) {
		extra |= 8
		// Fold the channels in: "has a swatch" alone let a recolored node
		// keep its cached row.
		extra ^= bits32(c.R)*31 ^ bits32(c.G)*131 ^ bits32(c.B)*313 ^ bits32(c.A)*1013
	}
	return visualSig(n == t.Selected, n == t.hover, extra, n.Label)
}

func paintTreeSwatch(ctx *paintengine2d.Context, row paintengine2d.Rect, col paintengine2d.Color) {
	if col == (paintengine2d.Color{}) {
		return
	}
	side := float32(8)
	if side > row.Dy()-6 {
		side = row.Dy() - 6
	}
	if side < 5 {
		return
	}
	cx := row.Max.X - 12
	cy := (row.Min.Y + row.Max.Y) * 0.5
	ctx.DrawCircle(paintengine2d.Pt(cx, cy), side*0.5, paintengine2d.Fill(col))
}

func (t *TreeView) rowAt(y float32) int {
	rows := t.flatten()
	return rowIndexAt(y, t.OffsetY, t.rowH(), len(rows))
}

// SetRoots replaces the tree and drops the row paint cache so expand/collapse
// and rebuilds cannot paint stale Y slots.
func (t *TreeView) SetRoots(roots []*TreeNode) {
	t.Roots = roots
	t.dropFlat()
	t.rows.reset()
	t.clamp()
	t.Invalidate()
}

func (t *TreeView) nodeAt(y float32) *TreeNode {
	i := t.rowAt(y)
	if i < 0 {
		return nil
	}
	return t.flatten()[i].node
}

func (t *TreeView) expanderHit(e widget.MouseEvent, n *TreeNode, depth int) bool {
	if n == nil || n.Leaf() {
		return false
	}
	m := t.Look().Metrics()
	indent := m.TreeIndent
	if indent <= 0 {
		indent = 16
	}
	pad := float32(8)
	if m.RowPad > 0 {
		pad = m.RowPad + 2
	}
	x := pad + float32(depth)*indent
	hit := indent * 0.45
	if hit < 12 {
		hit = 12
	}
	return e.Pos.X >= x-2 && e.Pos.X <= x+hit
}

func (t *TreeView) invalidateNode(n *TreeNode) {
	i := t.indexOf(n)
	if i < 0 {
		return
	}
	rh := t.rowH()
	y := float32(i)*rh - t.OffsetY
	t.InvalidateRect(paintengine2d.XYWH(0, y, t.LocalBounds().Dx(), rh).Inset(-1))
}

func (t *TreeView) MouseEnter() {}

func (t *TreeView) MouseMove(e widget.MouseEvent) bool {
	track, thumb := t.scrollTrack()
	if off, apply, handled, dirty := t.vbar.move(e.Pos, track, thumb, true, t.MaxOffset()); apply || handled || dirty {
		if apply {
			t.OffsetY = off
			t.clamp()
		}
		t.Invalidate()
		if apply || handled {
			return true
		}
	}
	n := t.nodeAt(e.Pos.Y)
	if n != t.hover {
		old := t.hover
		t.hover = n
		t.invalidateNode(old)
		t.invalidateNode(n)
	}
	return true
}

func (t *TreeView) MouseExit() {
	old := t.hover
	t.hover = nil
	t.vbar.over = false
	t.invalidateNode(old)
}

func (t *TreeView) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	track, thumb := t.scrollTrack()
	if off, ok := t.vbar.press(e.Pos, track, thumb, true, t.OffsetY, t.MaxOffset(), t.LocalBounds().Dy()*0.9); ok {
		t.OffsetY = off
		t.clamp()
		t.Invalidate()
		return true
	}
	i := t.rowAt(e.Pos.Y)
	if i < 0 {
		return true
	}
	rows := t.flatten()
	n := rows[i].node
	if e.Button == platform.ButtonRight {
		t.selectNode(n)
		if t.OnContext != nil {
			o := widget.DeviceOrigin(t)
			t.OnContext(n, paintengine2d.Pt(o.X+e.Pos.X, o.Y+e.Pos.Y))
		}
		return true
	}
	if t.expanderHit(e, n, rows[i].depth) {
		t.Toggle(n)
		return true
	}
	if n == t.lastClick && time.Since(t.lastAt) < doubleClickInterval {
		t.Toggle(n)
		t.lastClick = nil
		return true
	}
	t.lastClick = n
	t.lastAt = time.Now()
	t.selectNode(n)
	return true
}

func (t *TreeView) MouseRelease(widget.MouseEvent) bool {
	if t.vbar.release() {
		t.Invalidate()
		return true
	}
	return false
}

// MouseWheel scrolls, and reports false when it cannot: an unscrollable or
// already-at-the-edge view must let the wheel bubble to an outer scroll pane
// instead of swallowing it.
func (t *TreeView) MouseWheel(e widget.MouseEvent) bool {
	if t.MaxOffset() <= 0 {
		return false
	}
	before := t.OffsetY
	t.OffsetY += wheelDelta(e.Scroll.Y, t.rowH())
	t.clamp()
	if t.OffsetY == before {
		return false
	}
	t.Invalidate()
	return true
}

func (t *TreeView) KeyPress(e widget.KeyEvent) bool {
	if !t.Enabled() {
		return false
	}
	rows := t.flatten()
	if len(rows) == 0 {
		return false
	}
	// -1 means "nothing selected yet": the first Down / Up must land on the
	// first row, not skip it.
	idx := t.indexOf(t.Selected)
	switch e.Key {
	case platform.KeyDown:
		if idx < 0 {
			t.selectNode(rows[0].node)
		} else if idx+1 < len(rows) {
			t.selectNode(rows[idx+1].node)
		}
		return true
	case platform.KeyUp:
		if idx < 0 {
			t.selectNode(rows[0].node)
		} else if idx > 0 {
			t.selectNode(rows[idx-1].node)
		}
		return true
	case platform.KeyHome:
		t.selectNode(rows[0].node)
		return true
	case platform.KeyEnd:
		t.selectNode(rows[len(rows)-1].node)
		return true
	case platform.KeyPageDown:
		page := int(t.LocalBounds().Dy()/t.rowH()) - 1
		if page < 1 {
			page = 1
		}
		next := idx + page
		if idx < 0 {
			next = 0
		}
		if next >= len(rows) {
			next = len(rows) - 1
		}
		t.selectNode(rows[next].node)
		return true
	case platform.KeyPageUp:
		page := int(t.LocalBounds().Dy()/t.rowH()) - 1
		if page < 1 {
			page = 1
		}
		next := idx - page
		if idx < 0 {
			next = 0
		}
		if next < 0 {
			next = 0
		}
		t.selectNode(rows[next].node)
		return true
	case platform.KeyRight:
		if t.Selected != nil && !t.Selected.Leaf() {
			if !t.Selected.Expanded {
				t.Toggle(t.Selected)
			} else if len(t.Selected.Children) > 0 {
				t.selectNode(t.Selected.Children[0])
			}
		}
		return true
	case platform.KeyLeft:
		if t.Selected != nil && t.Selected.Expanded && !t.Selected.Leaf() {
			t.Toggle(t.Selected)
			return true
		}
		if parent := t.parentOf(t.Selected); parent != nil {
			t.selectNode(parent)
		}
		return true
	case platform.KeyReturn, platform.KeySpace:
		if t.Selected != nil && !t.Selected.Leaf() {
			t.Toggle(t.Selected)
		}
		return true
	}
	return false
}

func (t *TreeView) parentOf(n *TreeNode) *TreeNode {
	if n == nil {
		return nil
	}
	var found *TreeNode
	var walk func([]*TreeNode)
	walk = func(nodes []*TreeNode) {
		for _, p := range nodes {
			if p == nil {
				continue
			}
			for _, c := range p.Children {
				if c == n {
					found = p
					return
				}
			}
			if found == nil {
				walk(p.Children)
			}
		}
	}
	walk(t.Roots)
	return found
}

// Toggle expands or collapses n.
func (t *TreeView) Toggle(n *TreeNode) {
	if n == nil || n.Leaf() {
		return
	}
	n.Expanded = !n.Expanded
	t.dropFlat()
	t.rows.reset()
	t.clamp()
	t.Invalidate()
	if t.OnToggle != nil {
		t.OnToggle(n)
	}
}

func (t *TreeView) selectNode(n *TreeNode) {
	if n == nil {
		return
	}
	t.Selected = n
	t.ensureVisible(n)
	t.Invalidate()
	if t.OnSelect != nil {
		t.OnSelect(n)
	}
}

func (t *TreeView) ensureVisible(n *TreeNode) {
	i := t.indexOf(n)
	if i < 0 {
		return
	}
	rh := t.rowH()
	top := float32(i) * rh
	bot := top + rh
	view := t.LocalBounds().Dy()
	if top < t.OffsetY {
		t.OffsetY = top
	}
	if bot > t.OffsetY+view {
		t.OffsetY = bot - view
	}
	t.clamp()
}
