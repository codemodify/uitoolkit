package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// dressRig is an offscreen window in pack whose simulated desktop takes a
// frame palette when palette is set, with the user's cache directory in a
// temporary one.
func dressRig(t *testing.T, pack string, palette bool) (*Application, *Window, *platform.Offscreen, string) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv(EnvDecorationPalette, "")
	p, ok := style.LoadTheme(pack)
	if !ok {
		t.Fatalf("no %s pack", pack)
	}
	a := New(Options{Look: p.Look(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Dress", Width: 300, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("Body"))
	o := w.Surface().(*platform.Offscreen)
	o.SimulateDecorationPalette(palette)
	// The desktop is asked again whenever the look changes.
	a.SetLook(p.Look())
	a.PumpOnce()
	return a, w, o, cache
}

// Under a desktop that takes one, the desktop's frame is given the look's
// colour scheme: a file under the cache directory, never the config one,
// named after its contents, and a new one when the look changes.
func TestDecorationPaletteFollowsTheLook(t *testing.T) {
	a, _, o, cache := dressRig(t, "luna", true)
	calls := o.FrameCalls().Palettes
	if len(calls) != 1 {
		t.Fatalf("palettes %v, want one", calls)
	}
	luna := calls[0]
	if dir := filepath.Join(cache, "uitoolkit", "colors"); filepath.Dir(luna) != dir || !strings.HasSuffix(luna, ".colors") {
		t.Fatalf("palette %s is not a .colors file in %s", luna, dir)
	}
	data, err := os.ReadFile(luna)
	if err != nil {
		t.Fatal(err)
	}
	if s := string(data); !strings.Contains(s, "[Colors:Header]") || !strings.Contains(s, "[WM]") || !strings.Contains(s, "ColorScheme=uitoolkit") {
		t.Fatalf("not the look's colour scheme:\n%s", s)
	}
	aqua, _ := style.LoadTheme("aqua")
	a.SetLook(aqua.Look())
	a.PumpOnce()
	calls = o.FrameCalls().Palettes
	if len(calls) != 2 || calls[1] == luna {
		t.Fatalf("after Aqua: palettes %v", calls)
	}
	// Back to Luna: the same file, not a new one.
	lp, _ := style.LoadTheme("luna")
	a.SetLook(lp.Look())
	a.PumpOnce()
	calls = o.FrameCalls().Palettes
	if len(calls) != 3 || calls[2] != luna {
		t.Fatalf("back to Luna: palettes %v, want %s again", calls, luna)
	}
	if files, _ := filepath.Glob(filepath.Join(cache, "uitoolkit", "colors", "*")); len(files) != 2 {
		t.Fatalf("files %v, want Luna's and Aqua's", files)
	}
}

// Nothing is named, and nothing written, where the desktop takes no palette
// or UITK_DECORATION_PALETTE=0 says not to.
func TestDecorationPaletteOnlyWhereTheDesktopTakesOne(t *testing.T) {
	_, _, o, cache := dressRig(t, "luna", false)
	if calls := o.FrameCalls().Palettes; len(calls) != 0 {
		t.Fatalf("palettes %v on a desktop that takes none", calls)
	}
	if _, err := os.Stat(filepath.Join(cache, "uitoolkit")); !os.IsNotExist(err) {
		t.Fatalf("a scheme was written for nobody: %v", err)
	}
	t.Setenv(EnvDecorationPalette, "0")
	p, _ := style.LoadTheme("luna")
	a := New(Options{Look: p.Look(), Headless: true})
	w, _ := a.NewWindow(platform.WindowOptions{Title: "Off", Headless: true})
	off := w.Surface().(*platform.Offscreen)
	off.SimulateDecorationPalette(true)
	a.SetLook(p.Look())
	a.PumpOnce()
	if calls := off.FrameCalls().Palettes; len(calls) != 0 {
		t.Fatalf("palettes %v with %s=0", calls, EnvDecorationPalette)
	}
}

func square(n int) *paintengine2d.Image { return paintengine2d.NewImage(n, n) }

// The application's icon goes to every window, a window's own wins over it,
// and dropping the window's own gives the application's back.
func TestWindowIcons(t *testing.T) {
	a, w, o, _ := dressRig(t, "win95", false)
	if calls := o.FrameCalls().Icons; len(calls) != 0 {
		t.Fatalf("icons %v before any was given", calls)
	}
	a.SetIcon(square(32), square(16), paintengine2d.NewImage(20, 10))
	w2, _ := a.NewWindow(platform.WindowOptions{Title: "Second", Headless: true})
	o2 := w2.Surface().(*platform.Offscreen)
	for i, s := range []*platform.Offscreen{o, o2} {
		calls := s.FrameCalls().Icons
		if len(calls) != 1 || len(calls[0]) != 2 || calls[0][0] != 16 || calls[0][1] != 32 {
			t.Fatalf("window %d: icons %v, want the application's 16 and 32 (the oblong dropped)", i, calls)
		}
	}
	w.SetIcon(square(48))
	if calls := o.FrameCalls().Icons; len(calls) != 2 || len(calls[1]) != 1 || calls[1][0] != 48 {
		t.Fatalf("icons %v, want the window's own 48", calls)
	}
	a.SetIcon(square(24))
	if calls := o.FrameCalls().Icons; len(calls) != 2 {
		t.Fatalf("the application's new icon replaced the window's own: %v", calls)
	}
	if calls := o2.FrameCalls().Icons; len(calls) != 2 || calls[1][0] != 24 {
		t.Fatalf("second window: icons %v, want the application's 24", calls)
	}
	w.SetIcon()
	if calls := o.FrameCalls().Icons; len(calls) != 3 || calls[2][0] != 24 {
		t.Fatalf("icons %v, want the application's 24 back", calls)
	}
}
