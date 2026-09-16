package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widgets"
)

// Asking the user what a drop should do.
//
// A drag that offers more than one action can leave the choice to the
// user: XDND's XdndActionAsk, and on Wayland the ask action a target
// names as the one it would rather have. Both mean the same thing — put
// the Copy / Move / Link menu under the pointer at the drop, the way
// every file manager does — and the answer goes back to the source as the
// action the drop performed.
//
// Holding a modifier settles the question instead: Shift moves, Ctrl
// copies, both link. The X11 source sends that action rather than Ask and
// a Wayland compositor answers with it rather than ask, so a drop with a
// modifier held never sees the menu at all. Nothing here has to know that
// — it only ever sees the answer.

// dropAskLabels is what each action is called in the menu, in the order a
// file manager lists them.
var dropAskLabels = []struct {
	action platform.DragAction
	text   string
}{
	{platform.DragCopy, "Copy Here"},
	{platform.DragMove, "Move Here"},
	{platform.DragLink, "Link Here"},
}

// wantsDropAsk reports that the user should be offered the choice for
// this drag: its source is willing to be asked, and there is more than
// one action to choose between. It is what makes the window name ask
// among the actions it allows, which on Wayland is the only way the
// choice can ever be offered.
func wantsDropAsk(ev platform.Event, allowed platform.DragAction) bool {
	return (ev.Actions.Asks() || ev.Action.Asks()) && askDropActions(ev, allowed) != platform.DragNone
}

// askDropActions is the set a drop would put to the user: the actions
// both sides allow, with the question itself taken out. DragNone when
// there is nothing to ask — one action, or none.
func askDropActions(ev platform.Event, allowed platform.DragAction) platform.DragAction {
	both := (offeredActions(ev) & allowed).Actions()
	if both == both.One() {
		return platform.DragNone
	}
	return both
}

// dropAsks reports that this drop is the one to ask about: the source
// asked for the choice (XDND), or the compositor settled on ask
// (Wayland), and there is more than one answer.
func dropAsks(ev platform.Event, allowed platform.DragAction) platform.DragAction {
	if !ev.Action.Asks() {
		return platform.DragNone
	}
	return askDropActions(ev, allowed)
}

// askDropAction puts the choice to the user at the drop point and calls
// done with what they picked — DragNone when they dismissed the menu,
// which cancels the drop. It reports whether the menu is up; a window
// that cannot show one answers false and the caller drops without asking,
// since losing the drop would be worse than not asking.
func (w *Window) askDropAction(at paintengine2d.Point, choices platform.DragAction, done func(platform.DragAction)) bool {
	anchor := w.Content()
	if anchor == nil || done == nil {
		return false
	}
	var items []*widgets.MenuItem
	of := map[*widgets.MenuItem]platform.DragAction{}
	for _, l := range dropAskLabels {
		if choices.Has(l.action) {
			it := widgets.Item(l.text, nil)
			of[it] = l.action
			items = append(items, it)
		}
	}
	if len(items) < 2 {
		return false
	}
	cancel := widgets.Item("Cancel", nil)
	items = append(items, widgets.Sep(), cancel)
	pop := widgets.ShowContextMenu(anchor, at, items...)
	if pop == nil {
		return false
	}
	// The choice comes through OnPick rather than an item's own OnClick:
	// a menu runs OnClick only after taking itself down, and taking this
	// one down is a refusal, so the refusal would always arrive first.
	settled := false
	pop.OnPick = func(it *widgets.MenuItem) {
		// Settled before the menu goes down, so its dismissal does not
		// answer for the user; down before the drop, so the data does
		// not land under a menu still painting over it.
		settled = true
		w.DismissPopup()
		done(of[it])
	}
	// Escape, a click outside, or the window losing the menu any other
	// way is a refusal: the source has to be told, or it waits for an
	// answer that never comes.
	pop.OnDismiss = func() {
		if settled {
			return
		}
		settled = true
		done(platform.DragNone)
	}
	return true
}
