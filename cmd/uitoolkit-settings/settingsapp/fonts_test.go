package settingsapp

import (
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The two typeface choosers, and why there are two.
//
// The toolkit has two font roles and every pack names a face for each —
// an interface face and a monospaced one. One chooser over one list
// could set one of them and leave the other to the pack, or set both to
// the same family, and a page that did either under the word "Font"
// would be lying about half of what it changed.
//
// The list leads with what the toolkit carries, which is what makes it
// the same list on every machine before it is the machine's list: the
// first item is not a font at all but the pack's own typography, and the
// two after it are Titillium Web and JetBrains Mono.
func TestTheTwoFontChoosers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 860)
	for _, name := range []string{"Text font", "Code font"} {
		cb := namedCombo(w.Content(), name)
		if cb == nil {
			t.Fatalf("no %q chooser on the page", name)
		}
		if insidePreview(cb) {
			t.Errorf("the %q chooser is inside the preview", name)
		}
		if len(cb.Items) < 3 {
			t.Fatalf("the %q chooser offers %v", name, cb.Items)
		}
		if cb.Items[0] != themeFontItem {
			t.Errorf("the %q chooser leads with %q, want the way back to the pack's own", name, cb.Items[0])
		}
		if cb.Items[1] != style.FamilyUI || cb.Items[2] != style.FamilyMono {
			t.Errorf("the %q chooser's first two typefaces are %q and %q, want the two the toolkit carries (%q, %q)",
				name, cb.Items[1], cb.Items[2], style.FamilyUI, style.FamilyMono)
		}
		if cb.Selected != 0 {
			t.Errorf("the %q chooser starts at %q; nothing chosen is the default", name, cb.Items[cb.Selected])
		}
		// Every family it lists can actually be drawn with.
		for _, fam := range cb.Items[1:] {
			if !style.FontInstalled(fam) {
				t.Errorf("the %q chooser offers %q, which is not on this machine", name, fam)
			}
		}
	}
	// Both are handed the same list: the difference between them is
	// which role they set, not which families they believe are suitable
	// for it.
	ui, mono := namedCombo(w.Content(), "Text font"), namedCombo(w.Content(), "Code font")
	if strings.Join(ui.Items, "|") != strings.Join(mono.Items, "|") {
		t.Error("the two choosers offer different families; neither role forbids a family to the other")
	}
}

// Choosing a family stages it, and it reaches the previewed window,
// which is the whole of what a picker is for.
func TestChoosingAFontStagesIt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)
	ui := namedCombo(w.Content(), "Text font")
	mono := namedCombo(w.Content(), "Code font")
	// The bundled monospaced face for the interface and the bundled UI
	// face for code: a silly pair, installed everywhere, and visibly
	// each other's opposite.
	pickItem(t, ui, style.FamilyMono)
	pickItem(t, mono, style.FamilyUI)
	a.PumpOnce()
	look := previewLook(t, w)
	if look.UIFamily() != style.FamilyMono {
		t.Errorf("the previewed window reads in %q, want the chosen %q", look.UIFamily(), style.FamilyMono)
	}
	if look.MonoFamily() != style.FamilyUI {
		t.Errorf("the previewed window's code face is %q, want the chosen %q", look.MonoFamily(), style.FamilyUI)
	}
	// Apply writes both to look.json, and the way back writes neither.
	clickApply(t, w)
	saved := style.LoadAppearance()
	if saved.FontUI != style.FamilyMono || saved.FontMono != style.FamilyUI {
		t.Errorf("look.json says %q / %q", saved.FontUI, saved.FontMono)
	}
	ui = namedCombo(w.Content(), "Text font")
	pickItem(t, ui, themeFontItem)
	a.PumpOnce()
	clickApply(t, w)
	if got := style.LoadAppearance().FontUI; got != "" {
		t.Errorf("%q left %q in look.json", themeFontItem, got)
	}
}

// The one thing a person cannot see by looking at the control: which of
// the two — the pack's era typeface or the family they picked — is what
// the window is actually reading in. Both choosers say it, and both name
// the family that is really being drawn with.
func TestTheFontChoosersSayWhoWins(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)
	ui := namedCombo(w.Content(), "Text font")
	if ui.Tip == "" {
		t.Fatal("the interface typeface chooser says nothing about itself")
	}
	// Nothing chosen: it names the pack and the face the pack came down
	// to on this machine.
	if !strings.Contains(ui.Tip, themeFontItem) {
		t.Errorf("with nothing chosen the tip does not mention %q: %q", themeFontItem, ui.Tip)
	}
	if drawn := previewLook(t, w).UIFamily(); !strings.Contains(ui.Tip, drawn) {
		t.Errorf("the tip does not name the face being drawn with (%q): %q", drawn, ui.Tip)
	}
	// Something chosen: it says, in as many words, that the choice beats
	// the pack.
	pickItem(t, ui, style.FamilyMono)
	a.PumpOnce()
	ui = namedCombo(w.Content(), "Text font")
	if !strings.Contains(ui.Tip, "beats the pack") {
		t.Errorf("the tip does not say which of the two wins: %q", ui.Tip)
	}
	if !strings.Contains(ui.Tip, style.FamilyMono) {
		t.Errorf("the tip does not name the chosen face: %q", ui.Tip)
	}
	if ui.AccessibleDescription() != ui.Tip {
		t.Error("what is hovered and what is read out are two different sentences")
	}
}

// The icon chooser lists the desktop's own themes after the toolkit's,
// and says what it left out.
func TestTheIconChooserListsInstalledThemes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 860)
	cb := namedCombo(w.Content(), "Icons")
	if cb == nil {
		t.Fatal("no icon chooser")
	}
	if cb.Items[0] != "Classic" || cb.Items[1] != "Sharp" {
		t.Errorf("the chooser leads with %v, want the two drawn sets", cb.Items[:2])
	}
	system := style.ListSystemIconThemes()
	if len(system) > 0 {
		// The desktop's themes come last, in the order the style
		// package lists them, and every one of them is one this
		// toolkit can actually draw.
		tail := cb.Items[len(cb.Items)-len(system):]
		for i, s := range system {
			if tail[i] != s.Label {
				t.Errorf("item %d of the desktop's themes is %q, want %q", i, tail[i], s.Label)
			}
			if !style.SystemIconThemeHas(s.Name, style.IconSave) {
				t.Errorf("%q is listed and cannot draw a Save icon", s.Label)
			}
		}
	}
	if !strings.Contains(cb.Tip, "icon themes this desktop has installed") {
		t.Errorf("the chooser does not say where the last group comes from: %q", cb.Tip)
	}
	for _, p := range style.UnavailableSystemIconThemes() {
		for _, it := range cb.Items {
			if it == p.Name {
				t.Errorf("%q cannot be drawn (%s) and is in the chooser", p.Name, p.Reason)
			}
		}
	}
	if len(style.UnavailableSystemIconThemes()) > 0 && !strings.Contains(cb.Tip, "Not listed") {
		t.Errorf("themes were left out and the chooser does not say which: %q", cb.Tip)
	}
}

// pickItem chooses a combo box's item by its text.
func pickItem(t *testing.T, cb *widgets.ComboBox, item string) {
	t.Helper()
	if cb == nil || cb.OnChange == nil {
		t.Fatalf("no combo to pick %q from", item)
	}
	for i, it := range cb.Items {
		if it == item {
			cb.Selected = i
			cb.OnChange(i)
			return
		}
	}
	t.Fatalf("no item %q among %d", item, len(cb.Items))
}

// previewLook is the look the previewed window is drawn in.
func previewLook(t *testing.T, w *app.Window) *style.Classic {
	t.Helper()
	var scope *widgets.ThemeScope
	widget.Walk(w.Content(), func(c widget.Component) {
		if s, ok := c.(*widgets.ThemeScope); ok && scope == nil {
			scope = s
		}
	})
	if scope == nil {
		t.Fatal("no theme preview")
	}
	c, ok := scope.Theme().(*style.Classic)
	if !ok {
		t.Fatal("the preview is not drawn in a Classic look")
	}
	return c
}
