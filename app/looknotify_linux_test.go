//go:build linux

package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/style"
)

// An app with a display watches look.json's directory instead of polling
// it: a save wakes it, and the run loop sets no idle poll.
func TestLookNotifyWakesOnSave(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	a := New(Options{Headless: false, Backend: "offscreen", Scale: 1})
	if a.lookNotify == nil {
		t.Skip("no inotify")
	}
	if d := a.waitTimeout(time.Now(), time.Time{}); d >= 0 && d <= lookWatchInterval {
		t.Fatalf("the idle loop still polls look.json every %v", d)
	}
	// The config directory does not exist yet: its parent is watched, and
	// the first save (which creates it) is noticed.
	if err := style.SaveAppearance(style.Appearance{Name: "breeze", Theme: style.ThemeLight}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(xdg, "uitoolkit", "look.json")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		a.DrainPosted()
		if c, ok := a.Look().(*style.Classic); ok && c.Pack() == "breeze" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("look after the save: %v", a.Look().Name())
}
