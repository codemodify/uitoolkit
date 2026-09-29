package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/widget"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// The three helpers drive a headless window the way a keyboard and a
// mouse would, and each has run a frame by the time it returns.
func TestDriveAHeadlessWindow(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 420, Height: 220, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	field := widgets.NewTextField("", "Name", nil)
	clicked := 0
	ok := widgets.NewButton("OK", func() { clicked++ })
	submitted := ""
	field.OnSubmit = func(s string) { submitted = s }
	w.SetContent(widgets.NewColumn(field, ok))
	a.PumpOnce()

	w.FocusOn(field)
	if w.Focus() != widget.Component(field) {
		t.Fatalf("FocusOn left the focus on %T", w.Focus())
	}
	w.Type("correct horse")
	if field.Text != "correct horse" {
		t.Fatalf("typed %q", field.Text)
	}
	// Modifiers combine, and the field's own handling runs.
	w.Press(platform.KeyA, platform.ModCtrl)
	if a, b := field.Selection(); a != 0 || b != len("correct horse") {
		t.Fatalf("Ctrl+A selected %d..%d", a, b)
	}
	w.Press(platform.KeyReturn)
	if submitted != "correct horse" {
		t.Fatalf("Return submitted %q", submitted)
	}
	if !w.ClickComponent(ok) {
		t.Fatal("the button had no box to click")
	}
	if clicked != 1 {
		t.Fatalf("%d clicks", clicked)
	}
}

// A widget with no box is not clicked, and says so rather than sending a
// press into the corner of the window.
func TestClickComponentNeedsABox(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("host"))
	a.PumpOnce()
	loose := widgets.NewButton("Nowhere", nil)
	if w.ClickComponent(loose) {
		t.Fatal("clicked a widget that was never laid out")
	}
	if w.ClickComponent(nil) {
		t.Fatal("clicked nil")
	}
}

// The helpers on a closed window do nothing and do not panic.
func TestDriveAClosedWindow(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("host"))
	a.PumpOnce()
	w.Close()
	w.Type("x")
	w.Press(platform.KeyReturn)
	w.FocusOn(nil)
	if w.ClickAt(paintengine2d.Pt(5, 5)) {
		t.Fatal("a closed window took a click")
	}
}
