package settingsapp

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// overlayText is everything the open dialog says, run together.
func overlayText(w *app.Window) string {
	var b strings.Builder
	if w.Overlay() == nil {
		return ""
	}
	widget.Walk(w.Overlay(), func(c widget.Component) {
		switch v := c.(type) {
		case *widgets.Label:
			b.WriteString(v.Text + "\n")
		case *widgets.LinkButton:
			b.WriteString(v.Text + " " + v.URI + "\n")
		case *widgets.Panel:
			b.WriteString(v.Title + "\n")
		}
	})
	return b.String()
}

func openAbout(t *testing.T, a *app.Application, w *app.Window) {
	t.Helper()
	about := findAboutButton(w.Content())
	if about == nil || about.OnClick == nil {
		t.Fatal("no About button at the foot of the window")
	}
	about.OnClick()
	a.PumpOnce()
	if w.Overlay() == nil {
		t.Fatal("About opened nothing")
	}
}

// The dialog has to be worth opening: the version, the licence, the
// repository and — the part a constant cannot give — what this copy
// and this window actually are.
func TestAboutDialogSaysWhatItIsAbout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)
	openAbout(t, a, w)
	said := overlayText(w)

	for _, want := range []string{
		"About uitoolkit",
		uitoolkit.Version,
		LicenseName,
		"github.com/codemodify/uitoolkit",
		"paintengine2d",
	} {
		if !strings.Contains(said, want) {
			t.Errorf("the About dialog never says %q:\n%s", want, said)
		}
	}
	// The honest, specific part, counted at the moment it is shown
	// rather than written down: how many packs, how many engines, and
	// which backend and paint device this window is on.
	packs, engines := len(style.ListBuiltinThemes()), len(style.EngineIDs())
	for _, want := range []string{
		fmt.Sprintf("%d theme packs", packs),
		fmt.Sprintf("%d engines", engines),
		a.BackendName(),
	} {
		if !strings.Contains(said, want) {
			t.Errorf("the About dialog never says %q:\n%s", want, said)
		}
	}
	if packs < 100 || engines < 20 {
		t.Errorf("the counts look wrong: %d packs, %d engines", packs, engines)
	}
	// The sentence about where windows come from names every platform
	// the toolkit has a backend for, or none of them. It named X11 and
	// Wayland alone for a while after Win32 and AppKit landed, so on a
	// Mac this dialog contradicted the paragraph under it — which
	// names the backend the window is really on. A prose list of
	// platforms goes stale the moment a port lands, and nothing but
	// this notices.
	var named, missing []string
	for _, n := range []string{"X11", "Wayland", "Win32", "AppKit"} {
		if strings.Contains(said, n) {
			named = append(named, n)
		} else {
			missing = append(missing, n)
		}
	}
	if len(named) > 0 && len(missing) > 0 {
		t.Errorf("About names %v but not %v:\n%s", named, missing, said)
	}
	// The window this dialog is in is on the CPU here, and the sentence
	// says so rather than reciting the preference.
	if w.PaintBackend() == style.RendererCPU && !strings.Contains(said, "CPU") {
		t.Errorf("a window painting on the CPU does not say so:\n%s", said)
	}

	// The repository is a link somebody can follow, not a string in a
	// paragraph, and it is named for a screen reader.
	var link *widgets.LinkButton
	widget.Walk(w.Overlay(), func(c widget.Component) {
		if l, ok := c.(*widgets.LinkButton); ok {
			link = l
		}
	})
	if link == nil {
		t.Fatal("the repository is not a link")
	}
	if link.URI != RepoURL {
		t.Errorf("the link goes to %q, want %q", link.URI, RepoURL)
	}
	if link.AccessibleName() == "" {
		t.Error("the repository link has no name")
	}

	// Close closes it, and the page behind it is untouched: About is
	// the one button on this page that changes nothing.
	before := style.LoadAppearance()
	close := findOverlayButton(w, "Close")
	if close == nil || close.OnClick == nil {
		t.Fatal("the About dialog has no Close")
	}
	close.OnClick()
	a.PumpOnce()
	if w.Overlay() != nil {
		t.Error("Close left the About dialog up")
	}
	if style.LoadAppearance() != before {
		t.Error("opening About changed the saved appearance")
	}
}

// The licence named in the dialog is the licence in the file.
func TestAboutNamesTheLicenceInTheRepository(t *testing.T) {
	raw, err := os.ReadFile("../../../LICENSE")
	if err != nil {
		t.Skipf("no LICENSE beside the package: %v", err)
	}
	if !strings.Contains(string(raw), LicenseName) {
		t.Errorf("LICENSE does not call itself %q", LicenseName)
	}
	if !strings.Contains(string(raw), "github.com/codemodify/uitoolkit") {
		t.Error("LICENSE does not name the repository the dialog links to")
	}
}

// About stands in front of Apply, in the row and in the Tab ring: the
// button that writes is the last one a Tab reaches, which is where
// every desktop puts it.
func TestAboutStandsInFrontOfApply(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 720, 580)
	about, apply := findAboutButton(w.Content()), findApply(w.Content())
	if about == nil || apply == nil {
		t.Fatal("the foot row is not About and Apply")
	}
	if about.Bounds().Min.X >= apply.Bounds().Min.X {
		t.Errorf("About is at x %v and Apply at %v: About is not in front", about.Bounds().Min.X, apply.Bounds().Min.X)
	}
	if widget.DeviceOrigin(about).Y != widget.DeviceOrigin(apply).Y {
		t.Error("About and Apply are not on the same row")
	}
	// Neither is squeezed at the 720x580 minimum, where the whole row
	// is 704 logical pixels wide.
	for _, b := range []*widgets.Button{about, apply} {
		if b.Bounds().Dx() < b.Measure(layout.Unbounded()).X-0.51 {
			t.Errorf("%q is squeezed to %v of %v", b.Text, b.Bounds().Dx(), b.Measure(layout.Unbounded()).X)
		}
	}

	// Apply is off until something is staged, and a disabled button is
	// not a tab stop — so the ring is read with a theme staged, which
	// is the state a user reaching for Apply is in. About is a tab stop
	// either way: it is never disabled.
	if ring := widget.Focusables(w.Content()); !hasFocusable(ring, about) {
		t.Error("About is not a tab stop with nothing staged")
	}
	clickTheme(t, w, "Windows 95")
	a.PumpOnce()
	about, apply = findAboutButton(w.Content()), findApply(w.Content())
	ring := widget.Focusables(w.Content())
	ai, pi := -1, -1
	for i, c := range ring {
		switch c {
		case widget.Component(about):
			ai = i
		case widget.Component(apply):
			pi = i
		}
	}
	if ai < 0 || pi < 0 {
		t.Fatalf("the Tab ring does not reach About (%d) and Apply (%d)", ai, pi)
	}
	if ai+1 != pi {
		t.Errorf("About is stop %d and Apply is stop %d: nothing should come between them", ai, pi)
	}
	if pi != len(ring)-1 {
		t.Errorf("Apply is stop %d of %d: the button that writes is the last one", pi, len(ring)-1)
	}
}

// The About *page* is not back. It was one of four behind a sidebar and
// it went when the four became one; this is a dialog now.
func TestAboutIsADialogAndNotAPage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 860)
	if findLabelWith(w.Content(), uitoolkit.Version) {
		t.Error("the version is back on the page; it is in the About dialog and in -version")
	}
	if findLabelWith(w.Content(), LicenseName) {
		t.Error("the licence is back on the page")
	}
	widget.Walk(w.Content(), func(c widget.Component) {
		if l, ok := c.(*widgets.LinkButton); ok && !insidePreview(c) {
			t.Errorf("a link to %q is back on the page", l.URI)
		}
	})
}

func hasFocusable(ring []widget.Component, c widget.Component) bool {
	for _, f := range ring {
		if f == c {
			return true
		}
	}
	return false
}
