package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func dialogWindow(t *testing.T, opts platform.WindowOptions) (*Application, *Window, *platform.Offscreen) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	opts.Headless = true
	if opts.Width == 0 {
		opts.Width, opts.Height = 460, 220
	}
	w, err := a.NewWindow(opts)
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewSecretField("Passphrase"))
	a.PumpOnce()
	o, ok := w.surf.(*platform.Offscreen)
	if !ok {
		t.Fatalf("surface %T", w.surf)
	}
	return a, w, o
}

// The three options a prompt opens with reach the backend at creation,
// not a frame later: a dialog raised after it is mapped can be covered
// in that frame, and one centred after it is shown jumps.
func TestDialogOptionsApplyAtCreation(t *testing.T) {
	_, _, o := dialogWindow(t, platform.WindowOptions{
		Title: "Unlock vault", Role: platform.RoleDialog,
		Center: true, KeepAbove: true, Sizing: platform.SizingFixed,
	})
	if o.WindowRole() != platform.RoleDialog {
		t.Fatalf("role %v", o.WindowRole())
	}
	if !o.Centered() {
		t.Fatal("not centred")
	}
	// The request, not the answer: whether the desktop grants keep-above
	// is its business, and the simulated one says no until a test says
	// it stacks windows.
	if !o.KeepAboveRequested() {
		t.Fatal("keep-above was not asked for")
	}
}

// An ordinary window asks for none of them.
func TestOrdinaryWindowIsNotADialog(t *testing.T) {
	_, _, o := dialogWindow(t, platform.WindowOptions{Title: "Main"})
	if o.WindowRole() != platform.RoleNormal || o.Centered() || o.KeepAboveRequested() {
		t.Fatalf("role %v centred %v above %v", o.WindowRole(), o.Centered(), o.KeepAboveRequested())
	}
}

// Activate reaches the surface, and a window that becomes a dialog later
// can say so.
func TestActivateAndLateRole(t *testing.T) {
	_, w, o := dialogWindow(t, platform.WindowOptions{Title: "Main"})
	if !w.Activate() {
		t.Fatal("Activate refused")
	}
	if o.Activations() != 1 {
		t.Fatalf("%d activations", o.Activations())
	}
	if !w.SetWindowRole(platform.RoleDialog) || o.WindowRole() != platform.RoleDialog {
		t.Fatalf("late role %v", o.WindowRole())
	}
	if !w.Center() || !o.Centered() {
		t.Fatal("Center refused")
	}
}

// A closed window asks for nothing and says so.
func TestDialogCallsOnAClosedWindow(t *testing.T) {
	_, w, _ := dialogWindow(t, platform.WindowOptions{Title: "Main"})
	w.Close()
	if w.Activate() || w.Center() || w.SetWindowRole(platform.RoleDialog) {
		t.Fatal("a closed window answered yes")
	}
}
