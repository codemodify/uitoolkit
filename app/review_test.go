package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Appearance() has to describe the application as it is, because the
// ordinary way to change one setting is to read it, change that, and
// apply it back — a.ApplyAppearance(a.Appearance()), which the tour
// does. The look does not carry ComboWheel, so leaving it out of the
// reconstruction turned the user's choice off on the way past.
func TestAppearanceRoundTripKeepsComboWheel(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Look: style.DarkLook(), Headless: true})

	ap := a.Appearance()
	ap.ComboWheel = true
	a.ApplyAppearance(ap)
	if !style.ComboWheel() {
		t.Fatal("applying an appearance with ComboWheel did not enable it")
	}

	// Read it back and apply it unchanged: nothing may change.
	a.ApplyAppearance(a.Appearance())
	if !style.ComboWheel() {
		t.Fatal("a read-modify-apply round trip turned combo-wheel selection off")
	}
}

// A drag dies with the window that started it. The backend delivers the
// completion event to the source window, and a closed window leaves the
// application's list and stops being pumped — so the run never finished:
// app.drag stayed set, every later StartDrag was refused, and the
// payload and its Done callback were held for the life of the process.
func TestClosingADragSourceEndsTheDrag(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Look: style.DarkLook(), Headless: true})
	first, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}

	done := platform.DragAction(0)
	told := false
	d := &widget.Drag{
		Types:   []string{"text/plain"},
		Data:    func(string) ([]byte, bool) { return []byte("x"), true },
		Local:   true,
		Actions: platform.DragCopy,
		Done:    func(act platform.DragAction) { done, told = act, true },
	}
	if !first.StartDrag(d) {
		t.Fatal("the drag did not start")
	}
	first.Close()
	a.PumpOnce()

	if !told {
		t.Error("the source was never told how its drag ended")
	}
	if told && done != platform.DragNone {
		t.Errorf("a drag whose source closed ended as %v, want none", done)
	}
	if second.Dragging() {
		t.Error("the application still reports a drag in progress")
	}
	// And another drag can start, which is what the stale state prevented.
	again := &widget.Drag{Types: []string{"text/plain"},
		Data:  func(string) ([]byte, bool) { return nil, false },
		Local: true}
	if !second.StartDrag(again) {
		t.Error("no further drag could start: the closed window's run is still held")
	}
}

// Hiding a widget has to take its pixels with it. SetVisible(false)
// invalidates the child but does not request layout, and subtreeDirty
// dismissed invisible nodes before looking at their dirty flag — so the
// parent reused a cached group that still contained the hidden child,
// and a hidden button stayed on screen until something else reset the
// cache.
func TestHidingAWidgetRemovesItFromTheScene(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	btn := widgets.NewButton("Hide me", nil)
	w.SetContent(widgets.NewColumn(btn))
	a.PumpOnce()

	shown := w.surf.Buffer().Clone()
	btn.SetVisible(false)
	a.PumpOnce()
	hidden := w.surf.Buffer().Clone()

	if n := diffPixels(t, shown, hidden); n == 0 {
		t.Fatal("hiding the button changed nothing on screen: its pixels are still in the parent's cached group")
	}
}
