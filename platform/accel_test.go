package platform

import (
	"runtime"
	"testing"
)

// The identity everywhere but macOS, and a swap there. Written as one
// test rather than two files so that the *whole* rule is visible from
// either platform, and so that changing one half without the other
// fails somewhere.
func TestAccelModsSwapOnlyOnDarwin(t *testing.T) {
	mac := runtime.GOOS == "darwin"
	if want := ModCtrl; !mac && PrimaryModifier() != want {
		t.Fatalf("the primary modifier is %v on %s, want %v", PrimaryModifier(), runtime.GOOS, want)
	}
	if mac && PrimaryModifier() != ModSuper {
		t.Fatalf("the primary modifier is %v on macOS, want Command", PrimaryModifier())
	}

	for _, c := range []struct {
		name          string
		in            Modifiers
		elsewhere, on Modifiers
	}{
		{"nothing", 0, 0, 0},
		{"the shortcut modifier", ModCtrl, ModCtrl, ModSuper},
		{"the other one", ModSuper, ModSuper, ModCtrl},
		{"shift alone is untouched", ModShift, ModShift, ModShift},
		{"alt alone is untouched", ModAlt, ModAlt, ModAlt},
		{"ctrl and shift", ModCtrl | ModShift, ModCtrl | ModShift, ModSuper | ModShift},
		{"super and shift", ModSuper | ModShift, ModSuper | ModShift, ModCtrl | ModShift},
		// Both held at once swap past each other rather than collapsing
		// into one, which a naive "if Super then Ctrl" would do.
		{"both at once", ModCtrl | ModSuper, ModCtrl | ModSuper, ModCtrl | ModSuper},
		{"everything", ModCtrl | ModSuper | ModShift | ModAlt,
			ModCtrl | ModSuper | ModShift | ModAlt, ModCtrl | ModSuper | ModShift | ModAlt},
	} {
		want := c.elsewhere
		if mac {
			want = c.on
		}
		if got := AccelMods(c.in); got != want {
			t.Errorf("%s: AccelMods(%v) is %v, want %v on %s", c.name, c.in, got, want, runtime.GOOS)
		}
	}
}

// Whatever the platform, applying it twice is applying it once: a swap
// is its own inverse, so an event that went through dispatch and then
// through something that re-normalised would not come out wrong.
func TestAccelModsIsItsOwnInverse(t *testing.T) {
	for m := Modifiers(0); m <= ModCtrl|ModShift|ModAlt|ModSuper; m++ {
		if got := AccelMods(AccelMods(m)); got != m {
			t.Errorf("AccelMods twice on %v gives %v", m, got)
		}
	}
}
