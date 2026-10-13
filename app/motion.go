package app

import "github.com/codemodify/uitoolkit/style"

// SetMotion states what this window has to say about moving, over the
// user's reduce-motion preference: [style.MotionAlways] for a window
// whose motion carries meaning — one that turns over to its settings —
// [style.MotionNever] for one that should stay still, and
// [style.MotionAuto], the default, to leave it to the preference.
//
// It reaches everything the window holds: its content, the dialogs it
// opens, its cover and its notes, because they are separate trees with
// this window as their host and [widget.MotionOf] asks the host last. A
// component or a dialog that states its own overrides it, which is how
// one window animates its flip and nothing else:
//
//	win.SetMotion(style.MotionAuto)        // the preference, for the window
//	win.FlipPanel().SetMotion(style.MotionAlways)  // except the turn
//
// **This is not a way around the preference.** [style.Motion] says what
// it is for: marking the motion that would leave a hole in the meaning
// if it went. A window that marks everything Always has turned the
// preference off for the person who asked for it.
func (w *Window) SetMotion(m style.Motion) {
	if w == nil || w.motion == m {
		return
	}
	w.motion = m
	w.fullInvalidate()
}

// Motion is what this window states about moving. It is also what
// [widget.MotionOf] reads for a component that states nothing itself and
// has no ancestor that does.
func (w *Window) Motion() style.Motion {
	if w == nil {
		return style.MotionAuto
	}
	return w.motion
}
