package widget

import "github.com/codemodify/uitoolkit/style"

// What a component has to say about moving, and who it inherits that from.
//
// The user's reduce-motion preference governs everything by default, and
// that is right for the motion that decorates — a hover fading in, a
// scroll bar fading out. It is wrong for the motion that *is* the thing:
// a card turning over to show it has a back, a window flipping to its
// settings. [style.Motion] is how an application says which is which, and
// this is how a widget reads the answer.

// stater is a component, or a host, that has a motion of its own. Every
// component built on [Base] has one, so the assertion is a formality for
// the toolkit's own widgets and a courtesy to anything hand-rolled.
type stater interface{ Motion() style.Motion }

// MotionOf is what c has to say about moving: its own [style.Motion] if
// it states one, otherwise the nearest ancestor's, otherwise its window's
// ([app.Window.SetMotion]), otherwise [style.MotionAuto].
//
// Inheritance is the point. A dialog that is itself the motion — one that
// slides a step into view, or turns a card over — says so once on the
// dialog, and every widget inside it is covered without being told.
func MotionOf(c Component) style.Motion {
	for ; c != nil; c = c.Parent() {
		if s, ok := c.(stater); ok {
			if m := s.Motion(); m != style.MotionAuto {
				return m
			}
		}
		if p := c.Parent(); p == nil {
			// The root of this tree: the window has the last word, which
			// is how one call covers a window's content, its cover, its
			// toasts and the dialogs it opens alike — they are separate
			// trees with the same host.
			if h, ok := c.Host().(stater); ok {
				return h.Motion()
			}
		}
	}
	return style.MotionAuto
}

// Animates reports whether c should draw a change as motion rather than
// at once: [style.AnimationsWith] of [MotionOf]. It is what a widget with
// something to animate asks, in place of [style.Animations].
func Animates(c Component) bool { return style.AnimationsWith(MotionOf(c)) }
