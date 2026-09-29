//go:build darwin && cgo

package platform

import (
	"testing"
	"time"
)

// The rows flatten into the pre-order array the C side builds from, with
// a click handler registered for each row that has one — and for no
// others: not the separator, not the submenu parent (a parent is not a
// command), not a disabled row, not a plain label.
func TestDarwinTrayMenuFlattens(t *testing.T) {
	clicked := ""
	items := []StatusMenuItem{
		{Text: "Unlock", OnClick: func() { clicked = "unlock" }},
		{Separator: true},
		{Text: "Vaults", OnClick: func() { clicked = "parent" }, Submenu: []StatusMenuItem{
			{Text: "Personal", OnClick: func() { clicked = "personal" }},
			{Text: "Work", Disabled: true, OnClick: func() { clicked = "work" }},
		}},
		{Text: "Locked", Checked: true},
		{Text: "Quit", OnClick: func() { clicked = "quit" }},
	}
	out, ids, n := flattenTrayMenuForTest(items)
	defer trayMenuForget(ids)

	// Five rows at the top level plus two children.
	if n != 7 || len(out) != 7 {
		t.Fatalf("flattened to n=%d len=%d, want 7", n, len(out))
	}
	if out[2].Kids != 2 {
		t.Fatalf("the submenu parent says %d children", out[2].Kids)
	}
	if out[2].ID != 0 {
		t.Fatal("a submenu parent was made clickable")
	}
	// Unlock, Personal and Quit.
	if len(ids) != 3 {
		t.Fatalf("%d handlers registered, want 3", len(ids))
	}
	seen := map[string]bool{}
	for _, id := range ids {
		clicked = ""
		uitkStatusMenuClick(id)
		if clicked == "" {
			t.Fatalf("id %d fired nothing", id)
		}
		seen[clicked] = true
	}
	for _, want := range []string{"unlock", "personal", "quit"} {
		if !seen[want] {
			t.Fatalf("%q never fired", want)
		}
	}
	if seen["work"] || seen["parent"] {
		t.Fatal("a disabled row or a submenu parent was made clickable")
	}

	// Forgetting them makes a stale click a no-op rather than a call
	// into a menu that is gone.
	trayMenuForget(ids)
	clicked = ""
	for _, id := range ids {
		uitkStatusMenuClick(id)
	}
	if clicked != "" {
		t.Fatalf("a forgotten id still fired %q", clicked)
	}
}

// Ids are never reused, so a menu replaced while the old one is open
// cannot fire the new menu's handler for the row under the pointer.
func TestDarwinTrayMenuIdsAreNotReused(t *testing.T) {
	first := trayMenuRegister(func() {})
	trayMenuForget([]uint64{first})
	second := trayMenuRegister(func() {})
	defer trayMenuForget([]uint64{second})
	if second == first {
		t.Fatalf("id %d was handed out twice", first)
	}
}

// applyMenu returns rather than deadlocking, and returns having handed
// the array over to be freed.
//
// This is the bug the first version of this code had: it built the menu
// a row at a time, each row a dispatch_sync to the main queue, and a
// dispatch_sync to the main queue waits for a queue nobody is draining
// whenever NSApp's run loop has not started — every test, and every
// moment of an application's start-up. The whole menu goes over in one
// async block now, which waits for nothing.
//
// The item pointer is nil here, which is the path C takes when there is
// no status item: it frees the array and returns. Making a real
// NSStatusItem needs a main-thread hop of its own that only a running
// NSApp can answer, so no test in this package makes one.
func TestDarwinTrayMenuDoesNotBlock(t *testing.T) {
	d := &darwinStatusItem{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.applyMenu([]StatusMenuItem{
			{Text: "Unlock", OnClick: func() {}},
			{Separator: true},
			{Text: "Vaults", Submenu: []StatusMenuItem{{Text: "Personal", OnClick: func() {}}}},
			{Text: "Quit", OnClick: func() {}},
		})
		d.applyMenu(nil)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("applyMenu did not return: something waited on a main queue nobody is draining")
	}
	// The second call replaced the first, so the first menu's handlers
	// are gone and the second's (none) are all that is left.
	if len(d.menuIDs) != 0 {
		t.Fatalf("%d handlers left after an empty menu", len(d.menuIDs))
	}
}
