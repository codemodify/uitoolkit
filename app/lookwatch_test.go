package app

import (
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestPreferredLookWatchesFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dark := style.Appearance{Theme: style.ThemeDark, Corners: style.CornersRound, Icons: style.IconSetClassic}
	if err := style.SaveAppearance(dark); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if !a.WatchingLook() {
		t.Fatal("nil Look should watch look.json")
	}
	w, err := a.NewWindow(platform.WindowOptions{Title: "watch", Width: 240, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("theme"))
	a.PumpOnce()
	if a.Look().Name() != "dark" {
		t.Fatalf("start %s", a.Look().Name())
	}

	light := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(light); err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("watcher %+v", got)
	}
	if a.Look().Metrics().Radius != 0 {
		t.Fatalf("square radius %v", a.Look().Metrics().Radius)
	}
}

func TestExplicitDarkLookDoesNotWatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Look: style.DarkLook(), Headless: true, Scale: 1})
	if a.WatchingLook() {
		t.Fatal("DarkLook must stay static")
	}
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("fixed"))
	a.PumpOnce()
	light := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(light); err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	if a.Look().Name() != "dark" {
		t.Fatalf("frozen look became %s", a.Look().Name())
	}
}

func TestWatchLookOptInWithPreferredLook(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Look: style.PreferredLook(), WatchLook: true, Headless: true, Scale: 1})
	if !a.WatchingLook() {
		t.Fatal("WatchLook + PreferredLook")
	}
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("opt-in"))
	a.PumpOnce()
	if err := style.SaveAppearance(style.Appearance{Theme: style.ThemeLight}); err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	if a.Look().Name() != "light" {
		t.Fatalf("opt-in watch %s", a.Look().Name())
	}
}

func TestDisableLookWatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Headless: true, Scale: 1, DisableLookWatch: true})
	if a.WatchingLook() {
		t.Fatal("DisableLookWatch")
	}
}

func TestLookWatchWaitTimeoutBounded(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("idle"))
	a.PumpOnce()
	d := a.waitTimeout(time.Now(), time.Time{})
	if d < 0 || d > lookWatchInterval {
		t.Fatalf("watch wait %v", d)
	}
}

func TestSecondAppSeesApplyWithoutRestart(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	listener := New(Options{Headless: true, Scale: 1})
	lw, err := listener.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer lw.Close()
	lw.SetContent(widgets.NewLabel("listener"))
	listener.PumpOnce()

	next := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(next); err != nil {
		t.Fatal(err)
	}
	listener.PumpOnce()
	got := style.LookAppearance(listener.Look())
	if got != next.Normalize() {
		t.Fatalf("listener %+v", got)
	}
}
