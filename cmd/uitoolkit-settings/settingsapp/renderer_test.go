package settingsapp

import (
	"os"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// pickRenderer chooses an item of the paint chooser by the word on it.
func pickRenderer(t *testing.T, w *app.Window, item string) {
	t.Helper()
	cb := namedCombo(w.Content(), "Paint renderer")
	if cb == nil {
		t.Fatal("no paint chooser on the page")
	}
	for i, it := range cb.Items {
		if it == item {
			if cb.OnChange == nil {
				t.Fatalf("the %q renderer does nothing", item)
			}
			cb.OnChange(i)
			return
		}
	}
	t.Fatalf("no %q in the paint chooser (%q)", item, cb.Items)
}

// The fourth chooser stages a renderer and Apply writes it, the way every
// other control on this page works — and the running application takes it
// for the windows it opens next.
func TestSettingsRendererChooserWritesLookJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(app.RendererEnv, "")
	t.Cleanup(func() { platform.SetPaintPref("") })
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)

	cb := namedCombo(w.Content(), "Paint renderer")
	if cb == nil {
		t.Fatal("no paint chooser on the page")
	}
	// Three answers, in the order a person meets them: let the toolkit
	// decide, or say which.
	if got := strings.Join(cb.Items, " "); got != "Auto GPU CPU" {
		t.Fatalf("the chooser offers %q", got)
	}
	if cb.Selected != 0 {
		t.Fatalf("nothing saved: the chooser should open on Auto, not %d", cb.Selected)
	}
	// The word in front of it is on the page and is inside the name a
	// screen reader says, as everywhere else on this row.
	if findRowLabel(w.Content(), "Paint") == nil {
		t.Error("the word Paint is not in front of the chooser")
	}
	if !strings.Contains(strings.ToLower(cb.AccessibleName()), "paint") {
		t.Errorf("the page says %q and a screen reader says %q", "Paint", cb.AccessibleName())
	}

	// Staging writes nothing, as with every other setting here.
	pickRenderer(t, w, "CPU")
	a.PumpOnce()
	if got := style.LoadAppearance().Renderer; got != style.RendererAuto {
		t.Fatalf("staging wrote look.json: %q", got)
	}
	if !findApply(w.Content()).Enabled() {
		t.Fatal("Apply should be enabled with a renderer staged")
	}

	clickApply(t, w)
	a.PumpOnce()
	if got := style.LoadAppearance().Renderer; got != style.RendererCPU {
		t.Fatalf("Apply saved %q", got)
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil || !strings.Contains(string(raw), `"renderer": "cpu"`) {
		t.Fatalf("look.json %s %v", raw, err)
	}
	// The running application asks for it from now on. It cannot give it
	// to the window this page is drawn in — see
	// TestSettingsRendererSaysHowFarItReaches — but the next window it
	// opens is bound with it.
	if got := a.Renderer(); got != style.RendererCPU {
		t.Fatalf("the application asks for %q after Apply", got)
	}
	if got := platform.PaintPref(); got != "cpu" {
		t.Fatalf("new surfaces would bind %q", got)
	}
	if cb := namedCombo(w.Content(), "Paint renderer"); cb == nil || cb.Selected != 2 {
		t.Fatal("the chooser should show the saved choice after Apply")
	}

	// Back to Auto, which is the default and so leaves the file alone.
	pickRenderer(t, w, "Auto")
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if raw, _ := os.ReadFile(style.AppearancePath()); strings.Contains(string(raw), "renderer") {
		t.Fatalf("auto is left out of look.json: %s", raw)
	}
	if got := platform.PaintPref(); got != "auto" {
		t.Fatalf("new surfaces would bind %q", got)
	}
}

// The chooser says three things a preference on its own never says: what
// is really painting this window, that the windows already open keep
// what they have, and — when it is set — that UITK_PAINT is what decided
// it. A chooser that silently did nothing until the next start is the
// shape of bug this page has been paying for.
func TestSettingsRendererSaysHowFarItReaches(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(app.RendererEnv, "")
	t.Cleanup(func() { platform.SetPaintPref("") })
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 860)
	cb := namedCombo(w.Content(), "Paint renderer")
	if cb == nil {
		t.Fatal("no paint chooser on the page")
	}
	// The tip and the accessible description are the same sentence: the
	// eye and the ear get the same answer.
	if cb.Tip == "" || cb.AccessibleDescription() != cb.Tip {
		t.Fatalf("tip %q description %q", cb.Tip, cb.AccessibleDescription())
	}
	// What is actually painting. These tests run offscreen, so it is the
	// CPU, and the page must say so rather than repeating "Auto" back.
	if !strings.Contains(cb.Tip, "This window is painting on the CPU.") {
		t.Errorf("the chooser does not say what is really painting: %q", cb.Tip)
	}
	// How far Apply reaches.
	if !strings.Contains(cb.Tip, "keep the device they were created with") {
		t.Errorf("the chooser does not say that open windows keep theirs: %q", cb.Tip)
	}
	if !strings.Contains(cb.Tip, "opened after") {
		t.Errorf("the chooser does not say which windows get it: %q", cb.Tip)
	}
	// And nothing about an environment variable that is not set.
	if strings.Contains(cb.Tip, app.RendererEnv) {
		t.Errorf("UITK_PAINT is unset and the chooser mentions it: %q", cb.Tip)
	}
}

// Started under UITK_PAINT, the page says the environment is in charge —
// and goes on showing, staging and saving the user's own choice, because
// look.json belongs to the desktop and the variable to this one run.
func TestSettingsRendererSaysWhenUITKPaintOverrides(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(app.RendererEnv, "cpu")
	t.Cleanup(func() { platform.SetPaintPref("") })
	saved := style.DefaultAppearance()
	saved.Renderer = style.RendererGPU
	if err := style.SaveAppearance(saved); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)

	cb := namedCombo(w.Content(), "Paint renderer")
	if cb == nil {
		t.Fatal("no paint chooser on the page")
	}
	if cb.Selected != 1 {
		t.Fatalf("the chooser shows the saved GPU, not the variable: %d", cb.Selected)
	}
	if !strings.Contains(cb.Tip, app.RendererEnv+"=cpu is set here") {
		t.Errorf("the chooser does not say the environment overrides it: %q", cb.Tip)
	}
	// Apply still writes what the page says, and the variable still wins
	// over it for this process.
	pickRenderer(t, w, "Auto")
	a.PumpOnce()
	clickApply(t, w)
	a.PumpOnce()
	if raw, _ := os.ReadFile(style.AppearancePath()); strings.Contains(string(raw), "renderer") {
		t.Fatalf("Apply should have written auto (nothing): %s", raw)
	}
	if got := platform.PaintPref(); got != "cpu" {
		t.Fatalf("UITK_PAINT=cpu must still decide the device, got %q", got)
	}
}
