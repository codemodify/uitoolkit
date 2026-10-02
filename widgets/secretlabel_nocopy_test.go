package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A masked SecretLabel draws a row of bullets. To do that it needs to know
// how many runes there were and whether there were any; it does not need
// the value, and it no longer keeps it.
//
// It used to copy the value in on every Measure and Paint whether it was
// revealed or not, so an application showing a vault item's fields masked
// held a second copy of every one of them for as long as the item was
// open, wiped only if it remembered to call Hide on each label on every
// way the item could go.
func TestAMaskedSecretLabelKeepsNoCopy(t *testing.T) {
	secret := []byte("correct horse battery staple")
	l := NewSecretLabel(func() []byte { return secret })
	l.SetLook(style.DarkLook())
	l.SetHost(&host{})

	paint := func() {
		l.Measure(layout.Unbounded())
		img := paintengine2d.NewImage(300, 40)
		l.Arrange(paintengine2d.XYWH(0, 0, 300, 40))
		l.Paint(paintengine2d.NewContext(img))
	}

	paint()
	if n := len(l.buf); n != 0 {
		t.Errorf("masked, the label is holding %d bytes of the secret", n)
	}
	// It still knows enough to lay itself out and to draw the right row.
	if got := l.Len(); got != len([]rune(string(secret))) {
		t.Errorf("Len is %d, want %d runes", got, len([]rune(string(secret))))
	}
	if l.mask() == "" {
		t.Error("a non-empty value drew no bullets")
	}

	// Revealed it must hold the value — that is the whole point — and it
	// must let go again the moment it is masked.
	l.Reveal = true
	paint()
	if string(l.buf) != string(secret) {
		t.Fatalf("revealed, the label did not hold the value: %q", l.buf)
	}
	held := l.buf[:len(secret)]
	l.Reveal = false
	paint()
	if n := len(l.buf); n != 0 {
		t.Errorf("after Reveal went off the label still holds %d bytes", n)
	}
	for i, b := range held[:cap(held)][:len(secret)] {
		if b != 0 {
			t.Fatalf("the buffer was emptied but not zeroed: byte %d is %q", i, b)
		}
	}

	// And a label taken out of its window has zeroed what it held.
	l.Reveal = true
	paint()
	kept := l.buf[:len(secret)]
	l.SetHost(nil)
	for i, b := range kept {
		if b != 0 {
			t.Fatalf("a label removed from the tree kept byte %d: %q", i, b)
		}
	}
	if n := len(l.buf); n != 0 {
		t.Errorf("a removed label still holds %d bytes", n)
	}
	// Len is not checked here: it asks Value again by contract, so a
	// removed label that is still asked will answer about the value the
	// application still has. What must not survive is the label's copy.
}
