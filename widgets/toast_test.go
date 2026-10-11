package widgets

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A note is read, not used: nothing in it answers a press, so a press
// goes to whatever is under it. The window does not hit-test the layer
// either; this is the half that holds for a note put anywhere else.
func TestAToastIsNeverPressed(t *testing.T) {
	note := NewToast("Copied", ToastInfo)
	note.SetLook(style.DarkLook())
	note.SetHost(&host{})
	sz := note.Measure(layout.Loose(400, 100))
	note.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	for _, at := range []paintengine2d.Point{
		{X: 1, Y: 1},
		{X: sz.X * 0.5, Y: sz.Y * 0.5},
		{X: sz.X - 1, Y: sz.Y - 1},
	} {
		if h := note.HitTest(at); h != nil {
			t.Errorf("a press at %v landed on %T", at, h)
		}
	}
}

// The note is as wide as its words need and no wider, and long words
// wrap rather than running off the end.
func TestAToastWrapsItsWords(t *testing.T) {
	short := NewToast("Copied", ToastInfo)
	long := NewToast("The vault could not be unlocked: the passphrase was refused three times and the vault has locked itself for a minute.", ToastError)
	for _, n := range []*Toast{short, long} {
		n.SetLook(style.DarkLook())
		n.SetHost(&host{})
	}
	const room float32 = 420
	a := short.Measure(layout.Loose(room, 200))
	b := long.Measure(layout.Loose(room, 200))
	if a.X >= b.X {
		t.Errorf("the short note is %g wide and the long one %g", a.X, b.X)
	}
	if b.X > room {
		t.Errorf("the long note is %g wide, past the %g it was offered", b.X, room)
	}
	if b.Y <= a.Y {
		t.Errorf("the long note is %g tall and the short one %g: it did not wrap", b.Y, a.Y)
	}
}

// Each kind wears its own mark and its own ink, which is how a note says
// something went wrong without the words having to.
func TestEachKindOfNoteIsItsOwn(t *testing.T) {
	seen := map[style.ToolIcon]ToastKind{}
	for _, kind := range []ToastKind{ToastInfo, ToastSuccess, ToastWarning, ToastError} {
		n := NewToast("Something happened", kind)
		n.SetLook(style.DarkLook())
		n.SetHost(&host{})
		if n.Mark == style.IconNone {
			t.Errorf("kind %d has no mark", kind)
		}
		if prev, dup := seen[n.Mark]; dup {
			t.Errorf("kinds %d and %d wear the same mark", kind, prev)
		}
		seen[n.Mark] = kind
		if n.Timeout != ToastDuration {
			t.Errorf("kind %d starts with a %v timeout", kind, n.Timeout)
		}
	}
	// The three that are not the plain note are painted with their own
	// colour; the plain one takes the accent.
	p := style.DarkLook().Palette()
	for kind, want := range map[ToastKind]paintengine2d.Color{
		ToastInfo:    p.Accent,
		ToastSuccess: p.Success,
		ToastWarning: p.Warning,
		ToastError:   p.Danger,
	} {
		n := NewToast("x", kind)
		n.SetLook(style.DarkLook())
		if got := n.markInk(); got != want {
			t.Errorf("kind %d is painted %v, want %v", kind, got, want)
		}
	}
}

// A note made by hand is an ordinary widget: it has a timeout but
// nothing is running it, and it paints at once.
func TestANoteMadeByHandIsAtRest(t *testing.T) {
	n := NewToast("Copied", ToastInfo)
	if n.Opacity() != 1 {
		t.Errorf("a new note is at %g opacity", n.Opacity())
	}
	if n.Timeout != ToastDuration || ToastDuration < time.Second {
		t.Errorf("timeout %v", n.Timeout)
	}
}
