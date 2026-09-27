package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The theme cascade where it meets a window: the application's own
// level, the window's frame, and popups on surfaces of their own.

func packName(t *testing.T, lk style.LookAndFeel) string {
	t.Helper()
	c, ok := lk.(*style.Classic)
	if !ok {
		t.Fatalf("not a Classic look: %T", lk)
	}
	return c.Pack()
}

// An application states its own level: the pack is its, everything it
// did not state still comes from the desktop's look.json.
func TestAppThemeOverridesTheDesktop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.Appearance{
		Name: "win95", Theme: style.ThemeLight,
		Corners: style.CornersSquare, Icons: style.IconSetSharp, IconSize: style.IconSizeLarge,
	}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1, Theme: style.ThemeOverride{Pack: "luna"}})
	if got := packName(t, a.Look()); got != "luna" {
		t.Fatalf("the application's pack: %s", got)
	}
	ap := a.Appearance()
	if ap.Corners != style.CornersSquare || ap.Icons != style.IconSetSharp || ap.IconSize != style.IconSizeLarge {
		t.Fatalf("what the app did not state should still be the desktop's: %+v", ap)
	}
	if !a.WatchingLook() {
		t.Fatal("an app that states only part of an appearance still follows look.json for the rest")
	}
}

// The application's level outlives a look.json change: the user applies
// another pack in Settings and the app that pinned one keeps it, while
// everything it did not pin follows.
func TestAppThemeSurvivesAReload(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.Appearance{Name: "win95", Corners: style.CornersSquare}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1, Theme: style.ThemeOverride{Pack: "luna"}})
	w, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.SetContent(widgets.NewLabel("hello"))
	a.PumpOnce()

	if err := style.SaveAppearance(style.Appearance{Name: "aqua", Corners: style.CornersRound}); err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()

	if got := packName(t, a.Look()); got != "luna" {
		t.Fatalf("the app's own pack did not survive the reload: %s", got)
	}
	if got := style.LookAppearance(a.Look()).Corners; got != style.CornersRound {
		t.Fatalf("what the app did not state should have followed the file: %s", got)
	}
	if got := packName(t, w.Look()); got != "luna" {
		t.Fatalf("the window: %s", got)
	}
}

// SetTheme replaces the application's level; clearing it hands the app
// back to the desktop.
func TestAppSetThemeAndClear(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.Appearance{Name: "win95"}); err != nil {
		t.Fatal(err)
	}
	a := New(Options{Headless: true, Scale: 1})
	if got := packName(t, a.Look()); got != "win95" {
		t.Fatalf("start: %s", got)
	}
	a.SetTheme(style.ThemeOverride{Pack: "metal-ocean"})
	if got := packName(t, a.Look()); got != "metal-ocean" {
		t.Fatalf("after SetTheme: %s", got)
	}
	a.SetTheme(style.ThemeOverride{})
	if got := packName(t, a.Look()); got != "win95" {
		t.Fatalf("after clearing: %s", got)
	}
}

// Options.Theme on top of an explicit Look: "this look, with these parts
// written over it", and still no watcher.
func TestAppThemeOverAnExplicitLook(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := New(Options{
		Look: style.DarkLook(), Headless: true, Scale: 1,
		Theme: style.ThemeOverride{Pack: "aqua", Corners: style.CornersSquare},
	})
	if a.WatchingLook() {
		t.Fatal("an explicit Look must still not watch the file")
	}
	ap := style.LookAppearance(a.Look())
	if ap.Name != "aqua" || ap.Corners != style.CornersSquare {
		t.Fatalf("%+v", ap)
	}
}

// A themed subtree does not touch the window's frame: the frame is drawn
// from the window's look, which the desktop and the compositor need one
// answer for.
func TestCascadeLeavesTheWindowFrameAlone(t *testing.T) {
	a := New(Options{Look: mustPack(t, "luna"), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "framed", Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	pane := widgets.NewPanel("Pane", widgets.NewButton("In the pane", nil))
	w.SetContent(widgets.NewColumn(pane))
	a.PumpOnce()

	before := packName(t, w.Look())
	widget.SetTheme(pane, style.ThemeOverride{Pack: "system7"})
	a.PumpOnce()

	if got := packName(t, w.Look()); got != before {
		t.Fatalf("the window's look changed with a subtree's: %s -> %s", before, got)
	}
	if w.caption != nil {
		if got := packName(t, w.caption.Look()); got != before {
			t.Fatalf("the frame's caption: %s", got)
		}
	}
	if got := packName(t, pane.Look()); got != "system7" {
		t.Fatalf("the pane: %s", got)
	}
	if got := packName(t, w.Content().Look()); got != before {
		t.Fatalf("the content around the pane: %s", got)
	}
}

// An in-window panel that stands in for a window (Panel.Window) *is* a
// widget, so its caption and its frame follow the cascade — unlike a
// real window's, which is the compositor's business.
func TestCascadeReachesAnInWindowPanel(t *testing.T) {
	a := New(Options{Look: mustPack(t, "luna"), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 500, Height: 400, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	inner := widgets.NewPanel("A window inside a window", widgets.NewLabel("body"))
	inner.Window = true
	w.SetContent(widgets.NewColumn(inner))
	a.PumpOnce()
	widget.SetTheme(inner, style.ThemeOverride{Pack: "aqua"})
	a.PumpOnce()
	if got := packName(t, inner.Look()); got != "aqua" {
		t.Fatalf("the in-window panel: %s", got)
	}
	if got := packName(t, w.Look()); got != "luna" {
		t.Fatalf("the real window around it: %s", got)
	}
}

// The display scale is the window's everywhere, however the cascade is
// nested: a scope cannot make one part of a window HiDPI and another not.
func TestCascadeKeepsTheDisplayScale(t *testing.T) {
	a := New(Options{Look: mustPack(t, "luna"), Headless: true, Scale: 2, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	leaf := widgets.NewButton("Leaf", nil)
	inner := widgets.NewPanel("", widgets.NewColumn(leaf))
	outer := widgets.NewPanel("", widgets.NewColumn(inner))
	w.SetContent(outer)
	a.PumpOnce()
	widget.SetTheme(outer, style.ThemeOverride{Pack: "aqua"})
	widget.SetTheme(inner, style.ThemeOverride{Pack: "win95"})
	a.PumpOnce()

	for _, c := range []widget.Component{outer, inner, leaf} {
		lk := c.Look().(*style.Classic)
		if lk.Scale() != 2 {
			t.Fatalf("%T is at %gx, the window is at 2x", c, lk.Scale())
		}
	}
	if w.Look().(*style.Classic).Scale() != 2 {
		t.Fatal("the window")
	}
}

// A popup on a surface of its own is painted from the look of the widget
// that opened it, not the window's: this is the place a cascade that
// only walks the widget tree breaks, because a popup surface is not in
// the window's tree at all.
func TestCascadePopupSurfaceCarriesTheScope(t *testing.T) {
	a := New(Options{Look: mustPack(t, "win95"), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if o, ok := w.Surface().(*platform.Offscreen); ok {
		o.SimulatePopups(platform.FrameRect{W: 1280, H: 760})
	}
	combo := widgets.NewComboBox([]string{"one", "two", "three"}, 0, nil)
	pane := widgets.NewPanel("Pane", combo)
	w.SetContent(widgets.NewColumn(pane))
	a.PumpOnce()
	widget.SetTheme(pane, style.ThemeOverride{Pack: "aqua"})
	a.PumpOnce()

	combo.Open()
	a.PumpOnce()
	pop := w.Popup()
	if pop == nil {
		t.Fatal("no popup")
	}
	if got := packName(t, w.layerLook(pop)); got != "aqua" {
		t.Fatalf("the popup surface paints in %s, not the pane's aqua", got)
	}
	// And the window it hangs from is untouched.
	if got := packName(t, w.Look()); got != "win95" {
		t.Fatalf("the window: %s", got)
	}
}

// A tooltip is in the theme of the widget it came from.
func TestCascadeTooltipCarriesTheScope(t *testing.T) {
	a := New(Options{Look: mustPack(t, "win95"), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	btn := widgets.NewButton("Hover me", nil)
	btn.Tip = "a tip"
	pane := widgets.NewPanel("Pane", btn)
	w.SetContent(widgets.NewColumn(pane))
	a.PumpOnce()
	widget.SetTheme(pane, style.ThemeOverride{Pack: "aqua"})
	a.PumpOnce()

	w.tipHover = btn
	w.showTip("a tip", paintengine2d.Pt(20, 20))
	tip := w.Tooltip()
	if tip == nil {
		t.Fatal("no tooltip")
	}
	if got := packName(t, tip.Look()); got != "aqua" {
		t.Fatalf("the tip: %s", got)
	}
}

func mustPack(t *testing.T, name string) style.LookAndFeel {
	t.Helper()
	p, ok := style.LoadTheme(name)
	if !ok {
		t.Fatalf("unknown pack %q", name)
	}
	return p.Look()
}

// The memo is what keeps Look off the hot path, and it is only as good
// as the number of times it is thrown away. A window that is laying out,
// painting, hovering and scrolling must not move the generation at all:
// if it does, every component in the process re-resolves on its next
// Look, on every frame.
func TestCascadeMemoSurvivesASteadyWindow(t *testing.T) {
	a := New(Options{Look: mustPack(t, "luna"), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 360, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	list := widgets.NewListView(80, func(i int) string { return "row" }, nil)
	pane := widgets.NewPanel("Pane", list)
	w.SetContent(widgets.NewColumn(pane))
	a.PumpOnce()
	widget.SetTheme(pane, style.ThemeOverride{Pack: "aqua"})
	a.PumpOnce()

	before := widget.LookGeneration()
	for i := 0; i < 20; i++ {
		w.Invalidate(nil, paintengine2d.Rect{})
		list.ScrollTo(float32(i % 2 * 40))
		w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(40, float32(20+i))})
		a.PumpOnce()
	}
	if got := widget.LookGeneration() - before; got != 0 {
		t.Fatalf("a steady window threw the look memo away %d times", got)
	}
	if got := packName(t, list.Look()); got != "aqua" {
		t.Fatalf("the list: %s", got)
	}
}
