package platform

import (
	"runtime"
	"testing"
)

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

// No watcher on the bus is not a bus error — it is a statement that
// nothing is registering status items, so nothing can be showing one.
// That is exactly what a GNOME without an AppIndicator extension looks
// like, which is the case Shown exists for; reading it as "shown"
// defeated the whole question.
//
// A watcher that is there and will not answer is still read as shown:
// refusing to close to a tray that works is the worse failure.
func TestLinuxTrayNotShownWithNoWatcher(t *testing.T) {
	// Linux's tray, as the name says — and the skip below is not enough
	// on its own, because it comes after the item has been made. On
	// macOS NewStatusItem reaches NSStatusItem, which blocks without a
	// logged-in console session: over SSH it hung for the whole test
	// timeout and took the rest of the platform package down with it, so
	// nothing in this package ran on the Mac at all.
	if runtime.GOOS != "linux" {
		t.Skip("the SNI tray is Linux's")
	}
	it, err := NewStatusItem(StatusItemOptions{ID: "uitoolkit-test", Title: "test"})
	if err != nil {
		t.Skip(err)
	}
	defer it.Close()
	if it.Backend() != "sni" {
		t.Skipf("no SNI backend here (%s)", it.Backend())
	}
	// The test bus has no org.kde.StatusNotifierWatcher on it.
	if it.Shown() {
		t.Error("Shown is true on a bus with no watcher — nothing is displaying the item")
	}
	if !it.Alive() {
		t.Error("Alive should still be true: there is a bus to send to")
	}
}
