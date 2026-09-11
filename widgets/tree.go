package widgets

import (
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// TreeNode is one expandable row in a TreeView.
type TreeNode struct {
	Label    string
	Children []*TreeNode
	Expanded bool
	Bold     bool
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
	rows      rowSceneCache
}

// NewTreeView constructs a tree.
func NewTreeView(roots ...*TreeNode) *TreeView {
	t := &TreeView{Roots: roots, RowHeight: 26}
	t.Init(t)
	t.SetWantsFocus(true)
	return t
}

func (t *TreeView) rowH() float32 {
	if t.RowHeight <= 0 {
		return 26
	}
	return t.RowHeight
}

func (t *TreeView) flatten() []treeRow {
	var out []treeRow
	var walk func([]*TreeNode, int)
	walk = func(nodes []*TreeNode, depth int) {
		for _, n := range nodes {
			if n == nil {
				continue
			}
			out = append(out, treeRow{node: n, depth: depth})
			if n.Expanded && len(n.Children) > 0 {
				walk(n.Children, depth+1)
			}
		}
	}
	walk(t.Roots, 0)
	return out
}

func (t *TreeView) contentH() float32 { return float32(len(t.flatten())) * t.rowH() }

func (t *TreeView) clamp() {
	mx := t.contentH() - t.LocalBounds().Dy()
	if mx < 0 {
		mx = 0
	}
	if t.OffsetY < 0 {
		t.OffsetY = 0
	}
	if t.OffsetY > mx {
		t.OffsetY = mx
	}
}

func (t *TreeView) Measure(c layout.Constraints) paintengine2d.Point {
	h := t.contentH()
	if h < 80 {
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
	b := t.LocalBounds()
	lk := t.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Field))
	ctx.Save()
	ctx.ClipRect(b)
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
		ob := t.Bounds()
		t.rows.ready(ob.Min.X, ob.Min.Y, b.Dx(), rh)
		recordScrollingRows(rec, &t.rows, t.ID()^(1<<32), t.OffsetY, 0, lo, hi,
			func(i int) uint64 { return t.ID()<<32 | uint64(i) + 1 },
			func(i int) uint64 {
				n := rows[i].node
				extra := uint64(rows[i].depth+1) << 8
				if n.Expanded {
					extra |= 1
				}
				if n.Leaf() {
					extra |= 2
				}
				if n.Bold {
					extra |= 4
				}
				return visualSig(n == t.Selected, n == t.hover, extra, n.Label)
			},
			func(i int) {
				n := rows[i].node
				y := float32(i) * rh
				lk.DrawTreeRow(ctx, paintengine2d.XYWH(0, y, b.Dx(), rh), n == t.Selected, n == t.hover, n.Expanded, n.Leaf(), rows[i].depth, n.Label, n.Bold)
			},
		)
	} else {
		for i := lo; i < hi; i++ {
			y := float32(i)*rh - t.OffsetY
			row := paintengine2d.XYWH(0, y, b.Dx(), rh)
			n := rows[i].node
			lk.DrawTreeRow(ctx, row, n == t.Selected, n == t.hover, n.Expanded, n.Leaf(), rows[i].depth, n.Label, n.Bold)
		}
	}
	ctx.Restore()
	if t.Focused() {
		lk.DrawFocusRing(ctx, b.Inset(-2))
	}
}

func (t *TreeView) rowAt(y float32) int {
	i := int((y + t.OffsetY) / t.rowH())
	rows := t.flatten()
	if i < 0 || i >= len(rows) {
		return -1
	}
	return i
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
	indent := t.Look().Metrics().TreeIndent
	if indent <= 0 {
		indent = 16
	}
	x := float32(8) + float32(depth)*indent
	return e.Pos.X >= x-2 && e.Pos.X <= x+14
}

func (t *TreeView) invalidateNode(n *TreeNode) {
	if n == nil {
		return
	}
	rows := t.flatten()
	rh := t.rowH()
	for i, row := range rows {
		if row.node == n {
			y := float32(i)*rh - t.OffsetY
			t.InvalidateRect(paintengine2d.XYWH(0, y, t.LocalBounds().Dx(), rh).Inset(-1))
			return
		}
	}
}

func (t *TreeView) MouseEnter() {}

func (t *TreeView) MouseMove(e widget.MouseEvent) bool {
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
	t.invalidateNode(old)
}

func (t *TreeView) MousePress(e widget.MouseEvent) bool {
	if !t.Enabled() {
		return false
	}
	t.RequestFocus()
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
	if n == t.lastClick && time.Since(t.lastAt) < 400*time.Millisecond {
		t.Toggle(n)
		t.lastClick = nil
		return true
	}
	t.lastClick = n
	t.lastAt = time.Now()
	t.selectNode(n)
	return true
}

func (t *TreeView) MouseWheel(e widget.MouseEvent) bool {
	dy := e.Scroll.Y
	if dy > -8 && dy < 8 && dy != 0 {
		dy *= t.rowH() * 3
	}
	t.OffsetY += dy
	t.clamp()
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
	idx := 0
	for i, r := range rows {
		if r.node == t.Selected {
			idx = i
			break
		}
	}
	switch e.Key {
	case platform.KeyDown:
		if idx+1 < len(rows) {
			t.selectNode(rows[idx+1].node)
		}
		return true
	case platform.KeyUp:
		if idx > 0 {
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
	rows := t.flatten()
	rh := t.rowH()
	for i, r := range rows {
		if r.node != n {
			continue
		}
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
		return
	}
}
