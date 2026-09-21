package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// shortcutBox is a component with a shortcut table written over
// characters, the way one naturally is.
type shortcutBox struct {
	widget.Base
	seen  []widget.KeyEvent
	fired int
}

func newShortcutBox() *shortcutBox {
	b := &shortcutBox{}
	b.Init(b)
	b.SetWantsFocus(true)
	return b
}

func (b *shortcutBox) Measure(c layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(c.MaxW, c.MaxH)
}

func (b *shortcutBox) KeyPress(e widget.KeyEvent) bool {
	b.seen = append(b.seen, e)
	switch e.Rune {
	case 's', 'z', 'b':
		b.fired++
		return true
	}
	return false
}

// TestKeyEventCarriesTheCharacter is the defect: a KeyEvent carried the key
// and nothing else, so every shortcut written against e.Rune was dead.
func TestKeyEventCarriesTheCharacter(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Keys", Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	box := newShortcutBox()
	w.SetContent(box)
	a.PumpOnce()
	w.RequestFocus(box)

	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyS})
	if box.fired != 1 {
		t.Fatalf("a shortcut on 's' did not fire: %d", box.fired)
	}
	if got := box.seen[0]; got.Rune != 's' || got.Key != platform.KeyS {
		t.Fatalf("key event %+v, want key S and rune 's'", got)
	}

	// The character is the key's, with no modifier folded in: Ctrl+S and
	// Shift+S are both 's', and the modifier is where a table looks for it.
	for _, m := range []platform.Modifiers{platform.ModCtrl, platform.ModShift, platform.ModCtrl | platform.ModShift} {
		box.seen = nil
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeyS, Mods: m})
		if got := box.seen[0]; got.Rune != 's' || got.Mods != m {
			t.Errorf("with mods %v: %+v, want rune 's' and the mods kept", m, got)
		}
	}

	// A key that navigates rather than types has no character, so a table
	// written over characters cannot fire on one by accident.
	for _, k := range []platform.Key{platform.KeyTab, platform.KeyReturn, platform.KeyBackspace,
		platform.KeyEscape, platform.KeyLeft, platform.KeyF1, platform.KeyDelete} {
		box.seen = nil
		w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: k})
		if len(box.seen) == 0 {
			// Tab and Escape are the window's before they are the widget's.
			continue
		}
		if r := box.seen[0].Rune; r != 0 {
			t.Errorf("key %v carries rune %q, want none", k, r)
		}
	}

	// Space is a character and Return is not: the two are told apart by
	// the field a table reads.
	box.seen = nil
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Key: platform.KeySpace})
	if got := box.seen[0]; got.Rune != ' ' {
		t.Errorf("space: %+v, want rune ' '", got)
	}
}

// TestKeyCharTable pins the character of every key that has one.
func TestKeyCharTable(t *testing.T) {
	for k, want := range map[platform.Key]rune{
		platform.KeyA: 'a', platform.KeyM: 'm', platform.KeyZ: 'z',
		platform.KeySpace: ' ', platform.Key3: '3', platform.KeyHash: '#',
		platform.KeyComma: ',',
		platform.KeyTab:   0, platform.KeyReturn: 0, platform.KeyEscape: 0,
		platform.KeyUp: 0, platform.KeyF4: 0, platform.KeyAlt: 0,
		platform.KeyMenu: 0, platform.KeyUnknown: 0,
	} {
		if got := platform.KeyChar(k); got != want {
			t.Errorf("KeyChar(%v) = %q, want %q", k, got, want)
		}
	}
}

// TestKeyReleaseCarriesTheCharacter: a table that watches a key going up
// (a push-to-talk, a modifier-less hold) reads the same field.
func TestKeyReleaseCarriesTheCharacter(t *testing.T) {
	t.Setenv(platform.EnvDecorations, "")
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Up", Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	box := newReleaseBox()
	w.SetContent(box)
	a.PumpOnce()
	w.RequestFocus(box)
	w.dispatch(platform.Event{Kind: platform.EventKeyUp, Key: platform.KeyV})
	if box.up.Rune != 'v' || box.up.Key != platform.KeyV {
		t.Errorf("key release %+v, want key V and rune 'v'", box.up)
	}
}

type releaseBox struct {
	widget.Base
	up widget.KeyEvent
}

func newReleaseBox() *releaseBox {
	b := &releaseBox{}
	b.Init(b)
	b.SetWantsFocus(true)
	return b
}

func (b *releaseBox) Measure(c layout.Constraints) paintengine2d.Point {
	return paintengine2d.Pt(c.MaxW, c.MaxH)
}

func (b *releaseBox) KeyRelease(e widget.KeyEvent) bool { b.up = e; return true }
