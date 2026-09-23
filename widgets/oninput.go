package widgets

// OnInput is the second callback the toolkit's value controls carry beside
// OnChange, and userEdit is how each of them knows which it is looking at.
//
// OnChange fires on every change, whoever made it. That is what a view bound
// to a model wants and it is why it stays. It is also a trap for the app
// driving a control *from* a model — a player moving its seek bar from the
// playback position, a settings page filling a form from what was saved —
// because the control then calls the app's own handler back with the value
// the app has just handed it, and the app either re-enters its own code or
// grows a suppress flag of its own. Every toolkit meets this somewhere: Qt
// has editingFinished beside valueChanged, and blockSignals; GTK has the
// freeze pair; WPF has UpdateSourceTrigger.
//
// OnInput is the other half of the answer. It fires only when the change
// came from the user — the pointer, the keyboard, or an accessibility action
// a screen reader ran — and never when the app set the value itself. An app
// driving a control from a data source listens to OnInput and needs no flag.
//
// The order is the same everywhere: **OnChange first, then OnInput.** Both
// see the control already holding the new value, so a handler that reads the
// widget rather than its argument gets the same answer from either.
//
// The mark nests safely. A control that drives another one as part of the
// user's edit — a spin button writing its inner field — sets its own flag
// and not the other's, so each control answers for its own change.
type userEdit struct{ on bool }

// did runs set with the change marked as the user's own.
func (u *userEdit) did(set func()) {
	was := u.on
	u.on = true
	set()
	u.on = was
}

// is reports whether the change being made is the user's.
func (u *userEdit) is() bool { return u.on }
