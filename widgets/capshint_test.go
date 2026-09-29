package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// A right passphrase refused is nearly always Caps Lock, and the field
// cannot work that out for itself without reading the secret. The state
// reaches it through LockKeysWatcher, so it needs no application code.
func TestSecretFieldIsALockKeysWatcher(t *testing.T) {
	var _ widget.LockKeysWatcher = (*SecretField)(nil)
}

func TestSecretFieldCapsHint(t *testing.T) {
	f := NewSecretField("Passphrase")
	if !f.CapsHint {
		t.Fatal("CapsHint should be on by default")
	}
	h := atScale(t, f, 1)
	sz := f.Measure(layout.Constraints{MaxW: 240, MaxH: -1})
	f.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))

	// Nothing heard yet, nothing shown.
	if !f.capsRect().Empty() {
		t.Error("the mark is drawn before anything said Caps Lock was on")
	}

	f.LockKeysChanged(true, false)
	if !f.CapsLockOn() {
		t.Error("CapsLockOn is false after the window said it was on")
	}
	// Focus is the other half: an unfocused field is not where the
	// passphrase is being typed.
	if !f.capsRect().Empty() {
		t.Error("the mark is drawn on a field that does not have the focus")
	}

	h.RequestFocus(f)
	if f.capsRect().Empty() {
		t.Error("a focused field with Caps Lock on draws no mark")
	}
	if r := f.capsRect(); r.Max.X > f.LocalBounds().Max.X {
		t.Errorf("the mark at %v runs past the field's %v", r, f.LocalBounds())
	}

	f.LockKeysChanged(false, false)
	if !f.capsRect().Empty() {
		t.Error("the mark stayed after Caps Lock went off")
	}

	// And it can be turned off.
	f.LockKeysChanged(true, false)
	f.CapsHint = false
	if !f.capsRect().Empty() {
		t.Error("CapsHint=false still draws the mark")
	}
}

// Num Lock is not Caps Lock: a field that showed a warning for it would
// be warning about nothing.
func TestSecretFieldIgnoresNumLock(t *testing.T) {
	f := NewSecretField("")
	h := atScale(t, f, 1)
	h.RequestFocus(f)
	f.LockKeysChanged(false, true)
	if f.CapsLockOn() || !f.capsRect().Empty() {
		t.Error("Num Lock turned the Caps Lock mark on")
	}
}
