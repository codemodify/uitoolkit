package platform

import "testing"

// Alive says there is somewhere to send the item; Shown says someone is
// showing it. An application that closes to the tray has to ask the
// second question — a GNOME with no AppIndicator extension has a session
// bus and no tray, and a window hidden there has no way back.
func TestFakeStatusItemShown(t *testing.T) {
	it := newFakeStatusItem(StatusItemOptions{})
	if !it.Shown() {
		t.Fatal("a fresh fake item should show: a test written before Shown existed must see what it always saw")
	}

	var got []bool
	it.SetOnShownChange(func(v bool) { got = append(got, v) })

	it.SetShown(false)
	if it.Shown() {
		t.Error("Shown is true after the host went away")
	}
	if !it.Alive() {
		t.Error("a hidden item is still alive: the two questions are different")
	}
	// Setting it again is not a change.
	it.SetShown(false)
	it.SetShown(true)
	if !it.Shown() {
		t.Error("Shown is false after the host came back")
	}
	if len(got) != 2 || got[0] || !got[1] {
		t.Errorf("callbacks %v, want one false then one true", got)
	}

	_ = it.Close()
	if it.Shown() {
		t.Error("a closed item shows nothing")
	}
}

// A stub has no host at all, so it never shows anything — an application
// that asked would otherwise hide its window behind an icon that does
// not exist.
func TestStubStatusItemShowsNothing(t *testing.T) {
	it := newStubStatusItem(StatusItemOptions{})
	if it.Shown() || it.Alive() {
		t.Error("a stub is neither alive nor shown")
	}
	it.SetOnShownChange(func(bool) { t.Error("a stub called back") })
}
