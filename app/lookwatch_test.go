package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"

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

func TestReloadPreferredLookAppliesWhenAppearanceLooksEqual(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(want); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Look: style.PreferredLook(), WatchLook: true, Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("equal"))
	a.PumpOnce()
	if style.LookAppearance(a.Look()).Normalize() != want.Normalize() {
		t.Fatalf("start %+v", style.LookAppearance(a.Look()))
	}
	// Same pack, different JSON (legacy triad). File contents change;
	// Appearance equality would match. Watcher must still SetLook.
	path := style.AppearancePath()
	if err := os.WriteFile(path, []byte("{\n  \"theme\": \"light\",\n  \"corners\": \"square\",\n  \"icons\": \"sharp\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("reapply %+v", got)
	}
}

func TestRunIdlePollsLookFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("run-idle"))
	done := make(chan error, 1)
	go func() { done <- a.Run() }()

	time.Sleep(30 * time.Millisecond)
	next := style.Appearance{Theme: style.ThemeLight, Corners: style.CornersSquare, Icons: style.IconSetSharp}
	if err := style.SaveAppearance(next); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		t.Fatalf("Run exited: %v", err)
	case <-time.After(lookWatchInterval + 200*time.Millisecond):
	}
	a.Quit()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got := style.LookAppearance(a.Look())
	if got.Theme != style.ThemeLight || got.Corners != style.CornersSquare || got.Icons != style.IconSetSharp {
		t.Fatalf("Run idle poll %+v", got)
	}
}

func BenchmarkLookWatchSettled(b *testing.B) {
	b.Setenv("XDG_CONFIG_HOME", b.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		b.Fatal(err)
	}
	s := newLookFileStamp()
	past := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(s.path, past, past); err != nil {
		b.Fatal(err)
	}
	s.refreshMeta()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s.changed() {
			b.Fatal("settled file reported change")
		}
	}
}

func TestLookWatchStatFirstSkipsSettledFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("stat"))
	a.PumpOnce()
	if a.lookWatch == nil {
		t.Fatal("expected watcher")
	}
	path := style.AppearancePath()
	past := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}
	a.lookWatch.refreshMeta()
	n := a.lookWatch.reads
	a.PumpOnce()
	if a.lookWatch.reads != n {
		t.Fatalf("settled size+mtime must not ReadFile, reads %d→%d", n, a.lookWatch.reads)
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

// TestLookWatchFollowsPackFile pins the watcher half of the style review:
// editing the selected pack's theme.json used to need a restart because only
// look.json was stamped.
func TestLookWatchFollowsPackFile(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	dir := filepath.Join(cfg, "uitoolkit", "themes", "mypack")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pack := filepath.Join(dir, "theme.json")
	write := func(body string) {
		if err := os.WriteFile(pack, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		past := time.Now().Add(-2 * time.Second)
		_ = os.Chtimes(pack, past, past)
	}
	write(`{"palette":"dark","metrics":{"scroll":12},"colors":{"accent":"#112233"}}`)
	if err := style.SaveAppearance(style.Appearance{Name: "mypack", Theme: style.ThemeDark}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	w, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 80, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("pack"))
	a.PumpOnce()
	if a.Look().Metrics().Scroll != 12 {
		t.Fatalf("pack metrics not applied: %v", a.Look().Metrics().Scroll)
	}
	if a.lookWatch == nil || a.lookWatch.pack == nil || a.lookWatch.pack.path != pack {
		t.Fatalf("watcher should stamp %s", pack)
	}

	// Edit only the pack file: look.json is untouched.
	write(`{"palette":"dark","metrics":{"scroll":20},"colors":{"accent":"#445566"}}`)
	a.PumpOnce()
	if got := a.Look().Metrics().Scroll; got != 20 {
		t.Fatalf("pack edit needs a restart, scroll %v", got)
	}
	if a.Look().Palette().Accent != mustColor(t, "#445566") {
		t.Fatalf("pack accent %+v", a.Look().Palette().Accent)
	}

	// A settled pack file must not be re-read on every poll.
	a.lookWatch.refreshMeta()
	n := a.lookWatch.reads
	a.PumpOnce()
	a.PumpOnce()
	if a.lookWatch.reads != n {
		t.Fatalf("settled pack re-read: %d→%d", n, a.lookWatch.reads)
	}
}

// TestLookWatchSettleWindowIsAbsolute pins the future-mtime case: a file
// stamped ahead of the clock used to be re-read on every poll forever.
func TestLookWatchSettleWindowIsAbsolute(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	s := newLookFileStamp()
	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(s.path, future, future); err != nil {
		t.Fatal(err)
	}
	s.refreshMeta()
	if s.changed() {
		t.Fatal("a future-stamped file should settle, not report a change")
	}
	n := s.reads
	for i := 0; i < 5; i++ {
		if s.changed() {
			t.Fatal("repeat poll reported a change")
		}
	}
	if s.reads != n {
		t.Fatalf("future mtime kept re-reading: %d→%d", n, s.reads)
	}
}

func mustColor(t *testing.T, hex string) paintengine2d.Color {
	t.Helper()
	c, ok := style.ParseHexColor(hex)
	if !ok {
		t.Fatalf("bad hex %q", hex)
	}
	return c
}
