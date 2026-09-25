package settingsapp

import (
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// classic is the look of c (a window, or a widget on the page) as a pack.
func classicOf(t *testing.T, lk style.LookAndFeel) *style.Classic {
	t.Helper()
	c, ok := lk.(*style.Classic)
	if !ok {
		t.Fatalf("look %T is not a pack", lk)
	}
	return c
}

// Apply puts Settings' own window into the look it has just applied: the
// window, the widgets outside the preview's ThemeScope and the frame all
// wear the new pack without a restart. Staging must not do any of that —
// that is what the ThemeScope beside them is for.
func TestApplyRestylesSettingsItself(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, scale := range []float32{1, 1.75} {
		// Each pass starts from the saved light starter: the one before
		// it applied win95 to the same look.json.
		if err := style.SaveAppearance(style.Appearance{Theme: style.ThemeLight, Corners: style.CornersRound}.Normalize()); err != nil {
			t.Fatal(err)
		}
		a := uitoolkit.New(uitoolkit.Options{Look: style.PreferredLook(), Headless: true, Scale: scale, DisableLookWatch: true})
		w, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: 1024, Height: 780, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(SettingsApp(a, w))
		a.PumpOnce()

		// The one app that writes look.json is the one app that does not
		// watch it: two mechanisms would restyle this window twice.
		if a.WatchingLook() {
			t.Fatal("Settings must not watch the file it writes")
		}

		before := classicOf(t, w.Look())
		if before.Pack() == "win95" {
			t.Fatal("pick a starting pack that is not the one being applied")
		}

		clickTheme(t, w, "Windows 95")
		a.PumpOnce()
		if got := classicOf(t, w.Look()).Pack(); got != before.Pack() {
			t.Fatalf("staging restyled the window to %q — the preview's scope is what stages", got)
		}
		if got := classicOf(t, previewScope(t, w).Look()).Pack(); got != "win95" {
			t.Fatalf("the preview should show the staged pack, got %q", got)
		}

		clickApply(t, w)
		a.PumpOnce()

		after := classicOf(t, w.Look())
		if after.Pack() != "win95" {
			t.Errorf("at %g×: the window still wears %q after Apply", scale, after.Pack())
		}
		if after.Palette().Background == before.Palette().Background {
			t.Errorf("at %g×: the window's background did not change", scale)
		}
		// The window keeps its own display scale through the switch: the
		// look is applied to the app unscaled and rebuilt per window.
		if got := after.Scale(); got != scale {
			t.Errorf("at %g×: the window came back at %g×", scale, got)
		}
		// And so does everything on the page that is not in the scope.
		apply := findApply(w.Content())
		if apply == nil {
			t.Fatal("no Apply button")
		}
		if got := classicOf(t, apply.Look()).Pack(); got != "win95" {
			t.Errorf("at %g×: the Apply button still wears %q", scale, got)
		}
		var checked bool
		widget.Walk(w.Content(), func(c widget.Component) {
			if b, ok := c.(*widgets.Checkbox); ok && b.Text == "OS colors" {
				checked = true
				if got := classicOf(t, b.Look()).Pack(); got != "win95" {
					t.Errorf("at %g×: the options row still wears %q", scale, got)
				}
			}
		})
		if !checked {
			t.Error("the options row was not found")
		}
		// Nothing left staged, so the preview and the window agree.
		if got := classicOf(t, previewScope(t, w).Look()).Pack(); got != "win95" {
			t.Errorf("at %g×: the preview shows %q after Apply", scale, got)
		}
		w.Close()
	}
}
