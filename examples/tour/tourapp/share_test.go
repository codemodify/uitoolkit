package demo

import (
	"regexp"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// sharePages builds the pages that show part of the appearance in a tour
// window, so that each is listening.
func sharePages(t *testing.T, a *app.Application, win *app.Window) (*framesPage, *skinsPage, *accessPage) {
	t.Helper()
	tour := stateOf(t, win)
	for i := 0; i < tour.strip.Len(); i++ {
		tour.strip.Select(i)
		a.PumpOnce()
	}
	return tour.page(pageFrames).(*framesPage), tour.page(pageSkins).(*skinsPage), tour.page(pageAccess).(*accessPage)
}

func factIs(text, name, value string) bool {
	return regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + ` +` + regexp.QuoteMeta(value) + `$`).MatchString(text)
}

// The tour's windows share the application's one appearance: a change
// made in either — the pack, who frames the window, where its buttons go,
// motion — shows at once in the other's choices and readouts, "Back to
// where we started" means where the tour started from whichever window it
// is pressed in, and closing a window that is not the last leaves the look
// to the ones still open.
func TestTourWindowsShareOneAppearance(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(style.AnimationsEnv, "")
	t.Cleanup(func() { style.SetReduceMotion(false) })
	luna, _ := style.LoadTheme("luna")
	a := uitoolkit.New(uitoolkit.Options{Look: luna.Look(), Headless: true, Scale: 1, DisableLookWatch: true})
	pages := []int{pageFrames, pageSkins, pageAccess}
	one, err := TourWindow(a, pages...)
	if err != nil {
		t.Fatal(err)
	}
	defer one.Close()
	a.PumpOnce()
	f1, s1, m1 := sharePages(t, a, one)

	// A change of pack before the second window opens: the tour still
	// started in Luna.
	s1.t.apply(func(ap *style.Appearance) { ap.Name, ap.Theme = "win95", style.ThemeLight })
	a.PumpOnce()

	two, err := TourWindow(a, pages...)
	if err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	f2, s2, m2 := sharePages(t, a, two)
	if !factIs(s2.facts.Text, "in use now", "win95") {
		t.Fatalf("the second window's readout:\n%s", s2.facts.Text)
	}

	// The pack, from the second window: the first one's readout follows.
	s2.t.apply(func(ap *style.Appearance) { ap.Name, ap.Theme = "aqua", style.ThemeLight })
	a.PumpOnce()
	if !factIs(s1.facts.Text, "in use now", "aqua") {
		t.Errorf("the first window still reads\n%s", s1.facts.Text)
	}

	// The frame and the caption buttons, from the second window.
	f2.decor.Select(2)
	f2.caps.Select(1)
	a.PumpOnce()
	if f1.decor.Selected() != 2 || f1.caps.Selected() != 1 {
		t.Errorf("the first window's choices are %d and %d, want 2 and 1", f1.decor.Selected(), f1.caps.Selected())
	}
	if !factIs(f1.facts.Text, "asked for", "the toolkit's frame") || !factIs(f1.facts.Text, "buttons from", "the theme") {
		t.Errorf("the first window's frame readout:\n%s", f1.facts.Text)
	}

	// Motion, from the first window.
	m1.motion.OnChange(false)
	a.PumpOnce()
	if m2.motion.On || !factIs(m2.facts.Text, "animations", "no") {
		t.Errorf("the second window's motion switch is on=%v:\n%s", m2.motion.On, m2.facts.Text)
	}
	m1.motion.OnChange(true)
	a.PumpOnce()

	// Closing the second window leaves the look as it is: the first is
	// still open and in it.
	two.Inject(platform.Event{Kind: platform.EventClose})
	a.PumpOnce()
	if !two.Closed() {
		t.Fatal("the second window did not close")
	}
	if got := a.Appearance().Name; got != "aqua" {
		t.Errorf("closing one of two tour windows put the look back to %q", got)
	}
	if a.Decorations() != style.DecorationsToolkit {
		t.Errorf("closing one of two tour windows reset the frame to %q", a.Decorations())
	}

	// "Back to where we started" is Luna, where the tour began, from the
	// window that opened in Windows 95 as much as from any.
	s1.t.apply(s1.restore)
	a.PumpOnce()
	if got := a.Appearance().Name; got != "luna" {
		t.Errorf("back to where we started is %q, want luna", got)
	}
}
