package widget

import (
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
)

// Accessible is implemented by components that assistive technology sees.
// Describe fills n: the builder has already set its ID and bounds, the
// focusable, focused and disabled states, and any name set with
// SetAccessibleName (which Describe should keep). Components that do not
// implement it are transparent: their accessible descendants join the
// nearest accessible ancestor.
type Accessible interface {
	Describe(n *a11y.Node)
}

// AccessibleItems is implemented by views whose items are not components
// (list rows, tree nodes, table rows, tabs, tools, menu items): it returns
// their nodes, with IDs from ItemID.
type AccessibleItems interface {
	AccessibleItems() []*a11y.Node
}

// AccessibleActor performs an assistive technology's action on the
// component (item < 0) or on one of its items; it reports whether it did.
type AccessibleActor interface {
	AccessibleAction(item int, a a11y.Action) bool
}

// AccessibleFocusItem is implemented by views whose keyboard focus sits on
// one of their items (the current row, the selected tab): the item's
// index as in AccessibleItems, or -1.
type AccessibleFocusItem interface {
	AccessibleFocusItem() int
}

// FocusID is the accessible ID of the object with the keyboard focus
// when c has it: c's current item, the accessible leaf that contains c
// (a spin button's field), or c itself.
func FocusID(c Component) uint64 {
	if c == nil {
		return 0
	}
	if fi, ok := c.(AccessibleFocusItem); ok {
		if i := fi.AccessibleFocusItem(); i >= 0 {
			return ItemID(c, i)
		}
	}
	for p := c.Parent(); p != nil; p = p.Parent() {
		if leaf, ok := p.(interface{ AccessibleLeaf() bool }); ok && leaf.AccessibleLeaf() {
			return p.ID()
		}
	}
	return c.ID()
}

// AccessibleEditor is implemented by editable text components: assistive
// technology (and automation such as dogtail) replaces their text.
type AccessibleEditor interface {
	AccessibleSetText(s string) bool
}

// itemBits is how many low ID bits number a view's items.
const itemBits = 24

// ItemID is the stable accessible ID of item i of view c.
func ItemID(c Component, i int) uint64 {
	return c.ID()<<itemBits | uint64(i+1)&(1<<itemBits-1)
}

// SplitItemID reverses ItemID: the view's component ID and the item
// index, or item -1 when id names a component.
func SplitItemID(id uint64) (comp uint64, item int) {
	if low := id & (1<<itemBits - 1); id>>itemBits != 0 && low != 0 {
		return id >> itemBits, int(low) - 1
	}
	return id, -1
}

// AccessibleName is a component's name as set with SetAccessibleName.
func (b *Base) AccessibleName() string {
	if b.acc == nil {
		return ""
	}
	return b.acc.name
}

// SetAccessibleName names a component for assistive technology when its
// own text does not say what it is: an icon-only button, a field whose
// label is a separate widget. Form rows name their fields this way.
func (b *Base) SetAccessibleName(name string) {
	if b.acc == nil {
		b.acc = &accLabel{}
	}
	b.acc.name = name
}

// AccessibleDescription adds to the name (a tooltip's text).
func (b *Base) AccessibleDescription() string {
	if b.acc == nil {
		return ""
	}
	return b.acc.desc
}

// SetAccessibleDescription sets the description.
func (b *Base) SetAccessibleDescription(d string) {
	if b.acc == nil {
		b.acc = &accLabel{}
	}
	b.acc.desc = d
}

type accessibleLabel interface {
	AccessibleName() string
	AccessibleDescription() string
}

// AccessibleTree builds the accessibility tree of the components under
// root, as children of parent.
func AccessibleTree(parent *a11y.Node, root Component) {
	build(parent, root)
}

// describe is c's own node (no children), or nil when c is not
// accessible.
func describe(c Component) *a11y.Node {
	acc, ok := c.(Accessible)
	if !ok {
		return nil
	}
	n := &a11y.Node{ID: c.ID(), Bounds: DeviceBounds(c)}
	if c.WantsFocus() {
		n.State |= a11y.StateFocusable
	}
	if h := c.Host(); h != nil && h.Focus() == c {
		n.State |= a11y.StateFocused
	}
	if !c.Enabled() {
		n.State |= a11y.StateDisabled
	}
	var named string
	if l, ok := c.(accessibleLabel); ok {
		named = l.AccessibleName()
		n.Name, n.Description = named, l.AccessibleDescription()
	}
	acc.Describe(n)
	if named != "" {
		n.Name = named
	}
	return n
}

// maxFocusItems caps the view size whose items FocusNode lists to find
// the focused one each frame.
const maxFocusItems = 4096

// FocusNode describes the object with the keyboard focus when c has it
// (see FocusID), without building the rest of the tree; nil when it
// cannot say cheaply. Adapters compare it frame to frame to announce what
// changed (a check box ticked, a branch opened).
func FocusNode(c Component) *a11y.Node {
	if c == nil {
		return nil
	}
	id := FocusID(c)
	comp, item := SplitItemID(id)
	for p := c; p != nil; p = p.Parent() {
		if p.ID() != comp {
			continue
		}
		if item < 0 {
			return describe(p)
		}
		items, ok := p.(AccessibleItems)
		if !ok {
			return nil
		}
		if fi, ok := p.(interface{ AccessibleItemCount() int }); ok && fi.AccessibleItemCount() > maxFocusItems {
			return nil
		}
		var hit *a11y.Node
		for _, n := range items.AccessibleItems() {
			n.Walk(func(x *a11y.Node) bool {
				if x.ID == id {
					hit = x
				}
				return hit == nil
			})
		}
		return hit
	}
	return nil
}

func build(parent *a11y.Node, c Component) {
	if c == nil || !c.Visible() {
		return
	}
	target := parent
	if n := describe(c); n != nil {
		if items, ok := c.(AccessibleItems); ok {
			for _, it := range items.AccessibleItems() {
				if !c.Enabled() {
					it.State |= a11y.StateDisabled
				}
				n.Children = append(n.Children, it)
			}
		}
		parent.Children = append(parent.Children, n)
		target = n
		if leaf, ok := c.(interface{ AccessibleLeaf() bool }); ok && leaf.AccessibleLeaf() {
			return // its parts are not separate objects (a spin button's field)
		}
	}
	for _, ch := range c.Children() {
		build(target, ch)
	}
}

// LocalToWindow moves a rect from c's local space to window coordinates.
func LocalToWindow(c Component, r paintengine2d.Rect) paintengine2d.Rect {
	return r.Translate(DeviceBounds(c).Min)
}

// PlainText strips a label's mnemonic marker ("&Save" → "Save", "&&" → "&").
func PlainText(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '&' {
			if i+1 < len(s) && s[i+1] == '&' {
				b.WriteByte('&')
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
