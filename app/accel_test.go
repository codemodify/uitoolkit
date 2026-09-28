package app

import (
	"runtime"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// accelRig is a window with a menu bar carrying one Ctrl+S item, and a
// flag that says whether it ran.
func accelRig(t *testing.T) (*Window, *bool) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	fired := false
	mb := widgets.NewMenuBar(&widgets.Menu{
		Title: "File",
		Items: []*widgets.MenuItem{
			{Text: "Save", Shortcut: "Ctrl+S", OnClick: func() { fired = true }},
		},
	})
	w.SetContent(widgets.NewColumn(mb, widgets.NewButton("x", nil)))
	a.PumpOnce()
	return w, &fired
}

// The shortcut a menu writes as Ctrl+S is the one the platform's own
// shortcut modifier fires: Control elsewhere, Command on a Mac. This
// is the end of the path platform.AccelMods starts.
func TestAcceleratorUsesThePlatformModifier(t *testing.T) {
	w, fired := accelRig(t)
	w.dispatch(platform.Event{
		Kind: platform.EventKeyDown, Key: platform.KeyS,
		Mods: platform.PrimaryModifier(),
	})
	if !*fired {
		t.Fatalf("a Ctrl+S menu item did not fire on %v, which is %s's shortcut modifier",
			platform.PrimaryModifier(), runtime.GOOS)
	}
}

// And the other modifier does not fire it. On a Mac that means a
// Control+S typed into a text field is not Save — Control there is the
// emacs-ism, not the shortcut key — and elsewhere it means a stray
// Super+S does not trip menus.
func TestAcceleratorIgnoresTheOtherModifier(t *testing.T) {
	other := platform.ModSuper
	if platform.PrimaryModifier() == platform.ModSuper {
		other = platform.ModCtrl
	}
	w, fired := accelRig(t)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyS, Mods: other})
	if *fired {
		t.Fatalf("a Ctrl+S menu item fired on %v, which is not %s's shortcut modifier",
			other, runtime.GOOS)
	}
}
