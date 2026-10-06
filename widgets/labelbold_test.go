package widgets

import (
	"testing"

	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A run-in heading is bold at the body's size — GTK's `heading`, Qt's
// QFont::setBold, HTML's <strong>. A label could be plain, Mono, or Title,
// and Title is the *page* title's size, a step above the section titles the
// heading would sit under. So a mail client naming each hop of a route over
// the words about it drew the line itself with the look's BoldFont and wrapped
// it by hand: a label reimplemented to change one thing about the face.
func TestLabelBoldIsTheBodySizeInTheLooksBoldFace(t *testing.T) {
	lk := style.DarkLook()
	plain, bold, title := NewLabel("From your mail server to theirs"), NewLabel("From your mail server to theirs"), NewLabel("From your mail server to theirs")
	bold.Bold = true
	title.Title = true
	for _, l := range []*Label{plain, bold, title} {
		l.SetHost(&fakeWindow{look: lk})
	}
	if got, want := bold.font(), lk.BoldFont(); got != want {
		t.Errorf("a bold label takes %v, want the look's bold face", got)
	}
	// The body's *size*, which is the whole point of it rather than Title.
	if bold.font().Height() != plain.font().Height() {
		t.Errorf("bold is %v tall and the body is %v: a run-in heading is the body's size",
			bold.font().Height(), plain.font().Height())
	}
	if title.font().Height() <= plain.font().Height() {
		t.Skip("this look's title face is not larger, so the distinction cannot be shown")
	}
	if bold.font().Height() == title.font().Height() {
		t.Error("bold came out at the title's size")
	}
}

// Title wins where both are set: a title is already the louder of the two.
func TestLabelTitleBeatsBold(t *testing.T) {
	lk := style.DarkLook()
	l := NewLabel("A page title")
	l.Title, l.Bold = true, true
	l.SetHost(&fakeWindow{look: lk})
	if got, want := l.font(), lk.TitleFont(); got != want {
		t.Errorf("Title+Bold took %v, want the title face", got)
	}
}

// And a bold label is a label: it wraps, and it measures to what it wraps to.
func TestABoldLabelWrapsLikeAnyOther(t *testing.T) {
	lk := style.DarkLook()
	l := NewLabel("From your mail server to theirs, and on to whoever reads it next")
	l.Bold, l.Wrap = true, true
	l.SetHost(&fakeWindow{look: lk})
	wide := l.Measure(layout.Loose(600, 400))
	narrow := l.Measure(layout.Loose(160, 400))
	if narrow.Y <= wide.Y {
		t.Errorf("a bold label measured %v tall at 160 and %v at 600: it did not wrap", narrow.Y, wide.Y)
	}
}
