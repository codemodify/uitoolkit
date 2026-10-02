package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// What secure input is doing can be asked for, rather than inferred from
// what SetSecureInput answered once.
//
// On X11 the grab follows the focus: dropped at a focus-out, retaken at
// the focus-in, and a retake can fail. Nothing reported any of that, so an
// application could not tell a person "the keyboard is not held" without
// being wrong on X11 every time.
func TestSecureInputReportsWhatIsInEffect(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("passphrase"))
	a.PumpOnce()

	var changes []bool
	w.OnSecureInput(func(held bool) { changes = append(changes, held) })

	// Asking is recorded whether or not the backend can do it: an
	// offscreen surface holds no keyboard, and saying so is the point.
	w.SetSecureInput(true)
	if !w.secureWanted {
		t.Fatal("the request was not recorded")
	}
	// An offscreen surface records the request and reports success, so
	// it reads as held: what SecureInputHeld answers is the backend's
	// own account of itself, not a guess from the request.
	if !w.SecureInputHeld() {
		t.Error("a surface that said it took the request does not report it held")
	}
	if len(changes) != 1 || !changes[0] {
		t.Errorf("the watcher heard %v, want one change to held", changes)
	}

	// The focus events are where it can change without the application
	// asking, so they are where a watcher has to hear about it.
	w.dispatch(platform.Event{Kind: platform.EventFocusOut})
	w.dispatch(platform.Event{Kind: platform.EventFocusIn})

	w.SetSecureInput(false)
	if w.secureWanted {
		t.Error("turning it off did not clear the request")
	}
	// And it falls out of effect when the request is dropped.
	if w.SecureInputHeld() {
		t.Error("it still reports held after being turned off")
	}
	if len(changes) < 2 || changes[len(changes)-1] {
		t.Errorf("the watcher heard %v, want a final change to not-held", changes)
	}
}
