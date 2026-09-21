package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func openWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Open", Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	return a, w
}

// TestWindowOpensFocused is the defect: a window opened with nothing
// focused, so every key that bubbles from the focus reached nobody until
// the user clicked or pressed Tab.
func TestWindowOpensFocused(t *testing.T) {
	a, w := openWindow(t)
	first := widgets.NewTextField("", "", nil)
	second := widgets.NewButton("Go", nil)
	w.SetContent(widgets.NewColumn(first, second))
	a.PumpOnce()
	if w.Focus() != widget.Component(first) {
		t.Fatalf("focus is %v, want the first focusable", w.Focus())
	}
	// And the keyboard reaches it without a click or a Tab.
	w.dispatch(platform.Event{Kind: platform.EventText, Rune: 'k'})
	if first.Text != "k" {
		t.Errorf("the field did not take the key: %q", first.Text)
	}
}

// TestWindowOpenFocusDoesNotShowAKeyboardRing: focus arrives the way a
// click's does, so a look that shows its ring only after keyboard
// navigation does not draw one on a window that has just opened.
func TestWindowOpenFocusDoesNotShowAKeyboardRing(t *testing.T) {
	a, w := openWindow(t)
	b := widgets.NewButton("Go", nil)
	w.SetContent(widgets.NewColumn(b))
	a.PumpOnce()
	if w.Focus() != widget.Component(b) {
		t.Fatalf("focus is %v, want the button", w.Focus())
	}
	if b.State()&style.StateFocused != 0 {
		t.Error("a freshly opened window draws a keyboard focus ring")
	}
	// Tab is keyboard navigation and does show it.
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyTab})
	if b.State()&style.StateFocused == 0 {
		t.Error("Tab did not show the focus ring")
	}
}

// TestWindowKeepsFocusTheAppPlaced: an app that focuses something itself
// before the first frame is not overruled.
func TestWindowKeepsFocusTheAppPlaced(t *testing.T) {
	a, w := openWindow(t)
	first := widgets.NewTextField("", "", nil)
	second := widgets.NewTextField("", "", nil)
	w.SetContent(widgets.NewColumn(first, second))
	w.RequestFocus(second)
	a.PumpOnce()
	if w.Focus() != widget.Component(second) {
		t.Fatalf("focus is %v, want the one the app placed", w.Focus())
	}
}

// TestSetInitialFocusNamesTheComponent: an app whose first control is not
// where it wants to start says so.
func TestSetInitialFocusNamesTheComponent(t *testing.T) {
	a, w := openWindow(t)
	first := widgets.NewTextField("", "", nil)
	second := widgets.NewTextField("", "", nil)
	w.SetContent(widgets.NewColumn(first, second))
	w.SetInitialFocus(second)
	a.PumpOnce()
	if w.Focus() != widget.Component(second) {
		t.Fatalf("focus is %v, want the named component", w.Focus())
	}
}

// TestWindowOpenFocusHappensOnce: it is the window opening, not every
// layout — a later Escape or content swap that leaves nothing focused
// stays that way.
func TestWindowOpenFocusHappensOnce(t *testing.T) {
	a, w := openWindow(t)
	f := widgets.NewTextField("", "", nil)
	w.SetContent(widgets.NewColumn(f))
	a.PumpOnce()
	if w.Focus() == nil {
		t.Fatal("nothing focused on open")
	}
	w.RequestFocus(nil)
	w.RequestLayout()
	a.PumpOnce()
	if w.Focus() != nil {
		t.Errorf("the window took focus back on a later layout: %v", w.Focus())
	}
}

// TestWindowOpenFocusSkipsChrome: a menu bar or a tool bar takes focus
// from Tab and from its mnemonic, never from a click and never from a
// window opening. Its accelerators work with no focus at all.
func TestWindowOpenFocusSkipsChrome(t *testing.T) {
	a, w := openWindow(t)
	ran := 0
	bar := widgets.NewMenuBar(widgets.NewMenu("&File",
		widgets.ItemAccel("&New", "Ctrl+N", func() { ran++ })))
	w.SetContent(widgets.NewColumn(bar))
	a.PumpOnce()
	if w.Focus() != nil {
		t.Errorf("a window of nothing but chrome opened focused on %v", w.Focus())
	}
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyN, Mods: platform.ModCtrl})
	if ran != 1 {
		t.Errorf("the accelerator ran %d times with no focus", ran)
	}
}

// TestWindowOpenFocusPrefersTheOverlay: a window that opens with a modal
// dialog up starts in the dialog, not behind it.
func TestWindowOpenFocusPrefersTheOverlay(t *testing.T) {
	a, w := openWindow(t)
	behind := widgets.NewTextField("", "", nil)
	w.SetContent(widgets.NewColumn(behind))
	inDialog := widgets.NewTextField("", "", nil)
	card := widgets.NewColumn(inDialog)
	card.SetBounds(paintengine2d.XYWH(0, 0, 200, 100))
	ov := widgets.NewOverlay(card)
	ov.Modal = true
	w.SetOverlay(ov)
	a.PumpOnce()
	if w.Focus() != widget.Component(inDialog) {
		t.Fatalf("focus is %v, want the field in the dialog", w.Focus())
	}
}
