package app

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/diag"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// The toolkit says when it did not do what it was told.
//
// Three layers may overrule an application without its knowing: the
// look's era decides how a title bar meets the frame, build tags decide
// which theme packs exist, the compositor decides who draws the frame.
// Each is a reasonable rule and each was silent, which is how a sample
// that asked for a browser's shape correctly, in four separate calls,
// still came out with a caption row above its tabs and nothing to read.
func TestDiagnosticsSayWhenTheToolkitOverruledTheApp(t *testing.T) {
	t.Run("a pinned pack this build cannot draw", func(t *testing.T) {
		diag.Reset()
		t.Cleanup(diag.Reset)
		New(Options{Headless: true, Theme: style.ThemeOverride{Pack: "no-such-pack-anywhere"}})
		f := findArea(t, "theme")
		if !strings.Contains(f.Asked, "no-such-pack-anywhere") {
			t.Errorf("the finding does not name the pack asked for: %v", f)
		}
		if !strings.Contains(f.Fix, "theme_engine") {
			t.Errorf("the finding does not say how to get it: %v", f)
		}
		if f.Level != diag.Warn {
			t.Errorf("a pack the app asked for and did not get is a warning, got %v", f.Level)
		}
	})

	t.Run("a title bar demoted under a stacked look's strip", func(t *testing.T) {
		diag.Reset()
		t.Cleanup(diag.Reset)
		a := New(Options{Headless: true})
		w, err := a.NewWindow(platform.WindowOptions{
			Title: "probe", Width: 800, Height: 500,
			Decorations: platform.DecorationsClient,
		})
		if err != nil {
			t.Fatal(err)
		}
		hb := widgets.NewHeaderBar(nil, widgets.NewLabel("TABS"), nil)
		hb.ShowTitle = false
		w.SetTitleBar(hb)
		if !hb.Stacked() {
			t.Skip("the default pack in this build is not a stacked one")
		}
		// The note waits for the first layout on purpose: an application
		// configures a window over several calls, and one that settles
		// its caption on the next line has not been overruled at all.
		if got := diag.Findings(); len(got) != 0 {
			t.Errorf("reported before the application had finished configuring: %v", got)
		}
		w.Inject(platform.Event{Kind: platform.EventResize, Width: 800, Height: 500})
		a.PumpOnce()
		f := findArea(t, "caption")
		if !strings.Contains(f.Fix, "CaptionMerged") {
			t.Errorf("the finding does not name the call that fixes it: %v", f)
		}

		// And an application that settles the shape before its first
		// layout is never told about the era at all.
		diag.Reset()
		w2, err := a.NewWindow(platform.WindowOptions{
			Title: "settled", Width: 800, Height: 500,
			Decorations: platform.DecorationsClient,
		})
		if err != nil {
			t.Fatal(err)
		}
		hb2 := widgets.NewHeaderBar(nil, widgets.NewLabel("TABS"), nil)
		hb2.ShowTitle = false
		w2.SetTitleBar(hb2)
		w2.SetCaptionStyle(style.CaptionMerged)
		w2.Inject(platform.Event{Kind: platform.EventResize, Width: 800, Height: 500})
		a.PumpOnce()
		for _, g := range diag.Findings() {
			if g.Area == "caption" {
				t.Errorf("told an application about the era it had already overruled: %v", g)
			}
		}
	})

	t.Run("the same finding is reported once, not per frame", func(t *testing.T) {
		diag.Reset()
		t.Cleanup(diag.Reset)
		f := diag.Finding{Level: diag.Warn, Area: "icons", Asked: "a", Got: "b", Fix: "c"}
		for i := 0; i < 50; i++ {
			diag.Report(f)
		}
		if n := len(diag.Findings()); n != 1 {
			t.Errorf("reported %d times, want 1", n)
		}
	})
}

func findArea(t *testing.T, area string) diag.Finding {
	t.Helper()
	for _, f := range diag.Findings() {
		if f.Area == area {
			return f
		}
	}
	t.Fatalf("nothing reported for %q; got %v", area, diag.Findings())
	return diag.Finding{}
}
