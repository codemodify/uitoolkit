package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Split is panes side by side (or one above the other) with a sash between
// each pair, as QSplitter and GtkPaned are — but for any number of panes,
// and honouring each one's minimum size rather than a bare ratio.
//
// Space is shared out by weight; a pane that would fall under its minimum
// takes its minimum instead and the rest share what is left. A collapsed
// stack keeps its chrome height and gives the remainder back to the others.
type Split struct {
	widget.Base
	vertical bool
	kids     []Node
	weights  []float32

	drag  int // the sash being dragged, -1 when none is
	hover int // the sash under the pointer, -1 when none is
	// grab is where inside the sash the drag started, so the sash keeps
	// its distance from the pointer.
	grab float32
}

// NewSplit is a split of nodes, each given an equal share. vertical stacks
// them top to bottom.
func NewSplit(vertical bool, nodes ...Node) *Split {
	s := &Split{vertical: vertical, drag: -1, hover: -1}
	s.Init(s)
	for _, n := range nodes {
		s.insert(len(s.kids), n, 1)
	}
	return s
}

func (s *Split) dockNode() {}

// Vertical reports whether the panes are stacked top to bottom.
func (s *Split) Vertical() bool { return s.vertical }

// Nodes are the split's panes, empty ones included.
func (s *Split) Nodes() []Node { return s.kids }

// Weights are the shares the panes take of the space beyond their minimums.
func (s *Split) Weights() []float32 { return s.weights }

// insert puts n at index i with the given weight.
func (s *Split) insert(i int, n Node, weight float32) {
	if n == nil {
		return
	}
	if i < 0 || i > len(s.kids) {
		i = len(s.kids)
	}
	if weight <= 0 {
		weight = 1
	}
	s.kids = append(s.kids, nil)
	copy(s.kids[i+1:], s.kids[i:])
	s.kids[i] = n
	s.weights = append(s.weights, 0)
	copy(s.weights[i+1:], s.weights[i:])
	s.weights[i] = weight
	s.Add(n)
}

// drop takes n out and reports whether it was there.
func (s *Split) drop(n Node) bool {
	for i, k := range s.kids {
		if k != n {
			continue
		}
		s.kids = append(s.kids[:i], s.kids[i+1:]...)
		s.weights = append(s.weights[:i], s.weights[i+1:]...)
		s.Remove(n)
		return true
	}
	return false
}

// indexOf is n's place among the panes, or -1.
func (s *Split) indexOf(n Node) int {
	for i, k := range s.kids {
		if k == n {
			return i
		}
	}
	return -1
}

// live are the panes worth space, as indices into kids.
func (s *Split) live() []int {
	out := make([]int, 0, len(s.kids))
	for i, k := range s.kids {
		if k != nil && k.Visible() && !k.Empty() {
			out = append(out, i)
		}
	}
	return out
}

// Empty reports whether no pane is worth space.
func (s *Split) Empty() bool { return len(s.live()) == 0 }

// Fixed is set when every live pane keeps a fixed extent — an area holding
// nothing but collapsed stacks, so the area collapses with them. Along the
// split's own axis the panes and their sashes add up; across it they share
// the box, so the largest decides.
func (s *Split) Fixed(vertical bool) (float32, bool) {
	live := s.live()
	if len(live) == 0 {
		return 0, false
	}
	var total float32
	if vertical == s.vertical {
		total = sashWidth(s.Look()) * float32(len(live)-1)
	}
	for _, i := range live {
		v, ok := s.kids[i].Fixed(vertical)
		if !ok {
			return 0, false
		}
		if vertical == s.vertical {
			total += v
		} else {
			total = maxF(total, v)
		}
	}
	return total, true
}

// MinSize is the panes' minimums summed along the axis (plus the sashes)
// and the largest of them across it.
func (s *Split) MinSize() paintengine2d.Point {
	live := s.live()
	if len(live) == 0 {
		return paintengine2d.Point{}
	}
	bar := sashWidth(s.Look()) * float32(len(live)-1)
	var along, across float32
	for _, i := range live {
		m := s.kids[i].MinSize()
		if s.vertical {
			along += m.Y
			across = maxF(across, m.X)
		} else {
			along += m.X
			across = maxF(across, m.Y)
		}
	}
	if s.vertical {
		return paintengine2d.Pt(across, along+bar)
	}
	return paintengine2d.Pt(along+bar, across)
}

// extents shares total out between the live panes: fixed panes first, then
// the rest by weight, with every pane clamped to its minimum. When even the
// minimums do not fit, everything is scaled down together so the panes stay
// exclusive rather than spilling out of the box.
func (s *Split) extents(total float32) (live []int, out []float32) {
	live = s.live()
	n := len(live)
	if n == 0 {
		return nil, nil
	}
	avail := total - sashWidth(s.Look())*float32(n-1)
	if avail < 0 {
		avail = 0
	}
	out = make([]float32, n)
	mins := make([]float32, n)
	free := make([]bool, n)
	for j, i := range live {
		mins[j] = axisOf(s.kids[i].MinSize(), s.vertical)
		if v, ok := s.kids[i].Fixed(s.vertical); ok {
			out[j] = v
			continue
		}
		free[j] = true
	}
	// Share the space that is left by weight, pinning any pane that would
	// fall under its minimum and sharing again without it.
	for pass := 0; pass <= n; pass++ {
		rem := avail
		var wsum float32
		for j := range out {
			if free[j] {
				w := s.weights[live[j]]
				if w <= 0 {
					w = 1
				}
				wsum += w
			} else {
				rem -= out[j]
			}
		}
		if rem < 0 {
			rem = 0
		}
		pinned := false
		for j := range out {
			if !free[j] {
				continue
			}
			w := s.weights[live[j]]
			if w <= 0 {
				w = 1
			}
			want := rem * w / wsum
			if wsum <= 0 {
				want = rem
			}
			if want < mins[j] {
				out[j] = mins[j]
				free[j] = false
				pinned = true
				continue
			}
			out[j] = want
		}
		if !pinned {
			break
		}
	}
	// The minimums may still not fit; scale them down together so no pane
	// escapes the split's box.
	var sum float32
	for _, v := range out {
		sum += v
	}
	if sum > avail && sum > 0 {
		k := avail / sum
		for j := range out {
			out[j] *= k
		}
	}
	return live, out
}

// panes are the live panes' boxes and the sashes between them, in local
// coordinates. Both are exclusive: no pane overlaps a sash or a sibling.
func (s *Split) panes() (live []int, panes, sashes []paintengine2d.Rect) {
	b := s.LocalBounds()
	live, ext := s.extents(axisOf(paintengine2d.Pt(b.Dx(), b.Dy()), s.vertical))
	if len(live) == 0 {
		return nil, nil, nil
	}
	bar := sashWidth(s.Look())
	panes = make([]paintengine2d.Rect, len(live))
	sashes = make([]paintengine2d.Rect, maxInt(0, len(live)-1))
	pos := float32(0)
	for j := range live {
		if s.vertical {
			panes[j] = paintengine2d.XYWH(b.Min.X, b.Min.Y+pos, b.Dx(), ext[j])
		} else {
			panes[j] = paintengine2d.XYWH(b.Min.X+pos, b.Min.Y, ext[j], b.Dy())
		}
		pos += ext[j]
		if j < len(sashes) {
			if s.vertical {
				sashes[j] = paintengine2d.XYWH(b.Min.X, b.Min.Y+pos, b.Dx(), bar)
			} else {
				sashes[j] = paintengine2d.XYWH(b.Min.X+pos, b.Min.Y, bar, b.Dy())
			}
			pos += bar
		}
	}
	return live, panes, sashes
}

func (s *Split) Measure(c layout.Constraints) paintengine2d.Point {
	m := s.MinSize()
	w, h := m.X, m.Y
	if c.HasMaxW() {
		w = c.MaxW
	}
	if c.HasMaxH() {
		h = c.MaxH
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (s *Split) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	// Which panes are worth space is settled before the space is shared
	// out: a pane that was hidden while it was empty — an area whose only
	// panel floated — and has just been given a panel back must be in
	// this share, or it is shown at the zero size it was hidden at until
	// something else happens to lay the host out again.
	for _, k := range s.kids {
		if k != nil {
			k.SetVisible(!k.Empty())
		}
	}
	live, panes, _ := s.panes()
	for j, i := range live {
		s.kids[i].Arrange(panes[j])
	}
}

// Paint draws the sashes; the panes paint themselves as children.
func (s *Split) Paint(ctx *paintengine2d.Context) {
	_, _, sashes := s.panes()
	lk := s.Look()
	for i, r := range sashes {
		st := s.State() &^ (style.StateHovered | style.StatePressed)
		if s.hover == i || s.drag == i {
			st |= style.StateHovered
		}
		if s.drag == i {
			st |= style.StatePressed
		}
		lk.DrawSplitter(ctx, r, !s.vertical, st)
	}
}

// sashAt is the sash at local point p, or -1.
func (s *Split) sashAt(p paintengine2d.Point) int {
	_, _, sashes := s.panes()
	for i, r := range sashes {
		if r.Contains(p) {
			return i
		}
	}
	return -1
}

// CursorAt is col-resize / row-resize over a sash, and while dragging one.
func (s *Split) CursorAt(p paintengine2d.Point) platform.Cursor {
	if s.drag >= 0 || s.sashAt(p) >= 0 {
		if s.vertical {
			return platform.CursorRowResize
		}
		return platform.CursorColResize
	}
	return platform.CursorDefault
}

func (s *Split) MousePress(e widget.MouseEvent) bool {
	i := s.sashAt(e.Pos)
	if i < 0 {
		return false
	}
	_, _, sashes := s.panes()
	s.drag = i
	s.grab = axisOf(paintengine2d.Pt(e.Pos.X-sashes[i].Min.X, e.Pos.Y-sashes[i].Min.Y), s.vertical)
	widget.ApplyCursor(s.Host(), s.CursorAt(e.Pos))
	s.Invalidate()
	return true
}

func (s *Split) MouseMove(e widget.MouseEvent) bool {
	if s.drag < 0 {
		if i := s.sashAt(e.Pos); i != s.hover {
			s.hover = i
			s.Invalidate()
		}
		widget.ApplyCursor(s.Host(), s.CursorAt(e.Pos))
		return s.hover >= 0
	}
	s.moveSash(s.drag, axisOf(e.Pos, s.vertical)-s.grab)
	widget.ApplyCursor(s.Host(), s.CursorAt(e.Pos))
	return true
}

func (s *Split) MouseRelease(e widget.MouseEvent) bool {
	if s.drag < 0 {
		return false
	}
	s.drag = -1
	widget.ApplyCursor(s.Host(), s.CursorAt(e.Pos))
	s.Invalidate()
	return true
}

func (s *Split) MouseExit() {
	if s.hover >= 0 {
		s.hover = -1
		s.Invalidate()
	}
	if s.drag < 0 {
		widget.ApplyCursor(s.Host(), platform.CursorDefault)
	}
	s.Base.MouseExit()
}

// moveSash puts sash k's leading edge at pos along the axis, taking the
// space from one neighbour and giving it to the other. Neither may go under
// its minimum, so the sash simply stops there.
func (s *Split) moveSash(k int, pos float32) {
	live, ext := s.extents(axisOf(paintengine2d.Pt(s.LocalBounds().Dx(), s.LocalBounds().Dy()), s.vertical))
	if k < 0 || k+1 >= len(live) {
		return
	}
	bar := sashWidth(s.Look())
	// Where the sash sits now: everything before it, sashes included.
	var start float32
	for j := 0; j < k; j++ {
		start += ext[j] + bar
	}
	minA := axisOf(s.kids[live[k]].MinSize(), s.vertical)
	minB := axisOf(s.kids[live[k+1]].MinSize(), s.vertical)
	pair := ext[k] + ext[k+1]
	a := pos - start
	if a < minA {
		a = minA
	}
	if a > pair-minB {
		a = pair - minB
	}
	if a < 0 {
		a = 0
	}
	ext[k], ext[k+1] = a, pair-a
	s.setWeights(live, ext)
	s.Arrange(s.Bounds())
	s.RequestLayout()
	s.Invalidate()
	if h := hostOf(s); h != nil {
		h.layoutChanged()
	}
}

// setWeights records the extents the panes ended up with as their shares,
// so the next layout keeps them and a resize grows them in proportion.
func (s *Split) setWeights(live []int, ext []float32) {
	var sum float32
	for _, v := range ext {
		sum += v
	}
	if sum <= 0 {
		return
	}
	for j, i := range live {
		w := ext[j] / sum * float32(len(live))
		if w < 0.01 {
			w = 0.01
		}
		s.weights[i] = w
	}
}

// SashPosition is where sash i sits as a share of the split, 0 to 1. Tests
// and assistive technology read it.
func (s *Split) SashPosition(i int) float32 {
	live, ext := s.extents(axisOf(paintengine2d.Pt(s.LocalBounds().Dx(), s.LocalBounds().Dy()), s.vertical))
	if i < 0 || i+1 >= len(live) {
		return 0
	}
	var at, total float32
	for j := range ext {
		if j <= i {
			at += ext[j]
		}
		total += ext[j]
	}
	if total <= 0 {
		return 0
	}
	return at / total
}

// Describe implements widget.Accessible: the split itself is a plain pane;
// its sashes are the split panes assistive technology can move.
func (s *Split) Describe(n *a11y.Node) { n.Role = a11y.RolePane }

// AccessibleItems implements widget.AccessibleItems: one node per sash.
func (s *Split) AccessibleItems() []*a11y.Node {
	live, _, sashes := s.panes()
	out := make([]*a11y.Node, 0, len(sashes))
	for i, r := range sashes {
		n := &a11y.Node{
			ID:     widget.ItemID(s, i),
			Role:   a11y.RoleSplitter,
			Name:   sashName(s.kids[live[i]], s.kids[live[i+1]]),
			Bounds: widget.LocalToWindow(s, r),
		}
		if s.vertical {
			n.State |= a11y.StateVertical
		} else {
			n.State |= a11y.StateHorizontal
		}
		n.HasRange = true
		n.Min, n.Max, n.Now, n.Step = 0, 1, float64(s.SashPosition(i)), 0.05
		out = append(out, n)
	}
	return out
}

// sashName says which two panes a sash is between, so a screen reader can
// tell one sash from the next.
func sashName(a, b Node) string {
	an, bn := nodeName(a), nodeName(b)
	switch {
	case an == "" && bn == "":
		return "Split"
	case an == "":
		return "Split before " + bn
	case bn == "":
		return "Split after " + an
	}
	return "Split between " + an + " and " + bn
}

// nodeName is what to call a node in a sash's name: its panel's title, or
// the first title inside it.
func nodeName(n Node) string {
	switch v := n.(type) {
	case *Stack:
		if cur := v.Current(); cur != nil {
			return widget.PlainText(cur.Title())
		}
	case *Split:
		for _, k := range v.kids {
			if k != nil && !k.Empty() {
				return nodeName(k)
			}
		}
	case *centrePane:
		return "the main view"
	}
	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
