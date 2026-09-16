package dock

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/codemodify/paintengine2d"
)

// LayoutVersion is the version stamped into a saved layout. A layout from
// a different version is refused rather than half-applied, so an app that
// grew new panels falls back to its default arrangement instead of coming
// up wrong.
const LayoutVersion = 1

// Layout is a whole dock arrangement, ready for encoding/json: which
// panels are in which area, how they are split and in what order they are
// tabbed, the shares the sashes were left at, which panels are closed, and
// where the floating ones are.
//
// Panels are named by their component name ([Panel.SetName]), so an app
// that renames a panel's title keeps its place.
type Layout struct {
	Version int `json:"version"`
	// Areas is the tree of each side, keyed "left", "right", "top" and
	// "bottom"; a side with no panels is left out.
	Areas map[string]*LayoutNode `json:"areas,omitempty"`
	// Weights is how the host shares its box out, in the order
	// top, middle, bottom and then left, centre, right.
	Weights *LayoutFrame `json:"weights,omitempty"`
	// Floating are the panels in windows of their own.
	Floating []LayoutFloat `json:"floating,omitempty"`
	// Closed are the panels that are hidden.
	Closed []string `json:"closed,omitempty"`
	// Collapsed are the panels showing only their title bar.
	Collapsed []string `json:"collapsed,omitempty"`
}

// LayoutFrame is how the host's own skeleton shares its space out.
type LayoutFrame struct {
	// Rows is the share of the height taken by the top area, the middle
	// and the bottom area.
	Rows [3]float32 `json:"rows"`
	// Cols is the share of the middle's width taken by the left area, the
	// centre and the right area.
	Cols [3]float32 `json:"cols"`
}

// LayoutNode is one node of an area's tree: a split with children, or a
// stack of panel names.
type LayoutNode struct {
	// Split is "h" for panes side by side, "v" for one above the other,
	// and empty for a stack.
	Split string `json:"split,omitempty"`
	// Children are a split's panes, with Weights their shares.
	Children []*LayoutNode `json:"children,omitempty"`
	Weights  []float32     `json:"weights,omitempty"`
	// Panels are a stack's panels, in tab order, and Current the one on
	// top.
	Panels  []string `json:"panels,omitempty"`
	Current int      `json:"current,omitempty"`
}

// LayoutFloat is one floating panel's window. The position is best effort:
// a Wayland client is not told where its windows are and cannot place
// them, so there only the size comes back.
type LayoutFloat struct {
	Panel string  `json:"panel"`
	X     float32 `json:"x"`
	Y     float32 `json:"y"`
	W     float32 `json:"w"`
	H     float32 `json:"h"`
}

// SaveLayout is the host's arrangement right now.
func (h *Host) SaveLayout() Layout {
	l := Layout{Version: LayoutVersion, Areas: map[string]*LayoutNode{}}
	for s := range h.areas {
		if n := h.saveNode(h.areas[s]); n != nil {
			l.Areas[Side(s).String()] = n
		}
	}
	l.Weights = &LayoutFrame{
		Rows: shares(h.root.weights),
		Cols: shares(h.middle.weights),
	}
	for _, p := range h.panels {
		if p.closed {
			l.Closed = append(l.Closed, p.Name())
		}
		if p.collapsed {
			l.Collapsed = append(l.Collapsed, p.Name())
		}
		if p.Floating() {
			g := p.FloatGeometry()
			l.Floating = append(l.Floating, LayoutFloat{
				Panel: p.Name(), X: g.Min.X, Y: g.Min.Y, W: g.Dx(), H: g.Dy(),
			})
		}
	}
	return l
}

// shares copies three weights out of a skeleton split, padding a short
// slice with ones so a malformed layout cannot panic.
func shares(w []float32) [3]float32 {
	var out [3]float32
	for i := range out {
		if i < len(w) && w[i] > 0 {
			out[i] = w[i]
		} else {
			out[i] = 1
		}
	}
	return out
}

// saveNode writes a node out, or nil when it holds no panel.
func (h *Host) saveNode(n Node) *LayoutNode {
	switch v := n.(type) {
	case *Stack:
		if len(v.panels) == 0 {
			return nil
		}
		out := &LayoutNode{Current: v.current}
		for _, p := range v.panels {
			out.Panels = append(out.Panels, p.Name())
		}
		return out
	case *Split:
		out := &LayoutNode{Split: "h"}
		if v.vertical {
			out.Split = "v"
		}
		for i, k := range v.kids {
			child := h.saveNode(k)
			if child == nil {
				continue
			}
			out.Children = append(out.Children, child)
			out.Weights = append(out.Weights, v.weights[i])
		}
		if len(out.Children) == 0 {
			return nil
		}
		if len(out.Children) == 1 && out.Children[0].Split == "" {
			// A split of one is no split: write the stack straight out, so
			// the file says what the user sees.
			return out.Children[0]
		}
		return out
	}
	return nil
}

// LayoutJSON is the host's arrangement as JSON, ready to write to a file.
func (h *Host) LayoutJSON() ([]byte, error) { return json.Marshal(h.SaveLayout()) }

// ApplyLayoutJSON reads an arrangement back. A layout that names a panel
// the host does not have, or that leaves one out, still applies: unknown
// names are skipped and panels the layout forgot are docked where they are
// now, so an app that gained a panel since the file was written still
// comes up with all of them.
func (h *Host) ApplyLayoutJSON(b []byte) error {
	var l Layout
	if err := json.Unmarshal(b, &l); err != nil {
		return fmt.Errorf("dock: reading the layout: %w", err)
	}
	return h.ApplyLayout(l)
}

// ErrLayoutVersion is returned for a layout written by another version of
// the format. An app that gets it should fall back to [Host.ResetLayout].
var ErrLayoutVersion = errors.New("dock: the saved layout is from another version")

// ApplyLayout rebuilds the arrangement from l.
func (h *Host) ApplyLayout(l Layout) error {
	if l.Version != LayoutVersion {
		return ErrLayoutVersion
	}
	h.cancelDrag()
	// Every panel comes out of the tree first, so a layout can move them
	// anywhere without tripping over where they were.
	seen := map[string]*Panel{}
	for _, p := range h.panels {
		seen[p.Name()] = p
		if p.Floating() {
			h.dockBack(p, func() {})
		}
	}
	for _, st := range h.stacks() {
		for _, p := range append([]*Panel(nil), st.panels...) {
			st.removePanel(p)
		}
	}
	for s := range h.areas {
		for _, k := range append([]Node(nil), h.areas[s].kids...) {
			h.areas[s].drop(k)
		}
	}
	placed := map[string]bool{}
	for s := range h.areas {
		node := l.Areas[Side(s).String()]
		if node == nil {
			continue
		}
		built := h.buildNode(node, Side(s), seen, placed)
		if built == nil {
			continue
		}
		if sp, ok := built.(*Split); ok && sp.vertical == h.areas[s].vertical {
			// The area is already a split that way, so take its panes
			// rather than nesting a split inside an identical one — which
			// would make a saved layout come back one level deeper each
			// time it went round.
			kids := append([]Node(nil), sp.kids...)
			weights := append([]float32(nil), sp.weights...)
			for _, k := range kids {
				sp.drop(k)
			}
			for i, k := range kids {
				h.areas[s].insert(len(h.areas[s].kids), k, weights[i])
			}
			continue
		}
		h.areas[s].insert(len(h.areas[s].kids), built, 1)
	}
	// Panels the layout never mentioned go back to the side they were on,
	// so a new panel is not lost when an old file is read.
	for _, p := range h.panels {
		if placed[p.Name()] {
			continue
		}
		area := h.areas[sideIndex(p.homeSide)]
		area.insert(len(area.kids), NewStack(p), 1)
		placed[p.Name()] = true
	}
	if l.Weights != nil {
		h.root.weights = l.Weights.Rows[:]
		h.middle.weights = l.Weights.Cols[:]
	}
	// Flags, then the floating windows, which need the panels to be in
	// the tree first so floating can take them out of it again.
	closed := map[string]bool{}
	for _, name := range l.Closed {
		closed[name] = true
	}
	collapsed := map[string]bool{}
	for _, name := range l.Collapsed {
		collapsed[name] = true
	}
	for _, p := range h.panels {
		p.collapsed = collapsed[p.Name()]
		p.closed = false
		if closed[p.Name()] {
			p.closed = true
		}
		if st := p.Stack(); st != nil {
			st.pickOpen()
			st.sync()
		}
	}
	for _, f := range l.Floating {
		p := seen[f.Panel]
		if p == nil {
			continue
		}
		g := paintengine2d.XYWH(f.X, f.Y, f.W, f.H)
		p.geom = g
		h.FloatPanel(p, g)
	}
	h.after()
	return nil
}

// buildNode makes the tree l describes, taking panels out of seen.
func (h *Host) buildNode(l *LayoutNode, side Side, seen map[string]*Panel, placed map[string]bool) Node {
	if l == nil {
		return nil
	}
	if l.Split == "" {
		var panels []*Panel
		for _, name := range l.Panels {
			p := seen[name]
			if p == nil || placed[name] {
				continue
			}
			placed[name] = true
			p.homeSide = side
			panels = append(panels, p)
		}
		if len(panels) == 0 {
			return nil
		}
		st := NewStack(panels...)
		if l.Current >= 0 && l.Current < len(st.panels) {
			st.current = l.Current
		}
		st.pickOpen()
		st.sync()
		return st
	}
	sp := NewSplit(l.Split == "v")
	for i, child := range l.Children {
		built := h.buildNode(child, side, seen, placed)
		if built == nil {
			continue
		}
		w := float32(1)
		if i < len(l.Weights) && l.Weights[i] > 0 {
			w = l.Weights[i]
		}
		sp.insert(len(sp.kids), built, w)
	}
	switch len(sp.kids) {
	case 0:
		return nil
	case 1:
		// Nothing to split: hand back the one pane that survived.
		only := sp.kids[0]
		sp.drop(only)
		return only
	}
	return sp
}

// SetDefaultLayout remembers the arrangement as it is now as the one
// [Host.ResetLayout] goes back to. An app calls it once, after it has
// built its panels and before it reads any saved layout.
func (h *Host) SetDefaultLayout() {
	l := h.SaveLayout()
	h.def = &l
}

// ResetLayout puts the arrangement back to the one
// [Host.SetDefaultLayout] recorded. It reports whether there was one.
func (h *Host) ResetLayout() bool {
	if h.def == nil {
		return false
	}
	return h.ApplyLayout(*h.def) == nil
}
