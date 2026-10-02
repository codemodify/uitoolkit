package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func captionWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Width: 480, Height: 300, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("content"))
	a.PumpOnce()
	if w.Caption() == nil {
		t.Skip("no toolkit-drawn caption in this configuration")
	}
	return a, w
}

func shownIn(w *Window) map[platform.CaptionButton]bool {
	out := map[platform.CaptionButton]bool{}
	c := w.Caption()
	for _, side := range []*widgets.WindowControls{c.LeadControls(), c.TrailControls()} {
		if side == nil {
			continue
		}
		for _, b := range side.Shown() {
			out[b] = true
		}
	}
	return out
}

// A look and the desktop decide which buttons *can* be there; a window
// decides which of those it wants. A tool window with no maximize, a
// dialog with only a close.
func TestWindowHidesACaptionButton(t *testing.T) {
	a, w := captionWindow(t)
	before := shownIn(w)
	if !before[platform.CaptionClose] {
		t.Skip("this look shows no close button to hide")
	}

	w.SetCaptionButtonVisible(platform.CaptionClose, false)
	a.PumpOnce()
	if shownIn(w)[platform.CaptionClose] {
		t.Error("the close button is still there")
	}
	if !w.CaptionButtonHidden(platform.CaptionClose) {
		t.Error("the window does not report it hidden")
	}

	w.SetCaptionButtonVisible(platform.CaptionClose, true)
	a.PumpOnce()
	if !shownIn(w)[platform.CaptionClose] {
		t.Error("the close button did not come back")
	}
}

// The application's own buttons sit beside the window's, on the side it
// asks for, and run their callback when pressed.
func TestWindowCaptionActions(t *testing.T) {
	a, w := captionWindow(t)
	profile, extensions := 0, 0
	w.SetCaptionActions(
		widgets.CaptionAction{Icon: style.IconUser, Name: "Profile", OnClick: func() { profile++ }},
		widgets.CaptionAction{Icon: style.IconMore, Name: "Extensions", Lead: true, OnClick: func() { extensions++ }},
	)
	a.PumpOnce()

	c := w.Caption()
	lead, trail := c.LeadControls(), c.TrailControls()
	if lead == nil || trail == nil {
		t.Fatal("no window controls")
	}
	if got := len(lead.Actions()); got != 1 {
		t.Errorf("%d lead actions, want 1", got)
	}
	if got := len(trail.Actions()); got != 1 {
		t.Errorf("%d trailing actions, want 1", got)
	}
	// They are in what the side paints and hit-tests.
	found := 0
	for _, side := range []*widgets.WindowControls{lead, trail} {
		for _, b := range side.Shown() {
			if b >= 128 {
				found++
			}
		}
	}
	if found != 2 {
		t.Errorf("%d actions reached the painted set, want 2", found)
	}

	// And pressing one runs it.
	for _, side := range []*widgets.WindowControls{lead, trail} {
		for i, b := range side.Shown() {
			if b >= 128 {
				side.ActivateForTest(i)
			}
		}
	}
	if profile != 1 || extensions != 1 {
		t.Errorf("callbacks ran profile=%d extensions=%d", profile, extensions)
	}
}

// The window-menu button hides like any other. It is the one an era puts
// on the *other* side — Windows' control-menu box, Motif's window menu,
// KDE's "M" — so a window that hides it is hiding something from the
// lead group rather than the trailing one.
func TestWindowHidesTheWindowMenuButton(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Width: 480, Height: 300, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("content"))
	a.PumpOnce()
	if w.Caption() == nil {
		t.Skip("no toolkit-drawn caption here")
	}
	// Put one there whatever this era would have chosen, so the test is
	// about hiding rather than about which pack is in the build.
	c := w.Caption()
	if lead := c.LeadControls(); lead != nil {
		lead.SetButtons([]platform.CaptionButton{platform.CaptionMenu})
	}
	a.PumpOnce()
	if !shownIn(w)[platform.CaptionMenu] {
		t.Skip("this configuration draws no window-menu button")
	}

	w.SetCaptionButtonVisible(platform.CaptionMenu, false)
	a.PumpOnce()
	if shownIn(w)[platform.CaptionMenu] {
		t.Error("the window-menu button is still there")
	}
	w.SetCaptionButtonVisible(platform.CaptionMenu, true)
	a.PumpOnce()
	if !shownIn(w)[platform.CaptionMenu] {
		t.Error("it did not come back")
	}
}

// Every one of them can go, which is what an application drawing its own
// chrome edge to edge wants.
func TestWindowHidesEveryCaptionButton(t *testing.T) {
	a, w := captionWindow(t)
	for _, b := range []platform.CaptionButton{
		platform.CaptionMenu, platform.CaptionMinimize,
		platform.CaptionMaximize, platform.CaptionClose, platform.CaptionKeepAbove,
	} {
		w.SetCaptionButtonVisible(b, false)
	}
	a.PumpOnce()
	if got := len(shownIn(w)); got != 0 {
		t.Errorf("%d buttons left after hiding them all", got)
	}
}

// The user's own preference, rather than the application's: Settings'
// "Window menu", unticked, leaves the button out of every caption the
// toolkit draws, under whatever layout the desktop asked for.
//
// It is the button that goes, not the menu — a right click on the caption
// still opens it — and it has to run before the keep-above swap, or hiding
// the button would simply put a different one in the slot.
func TestHidingTheWindowMenuDropsItsButton(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	// KWin's default: the window menu on the left, the rest on the right.
	a.SetTitleBarPrefs(platform.TitleBarPrefs{
		Layout: platform.ParseButtonLayout("menu:minimize,maximize,close"),
	})
	w, err := a.NewWindow(platform.WindowOptions{
		Width: 480, Height: 300, Headless: true,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("content"))
	a.PumpOnce()
	if w.Caption() == nil {
		t.Skip("no toolkit-drawn caption here")
	}
	if !shownIn(w)[platform.CaptionMenu] {
		t.Skip("this configuration draws no window-menu button")
	}

	a.SetHideWindowMenu(true)
	a.PumpOnce()
	shown := shownIn(w)
	if shown[platform.CaptionMenu] {
		t.Error("the window-menu button survived the preference")
	}
	// The slot is gone, not refilled.
	if shown[platform.CaptionKeepAbove] {
		t.Error("the keep-above button took the window menu's slot; the preference asked for no button there")
	}
	// The three that were never in question are untouched.
	for _, b := range []platform.CaptionButton{
		platform.CaptionMinimize, platform.CaptionMaximize, platform.CaptionClose,
	} {
		if !shown[b] {
			t.Errorf("hiding the window menu also took %v", b)
		}
	}
	if !a.HideWindowMenu() {
		t.Error("HideWindowMenu does not report what was set")
	}

	a.SetHideWindowMenu(false)
	a.PumpOnce()
	if !shownIn(w)[platform.CaptionMenu] {
		t.Error("the button did not come back")
	}
}

// The preference travels in the appearance file, so Settings' Apply and a
// restart agree about it.
func TestHideWindowMenuRoundTripsThroughTheAppearanceFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ap := style.DefaultAppearance()
	if ap.HideWindowMenu {
		t.Fatal("the default is to leave the desktop's layout alone")
	}
	ap.HideWindowMenu = true
	if err := style.SaveAppearance(ap); err != nil {
		t.Fatal(err)
	}
	if got := style.LoadAppearance(); !got.HideWindowMenu {
		t.Error("hideWindowMenu did not survive the file")
	}
}
