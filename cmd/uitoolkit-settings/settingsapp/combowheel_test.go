package settingsapp

import (
	"os"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The fifth box. It writes look.json like the other four, and until
// Apply it writes nothing at all.
func TestSettingsComboWheelApplyWritesLookJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	was := style.ComboWheel()
	t.Cleanup(func() { style.SetComboWheel(was) })

	a, w := openSettings(t, 1024, 860)
	box := findOption(w.Content(), "Combo wheel")
	if box == nil {
		t.Fatal("no Combo wheel option over the preview")
	}
	// Off is the default, in the file and on the box.
	if style.DefaultAppearance().ComboWheel {
		t.Error("the default appearance turns the combo wheel on")
	}
	if box.Checked {
		t.Error("the box is ticked with the preference off")
	}

	box.OnChange(true)
	a.PumpOnce()
	if style.LoadAppearance().ComboWheel {
		t.Fatal("ticking the box wrote look.json; only Apply writes")
	}
	clickApply(t, w)
	a.PumpOnce()

	if !style.LoadAppearance().ComboWheel {
		t.Error("Apply did not save the combo wheel preference")
	}
	raw, err := os.ReadFile(style.AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"comboWheel": true`) {
		t.Errorf("look.json: %s", raw)
	}
	// And Apply applies it to this process, the way it applies reduce
	// motion and the desktop's dialogs.
	if !style.ComboWheel() {
		t.Error("Apply did not put the preference into the process")
	}

	// Off again, and the key goes out of the file rather than being
	// written false — the same rule every other default-off preference
	// here follows.
	box2 := findOption(w.Content(), "Combo wheel")
	box2.OnChange(false)
	clickApply(t, w)
	a.PumpOnce()
	raw, _ = os.ReadFile(style.AppearancePath())
	if strings.Contains(string(raw), "comboWheel") {
		t.Errorf("an off preference is still in look.json: %s", raw)
	}
	if style.ComboWheel() {
		t.Error("Apply did not take the preference back out of the process")
	}
}

// The preference reaches the combo boxes on Settings' own page, which
// is the shortest possible proof that it reaches an application.
func TestSettingsComboWheelReachesAComboBox(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	was := style.ComboWheel()
	t.Cleanup(func() { style.SetComboWheel(was) })

	a, w := openSettings(t, 1024, 860)
	corners := namedCombo(w.Content(), "Window corners")
	if corners == nil {
		t.Fatal("no corners chooser")
	}
	corners.Select(0)
	down := widget.MouseEvent{Scroll: paintengine2d.Pt(0, 1)}

	style.SetComboWheel(false)
	if corners.MouseWheel(down) || corners.Selected != 0 {
		t.Error("a chooser stepped on the wheel with the preference off")
	}
	findOption(w.Content(), "Combo wheel").OnChange(true)
	clickApply(t, w)
	a.PumpOnce()
	corners = namedCombo(w.Content(), "Window corners")
	corners.Select(0)
	if !corners.MouseWheel(down) || corners.Selected != 1 {
		t.Errorf("a chooser did not step with the preference on (selected %d)", corners.Selected)
	}
}

// The theme search field is the one field in Settings with a clear
// button, and clearing it is the whole list again.
func TestSettingsSearchFieldClears(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	a, w := openSettings(t, 1024, 860)
	search := searchField(t, w)
	if !search.Clearable {
		t.Fatal("the theme search field has no clear button")
	}
	list := findThemeListAny(w.Content())
	all := list.Count
	if all < 100 {
		t.Fatalf("the browser lists %d packs", all)
	}

	search.SetText("win")
	a.PumpOnce()
	if list.Count >= all || list.Count == 0 {
		t.Fatalf("searching for %q left %d of %d packs", "win", list.Count, all)
	}
	// A screen reader clears it by name, and the whole list comes back.
	if n := search.AccessibleName(); n != "Search themes" {
		t.Errorf("the field is called %q", n)
	}
	if !search.AccessibleAction(0, a11y.ActionDefault) {
		t.Fatal("the clear button did not answer its default action")
	}
	a.PumpOnce()
	if search.Text != "" {
		t.Errorf("the field still reads %q", search.Text)
	}
	if list.Count != all {
		t.Errorf("clearing the search left %d of %d packs", list.Count, all)
	}

	// No other field in Settings has one: the export prompt's field is
	// answered once and the dialog closes, and the preview's sample is
	// a picture of a field.
	widget.Walk(w.Content(), func(c widget.Component) {
		f, ok := c.(*widgets.TextField)
		if ok && f != search && f.Clearable {
			t.Errorf("an unexpected field is clearable: %q / %q", f.Placeholder, f.AccessibleName())
		}
	})
}

func searchField(t *testing.T, w *app.Window) *widgets.TextField {
	t.Helper()
	var f *widgets.TextField
	widget.Walk(w.Content(), func(c widget.Component) {
		if tf, ok := c.(*widgets.TextField); ok && tf.AccessibleName() == "Search themes" {
			f = tf
		}
	})
	if f == nil {
		t.Fatal("no search field in the browser column")
	}
	return f
}
