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
	// Frameless drops the look's view frame (a tree that already sits in a
	// framed pane).
	Frameless bool
	hover     *TreeNode
	hoverExp  bool // the pointer is on the hovered row's expander
	lastClick *TreeNode
	lastAt    time.Time
	vbar      scrollDrag
	rows      rowSceneCache
	flat      []treeRow
	index     map[*TreeNode]int
	flatValid bool
	reveal    *TreeNode // brought into view at the next Arrange
	find      typeAhead
	// DisableTypeAhead turns off type-ahead find, for views whose letters
	// are commands (Mail's n / p / r).
	DisableTypeAhead bool
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

// frame is the look's view frame around the tree (zero on flat looks).
func (t *TreeView) frame() style.Insets { return viewFrame(t.Look(), t.Frameless) }

// inner is the viewport in view space (inside the frame).
func (t *TreeView) inner() paintengine2d.Rect { return viewInner(t.LocalBounds(), t.frame()) }

// MaxOffset is max(0, content − viewport).
func (t *TreeView) MaxOffset() float32 {
	return layout.MaxScroll(t.contentH(), t.inner().Dy())
}

func (t *TreeView) clamp() {
	t.OffsetY = layout.ClampScroll(t.OffsetY, t.contentH(), t.inner().Dy())
}

func (t *TreeView) vparts() style.ScrollParts {
	return vScrollParts(t.Look(), t.inner(), t.contentH(), t.OffsetY)
}

func (t *TreeView) vaxis() scrollAxis {
	return scrollAxis{
		vertical: true,
		parts:    t.vparts,
		get:      func() (float32, float32) { return t.OffsetY, t.MaxOffset() },
		set:      func(y float32) { t.OffsetY = y; t.clamp(); t.Invalidate() },
		steps:    func() (float32, float32) { return t.rowH(), t.inner().Dy() * 0.9 },
	}
}

// rowsW is the row width: the view minus the gutter a visible bar takes.
func (t *TreeView) rowsW() float32 {
	return t.inner().Dx() - scrollGutter(t.Look(), t.MaxOffset() > 0)
}

func (t *TreeView) scrollTrack() (track, thumb paintengine2d.Rect) {
	sp := t.vparts()
	in := t.frame()
	return fromView(sp.Track, in), fromView(sp.Thumb, in)
}

// VisibleRange is the half-open [lo, hi) window of flattened rows Paint draws.
func (t *TreeView) VisibleRange() (lo, hi int) {
	rows := t.flatten()
	rh := t.rowH()
	if rh <= 0 {
		return 0, 0
	}
	lo = int(t.OffsetY / rh)
	hi = int((t.OffsetY+t.inner().Dy())/rh) + 1
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

func (t *TreeView) Arrange(r paintengine2d.Rect) {
	t.SetBounds(r)
	if n := t.reveal; n != nil {
		t.reveal = nil
		t.ensureVisible(n)
	}
	t.clamp()
}

// EnsureVisible scrolls the least needed to bring n's row into view (n
// must be visible: its ancestors expanded). Called before the tree is laid
// out, it applies at the first Arrange.
func (t *TreeView) EnsureVisible(n *TreeNode) {
	if n == nil {
		return
	}
	if t.inner().Dy() <= 0 {
		t.reveal = n
		return
	}
	t.ensureVisible(n)
	t.Invalidate()
}

func (t *TreeView) Paint(ctx *paintengine2d.Context) {
	t.clamp()
	lk := t.Look()
	if beginViewFrame(ctx, lk, t.LocalBounds(), t.frame(), t.State()) {
		defer ctx.Restore()
	}
	b := t.inner()
	rw := t.rowsW()
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
		t.rows.ready(o.X, o.Y, rw, rh, lookSig(lk))
		recordScrollingRows(rec, ctx, &t.rows, t.ID()^(1<<32), b, rw, rh, t.OffsetY, 0, lo, hi,
			func(i int) uint64 { return t.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 { return t.rowSig(rows[i]) },
			func(i int) {
				n := rows[i].node
				row := paintengine2d.XYWH(0, 0, rw, rh)
				lk.DrawTreeRow(ctx, row, t.rowState(n), n.Expanded, n.Leaf(), rows[i].depth, n.Label, n.Bold)
				paintTreeSwatch(ctx, row, n.Color)
			},
		)
	} else {
		ctx.Save()
		ctx.ClipRect(b)
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - t.OffsetY
			row := paintengine2d.XYWH(0, y, rw, rh)
			n := rows[i].node
			lk.DrawTreeRow(ctx, row, t.rowState(n), n.Expanded, n.Leaf(), rows[i].depth, n.Label, n.Bold)
			paintTreeSwatch(ctx, row, n.Color)
		}
		ctx.Restore()
	}
	t.vbar.paint(t, ctx, lk, t.vparts(), true, t.OffsetY)
	// The current row carries the focus mark; a focused tree without one
	// rings itself.
	if t.Focused() && t.indexOf(t.Selected) < 0 {
		lk.DrawFocusRing(ctx, b)
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
	return visualSig(n == t.Selected, n == t.hover, extra^uint64(t.rowState(n))<<40, n.Label)
}

// rowState is n's item state for the look.
func (t *TreeView) rowState(n *TreeNode) style.ControlState {
	st := widget.RowItemState(t, t.indexOf(n), n == t.Selected, n == t.hover, n == t.Selected)
	if n == t.hover && t.hoverExp {
		st |= style.StateExpanderHot
	}
	return st
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

// expanderHit reports whether view-space x hits n's expander arrow.
func (t *TreeView) expanderHit(px float32, n *TreeNode, depth int) bool {
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
	return px >= x-2 && px <= x+hit
}

func (t *TreeView) invalidateNode(n *TreeNode) {
	i := t.indexOf(n)
	if i < 0 {
		return
	}
	rh := t.rowH()
	y := float32(i)*rh - t.OffsetY
	t.InvalidateRect(fromView(paintengine2d.XYWH(0, y, t.inner().Dx(), rh), t.frame()).Inset(-1))
}

func (t *TreeView) MouseEnter() {}

func (t *TreeView) MouseMove(e widget.MouseEvent) bool {
	p := toView(e.Pos, t.frame())
	if handled, dirty := t.vbar.move(t, p, t.vaxis()); handled || dirty {
		if dirty {
			t.Invalidate()
		}
		if handled {
			return true
		}
	}
	var n *TreeNode
	exp := false
	if p.Y >= 0 && p.Y < t.inner().Dy() {
		if i := t.rowAt(p.Y); i >= 0 {
			row := t.flatten()[i]
			n = row.node
			exp = t.expanderHit(p.X, n, row.depth)
		}
	}
	if n != t.hover || exp != t.hoverExp {
		old := t.hover
		t.hover, t.hoverExp = n, exp
		t.invalidateNode(old)
		t.invalidateNode(n)
	}
	return true
}

func (t *TreeView) MouseExit() {
	old := t.hover
	t.hover, t.hoverExp = nil, false
	t.vbar.exit()
	t.invalidateNode(old)
}

func (t *TreeView) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
	p := toView(e.Pos, t.frame())
	if t.vbar.press(t, p, t.vaxis()) {
		t.Invalidate()
		return true
	}
	i := t.rowAt(p.Y)
	if i < 0 || p.Y < 0 || p.Y >= t.inner().Dy() {
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
	if t.expanderHit(p.X, n, rows[i].depth) {
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
	if contextKey(e) {
		if t.OnContext == nil {
			return false
		}
		var row paintengine2d.Rect
		if i := t.indexOf(t.Selected); i >= 0 {
			rh := t.rowH()
			row = fromView(paintengine2d.XYWH(0, float32(i)*rh-t.OffsetY, t.inner().Dx(), rh), t.frame())
		}
		t.OnContext(t.Selected, contextPoint(t, row))
		return true
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
		page := int(t.inner().Dy()/t.rowH()) - 1
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
		page := int(t.inner().Dy()/t.rowH()) - 1
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

// TextInput is type-ahead find over the visible (expanded) rows.
func (t *TreeView) TextInput(r rune) bool {
	if !t.Enabled() || t.DisableTypeAhead {
		return false
	}
	rows := t.flatten()
	i, searched := t.find.next(r, t.indexOf(t.Selected), len(rows), func(i int) string { return rows[i].node.Label })
	if i >= 0 {
		t.selectNode(rows[i].node)
	}
	return searched
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
	view := t.inner().Dy()
	if top < t.OffsetY {
		t.OffsetY = top
	}
	if bot > t.OffsetY+view {
		t.OffsetY = bot - view
	}
	t.clamp()
}
