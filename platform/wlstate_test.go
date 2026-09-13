package platform

import "testing"

func TestClipCacheInvalidatedWhenSelectionLost(t *testing.T) {
	var c clipCache
	if _, ok := c.get(); ok {
		t.Fatal("fresh cache must not answer")
	}
	c.set("copied by us")
	if got, ok := c.get(); !ok || got != "copied by us" {
		t.Fatalf("get = %q,%v after set", got, ok)
	}
	// Another client takes the selection: wl_data_source.cancelled.
	c.invalidate()
	if got, ok := c.get(); ok {
		t.Fatalf("cache still answers %q after cancelled", got)
	}
}

func TestClipCacheSelectionChanged(t *testing.T) {
	var c clipCache
	c.set("ours")
	// A selection event while our source is alive keeps the cache.
	c.selectionChanged(true)
	if got, ok := c.get(); !ok || got != "ours" {
		t.Fatalf("own selection dropped the cache: %q,%v", got, ok)
	}
	// A selection event with no source of ours means a foreign owner.
	c.selectionChanged(false)
	if _, ok := c.get(); ok {
		t.Fatal("foreign selection must invalidate the cache")
	}
	// Re-copying re-arms it.
	c.set("again")
	if got, ok := c.get(); !ok || got != "again" {
		t.Fatalf("re-set = %q,%v", got, ok)
	}
}

func TestPickPresentSlot(t *testing.T) {
	if i, ok := pickPresentSlot([]bool{false, false, false, false}); !ok || i != 0 {
		t.Fatalf("all free = %d,%v", i, ok)
	}
	if i, ok := pickPresentSlot([]bool{true, true, false, true}); !ok || i != 2 {
		t.Fatalf("one free = %d,%v", i, ok)
	}
	// Every buffer still owned by the compositor: the caller must skip
	// the frame instead of scribbling over a busy buffer.
	if _, ok := pickPresentSlot([]bool{true, true, true, true}); ok {
		t.Fatal("all busy must report no slot")
	}
	if _, ok := pickPresentSlot(nil); ok {
		t.Fatal("no slots must report no slot")
	}
}

func TestOutputSetMaxScale(t *testing.T) {
	o := newOutputSet()
	if got := o.maxScale(); got != 1 {
		t.Fatalf("empty = %d, want 1", got)
	}
	o.setScale(1, 1) // laptop panel
	o.setScale(2, 2) // hidpi monitor
	// Before the first enter, fall back to the largest on the desktop.
	if got := o.maxScale(); got != 2 {
		t.Fatalf("no enter = %d, want 2", got)
	}
	if o.anyEntered() {
		t.Fatal("anyEntered before enter")
	}
	// A window on the 1x panel renders at 1x, not at the other
	// monitor's 2x (the old single-output bug).
	o.enter(1)
	if got := o.maxScale(); got != 1 {
		t.Fatalf("on 1x output = %d, want 1", got)
	}
	// Straddling both outputs: the larger scale wins.
	o.enter(2)
	if got := o.maxScale(); got != 2 {
		t.Fatalf("straddling = %d, want 2", got)
	}
	o.leave(2)
	if got := o.maxScale(); got != 1 {
		t.Fatalf("after leave = %d, want 1", got)
	}
	// A scale change on the entered output is picked up.
	o.setScale(1, 3)
	if got := o.maxScale(); got != 3 {
		t.Fatalf("rescaled = %d, want 3", got)
	}
	// Removing the output the surface is on falls back to the rest.
	o.remove(1)
	if got := o.maxScale(); got != 2 {
		t.Fatalf("after remove = %d, want 2", got)
	}
	if o.anyEntered() {
		t.Fatal("removed output still counted as entered")
	}
}

func TestOutputSetIgnoresBogusScale(t *testing.T) {
	o := newOutputSet()
	o.setScale(1, 0)
	o.setScale(2, -4)
	if got := o.maxScale(); got != 1 {
		t.Fatalf("bogus scales = %d, want 1", got)
	}
}

func TestNilOutputSetIsSafe(t *testing.T) {
	var o *outputSet
	o.setScale(1, 2)
	o.enter(1)
	o.leave(1)
	o.remove(1)
	if got := o.maxScale(); got != 1 {
		t.Fatalf("nil set = %d, want 1", got)
	}
	if o.anyEntered() {
		t.Fatal("nil set entered")
	}
}

func TestMaxScaleEnteredUsesConnectionScales(t *testing.T) {
	// The connection owns the scales; each surface owns only the names
	// of the outputs it overlaps. Resolving a surface's scale against
	// its own (empty) scale map always returned 1 -- the bug this
	// guards.
	conn := newOutputSet()
	conn.setScale(10, 1)
	conn.setScale(11, 3)

	surfA := newOutputSet() // on the 1x output
	surfA.enter(10)
	if got := conn.maxScaleEntered(surfA.enteredNames()); got != 1 {
		t.Fatalf("surface on 1x output = %d, want 1", got)
	}

	surfB := newOutputSet() // on the 3x output
	surfB.enter(11)
	if got := conn.maxScaleEntered(surfB.enteredNames()); got != 3 {
		t.Fatalf("surface on 3x output = %d, want 3", got)
	}

	// Not yet mapped anywhere: fall back to the largest on the desktop.
	unmapped := newOutputSet()
	if got := conn.maxScaleEntered(unmapped.enteredNames()); got != 3 {
		t.Fatalf("unmapped surface = %d, want 3", got)
	}
	if got := conn.maxScaleEntered(nil); got != 3 {
		t.Fatalf("nil entered set = %d, want 3", got)
	}
}
