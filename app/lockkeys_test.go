package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func lockWindow(t *testing.T) (*Application, *Window, *platform.Offscreen) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
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

// A window that has seen no key yet can still say whether Caps Lock is
// on, which is the point: the warning has to be there *before* the
// first keystroke, not after the first refused passphrase.
func TestLockKeysBeforeTheFirstKey(t *testing.T) {
	_, w, o := lockWindow(t)
	if caps, num := w.LockKeys(); caps || num {
		t.Fatalf("caps=%v num=%v before anything was set", caps, num)
	}
	o.SetLockKeys(true, false)
	if caps, _ := w.LockKeys(); !caps {
		t.Fatal("the window cannot see the keyboard's lock state")
	}
}

// The state follows the key events, and the callback fires on a change
// and only on a change — a keystroke with the lock unchanged is not
// news.
func TestLockKeysFollowEventsAndNotify(t *testing.T) {
	a, w, _ := lockWindow(t)
	type call struct{ caps, num bool }
	var calls []call
	w.OnLockKeys(func(caps, num bool) { calls = append(calls, call{caps, num}) })

	key := func(mods platform.Modifiers) {
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyA, Mods: mods})
		a.PumpOnce()
	}
	// The first event makes the state known rather than changing it.
	key(0)
	if len(calls) != 0 {
		t.Fatalf("the first event fired %v", calls)
	}
	key(platform.ModCapsLock)
	if caps, _ := w.LockKeys(); !caps {
		t.Fatal("Caps Lock did not stick")
	}
	if len(calls) != 1 || !calls[0].caps {
		t.Fatalf("calls %v", calls)
	}
	// Again, unchanged: not news.
	key(platform.ModCapsLock)
	if len(calls) != 1 {
		t.Fatalf("an unchanged lock fired again: %v", calls)
	}
	key(platform.ModCapsLock | platform.ModNumLock)
	if len(calls) != 2 || !calls[1].num {
		t.Fatalf("calls %v", calls)
	}
	key(0)
	if len(calls) != 3 || calls[2].caps || calls[2].num {
		t.Fatalf("calls %v", calls)
	}
}

// The lock keys are not part of a shortcut. Ctrl+S is Ctrl+S with Caps
// Lock on, and an accelerator table that compared the whole modifier set
// would have stopped matching the moment a user pressed it.
func TestLockKeysAreNotPartOfAShortcut(t *testing.T) {
	mods := platform.ModCtrl | platform.ModCapsLock | platform.ModNumLock
	if got := mods.Chord(); got != platform.ModCtrl {
		t.Fatalf("Chord() = %v, want just Ctrl", got)
	}
	if !mods.Ctrl() || !mods.CapsLock() || !mods.NumLock() {
		t.Fatal("the flags do not read back")
	}
	if (platform.ModCtrl).CapsLock() {
		t.Fatal("Ctrl reads as Caps Lock")
	}
}

// A menu accelerator still fires with Caps Lock on.
func TestAcceleratorFiresWithCapsLockOn(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	fired := 0
	mb := widgets.NewMenuBar(widgets.NewMenu("&File",
		widgets.ItemAccel("&Save", "Ctrl+S", func() { fired++ }),
	))
	w.SetContent(widgets.NewColumn(mb, widgets.NewLabel("body")))
	a.PumpOnce()
	w.dispatch(platform.Event{
		Kind: platform.EventKeyDown, Key: platform.KeyS,
		Mods: platform.PrimaryModifier() | platform.ModCapsLock,
	})
	a.PumpOnce()
	if fired != 1 {
		t.Fatalf("the accelerator fired %d times with Caps Lock on", fired)
	}
}
