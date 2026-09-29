package settingsapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// labelSaying finds a label whose text contains want.
func labelSaying(root widget.Component, want string) *widgets.Label {
	var found *widgets.Label
	widget.Walk(root, func(c widget.Component) {
		if l, ok := c.(*widgets.Label); ok && found == nil && strings.Contains(l.Text, want) {
			found = l
		}
	})
	return found
}

// A look.json naming a pack this build cannot paint leaves no row in the
// browser selected and a preview the user did not choose, which is what
// "the theme is wrong" looks like. Settings says which pack is missing
// and which it is showing instead.
func TestSettingsSaysWhenTheSavedThemeIsMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "uitoolkit"), 0o755); err != nil {
		t.Fatal(err)
	}
	const pack = "no-such-pack-in-any-build"
	body := `{"version":2,"theme":"` + pack + `"}`
	if err := os.WriteFile(filepath.Join(dir, "uitoolkit", "look.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 780)
	if l := labelSaying(w.Content(), pack); l == nil {
		t.Fatalf("the browser does not name the missing pack %q", pack)
	} else if !strings.Contains(l.Text, style.DefaultTheme()) {
		t.Fatalf("the note does not say what is being shown: %q", l.Text)
	}
}

// And says nothing in the ordinary case: the column is three controls of
// fixed height and a list that takes what is left.
func TestSettingsSaysNothingWhenTheThemeIsThere(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.Appearance{Name: style.DefaultTheme()}.Normalize()); err != nil {
		t.Fatal(err)
	}
	_, w := openSettings(t, 1024, 780)
	if l := labelSaying(w.Content(), "is not in this build"); l != nil {
		t.Fatalf("an unasked-for note: %q", l.Text)
	}
}
