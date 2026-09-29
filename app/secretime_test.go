package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// imeOn is what the window last told the surface.
func imeOn(t *testing.T, w *Window) bool {
	t.Helper()
	o, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Fatalf("surface %T is not offscreen", w.surf)
	}
	_, _, _, _, on := o.IMECursor()
	return on
}

// An input method is another process that sees every keystroke and
// learns from it. It gets an ordinary field, and it does not get a
// passphrase — not a masked preedit, nothing at all.
func TestInputMethodIsOffForSecretFields(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	plain := widgets.NewTextField("", "Name", nil)
	pass := widgets.NewPasswordField("Password", nil)
	secret := widgets.NewSecretField("Passphrase")
	w.SetContent(widgets.NewColumn(plain, pass, secret))
	a.PumpOnce()

	for _, c := range []struct {
		what string
		who  widget.Component
		want bool
	}{
		{"an ordinary field", plain, true},
		{"a password field", pass, false},
		{"a secret field", secret, false},
		{"back to the ordinary field", plain, true},
	} {
		w.RequestFocus(c.who)
		a.PumpOnce()
		if got := imeOn(t, w); got != c.want {
			t.Fatalf("%s: input method on = %v, want %v", c.what, got, c.want)
		}
	}
}

// The marker is asked, not assumed: a text field answers for its own
// Password flag, and a plain one says no.
func TestIsSecretTarget(t *testing.T) {
	plain := widgets.NewTextField("", "", nil)
	if widget.IsSecretTarget(plain) {
		t.Fatal("a plain text field says it is secret")
	}
	plain.Password = true
	if !widget.IsSecretTarget(plain) {
		t.Fatal("a password field does not say it is secret")
	}
	if widget.IsSecretTarget(widgets.NewLabel("x")) {
		t.Fatal("a label says it is secret")
	}
}
