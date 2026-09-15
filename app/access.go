package app

import (
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/widget"
)

// AccessibleTree is the window's accessibility tree: the window, its
// content, then whatever floats over it (a dialog, an open menu, a
// tooltip). Platform adapters ask for it when assistive technology does;
// nothing is built otherwise.
func (w *Window) AccessibleTree() *a11y.Node {
	root := &a11y.Node{Role: a11y.RoleWindow, Name: w.Title()}
	if w.root != nil {
		root.ID = 1<<63 | w.root.ID() // no component or item has bit 63
	}
	if w.Active() {
		root.State |= a11y.StateFocused
	}
	for _, c := range []widget.Component{w.root, w.overlay, w.popup, w.tooltip} {
		widget.AccessibleTree(root, c)
	}
	return root
}

// AccessibleAction performs an assistive technology's action on the node
// with the given ID (a component, or an item of a view); it reports
// whether anything did it.
func (w *Window) AccessibleAction(id uint64, a a11y.Action) bool {
	comp, item := widget.SplitItemID(id)
	var hit widget.Component
	for _, c := range []widget.Component{w.root, w.overlay, w.popup, w.tooltip} {
		if c == nil {
			continue
		}
		widget.Walk(c, func(x widget.Component) {
			if hit == nil && x.ID() == comp {
				hit = x
			}
		})
	}
	if hit == nil {
		return false
	}
	if a == a11y.ActionFocus && item < 0 {
		if !hit.WantsFocus() || !hit.Enabled() {
			return false
		}
		w.RequestFocus(hit)
		return true
	}
	if act, ok := hit.(widget.AccessibleActor); ok {
		return act.AccessibleAction(item, a)
	}
	return false
}
