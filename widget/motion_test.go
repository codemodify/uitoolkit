package widget

import (
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// motionBox is a plain component: everything a widget gets from Base and
// nothing else.
type motionBox struct{ Base }

func newMotionBox() *motionBox {
	b := &motionBox{}
	b.Init(b)
	return b
}

// **A component's motion is its own, or the nearest ancestor's.**
//
// An application says "this dialog is the motion" once, on the dialog,
// and everything in it is covered. Reading the flag off the component
// alone would have made the caller set it on every widget that animates
// — which is every widget it does not own.
func TestMotionIsInheritedFromTheNearestAncestorThatStatesOne(t *testing.T) {
	root, mid, leaf := newMotionBox(), newMotionBox(), newMotionBox()
	root.Add(mid)
	mid.Add(leaf)

	if got := MotionOf(leaf); got != style.MotionAuto {
		t.Errorf("a tree that states nothing is %v, want MotionAuto", got)
	}
	root.SetMotion(style.MotionAlways)
	if got := MotionOf(leaf); got != style.MotionAlways {
		t.Errorf("the leaf is %v; the root said Always and nothing since has said otherwise", got)
	}
	// Nearest wins, so one widget inside an animated dialog can sit still.
	mid.SetMotion(style.MotionNever)
	if got := MotionOf(leaf); got != style.MotionNever {
		t.Errorf("the leaf is %v; the branch over it said Never", got)
	}
	if got := MotionOf(root); got != style.MotionAlways {
		t.Errorf("the root is %v; a child's word does not reach back up", got)
	}
}

// motionHost is a host with a motion of its own, as a window has.
type motionHost struct {
	Host
	m style.Motion
}

func (h *motionHost) Motion() style.Motion { return h.m }

// **The window has the last word, and the first tree to ask is whichever
// one the widget is in.** A window's cover, its notes and the dialogs it
// opens are separate trees with the same host, so a motion set on the
// window reaches all of them — there is no component they share.
func TestMotionFallsBackToTheHost(t *testing.T) {
	h := &motionHost{m: style.MotionAlways}
	root, leaf := newMotionBox(), newMotionBox()
	root.Add(leaf)
	root.SetHost(h)
	leaf.SetHost(h)

	if got := MotionOf(leaf); got != style.MotionAlways {
		t.Errorf("the leaf is %v; its window said Always and no component in between said anything", got)
	}
	// A component that states one is nearer than the window.
	leaf.SetMotion(style.MotionNever)
	if got := MotionOf(leaf); got != style.MotionNever {
		t.Errorf("the leaf is %v; its own word beats its window's", got)
	}
}

// **Animates is the question a widget actually asks**, and the one
// switch nothing overrides still overrides it: UITK_ANIMATIONS is how a
// screenshot or a test is made reproducible, so a component that marked
// itself Always cannot animate under it.
func TestAnimatesAnswersTheMotionAndTheEnvironmentBeatsIt(t *testing.T) {
	c := newMotionBox()
	t.Setenv(style.AnimationsEnv, "")
	style.SetReduceMotion(true)
	t.Cleanup(func() { style.SetReduceMotion(false) })

	if Animates(c) {
		t.Error("a component with nothing to say animated under the reduce-motion preference")
	}
	c.SetMotion(style.MotionAlways)
	if !Animates(c) {
		t.Error("a component that says its motion carries meaning did not animate")
	}
	t.Setenv(style.AnimationsEnv, "0")
	if Animates(c) {
		t.Error("UITK_ANIMATIONS=0 did not beat MotionAlways; screenshots stop being reproducible")
	}
	t.Setenv(style.AnimationsEnv, "1")
	c.SetMotion(style.MotionNever)
	if !Animates(c) {
		t.Error("UITK_ANIMATIONS=1 did not beat MotionNever")
	}
}
