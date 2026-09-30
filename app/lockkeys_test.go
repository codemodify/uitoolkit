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

// lockWatcher counts what it is told, as a widget showing the state
// would.
type lockWatcher struct {
	widgets.Label
	caps, num bool
	n         int
}

func newLockWatcher() *lockWatcher {
	w := &lockWatcher{}
	w.Init(w)
	return w
}

func (w *lockWatcher) LockKeysChanged(caps, num bool) {
	w.caps, w.num, w.n = caps, num, w.n+1
}

// OnLockKeys is one callback a window, so a widget that wanted the state
// had to take it and hand it back — and two widgets in one window could
// not both follow it. Every component under the window's roots hears it
// now, the same way FocusWatcher works.
func TestEveryWatcherInTheWindowHearsTheLockKeys(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	one, two := newLockWatcher(), newLockWatcher()
	w.SetContent(widgets.NewColumn(one, widgets.NewColumn(two)))
	a.PumpOnce()

	// The window's own callback still works alongside them.
	own := 0
	w.OnLockKeys(func(bool, bool) { own++ })

	key := func(mods platform.Modifiers) {
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyA, Mods: mods})
		a.PumpOnce()
	}
	// Watchers hear the first event: for them it is not "the state
	// became known" but "here is the state", and a field drawing a Caps
	// Lock mark has nothing until it is told. The window's own callback
	// keeps its documented meaning and fires on a change only.
	key(0)
	if one.n != 1 || two.n != 1 {
		t.Fatalf("the first event did not reach the watchers: %d %d", one.n, two.n)
	}
	if own != 0 {
		t.Fatalf("OnLockKeys fired on the first event: %d", own)
	}

	key(platform.ModCapsLock)
	if one.n != 2 || !one.caps {
		t.Errorf("first watcher: %d calls, caps %v", one.n, one.caps)
	}
	if two.n != 2 || !two.caps {
		t.Errorf("second watcher (nested): %d calls, caps %v", two.n, two.caps)
	}
	if own != 1 {
		t.Errorf("the window's own callback fired %d times, want 1", own)
	}

	key(0)
	if one.caps || two.caps {
		t.Error("a watcher kept caps on after it went off")
	}
	if one.n != 3 || two.n != 3 {
		t.Errorf("calls %d %d, want 3 each", one.n, two.n)
	}
}

// And the passphrase field shows its own mark with no application code
// at all, which is the point of the interface.
func TestSecretFieldShowsCapsLockWithNoApplicationCode(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	f := widgets.NewSecretField("Passphrase")
	w.SetContent(f)
	w.RequestFocus(f)
	a.PumpOnce()

	key := func(mods platform.Modifiers) {
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyA, Mods: mods})
		a.PumpOnce()
	}
	key(0)
	if f.CapsLockOn() {
		t.Fatal("Caps Lock is on before anything said so")
	}
	key(platform.ModCapsLock)
	if !f.CapsLockOn() {
		t.Error("the field did not hear that Caps Lock came on")
	}
	key(0)
	if f.CapsLockOn() {
		t.Error("the field did not hear that Caps Lock went off")
	}
}

// A window that has been open with Caps Lock on since before a dialog
// was shown has no *change* to report, so a field that learns only of
// changes drew no mark on the passphrase that was about to be refused —
// which is the one that matters. It asks when it takes the focus.
func TestSecretFieldKnowsTheLockItOpensWith(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	// The lock was already on before anything was shown: one event, and
	// then nothing changes.
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyA, Mods: platform.ModCapsLock})
	a.PumpOnce()

	f := widgets.NewSecretField("Passphrase")
	w.SetContent(f)
	a.PumpOnce()
	w.RequestFocus(f)
	a.PumpOnce()

	if !f.CapsLockOn() {
		t.Error("a field focused in a window whose Caps Lock is already on does not know it")
	}
}
