package dock

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/codemodify/uitoolkit/widget"
)

// The arrangement goes to a file named for the app under the config
// directory, and comes back where it was.
func TestLayoutFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := filepath.Join(dir, "uitoolkit", "inspector-dock.json")
	if got := LayoutFile("inspector"); got != want {
		t.Fatalf("LayoutFile = %s, want %s", got, want)
	}

	r := newRig(t)
	r.host.Dock(r.log, SideTop)
	if err := r.host.SaveLayoutFile("inspector"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("nothing at %s: %v", want, err)
	}

	// A second host of the same shape comes up in the saved arrangement.
	r2 := newRig(t)
	if got := sideOf(t, r2.host, r2.log); got != SideBottom {
		t.Fatalf("the fresh host starts with the log at %v", got)
	}
	ok, err := r2.host.LoadLayoutFile("inspector")
	if err != nil || !ok {
		t.Fatalf("LoadLayoutFile = %v, %v", ok, err)
	}
	if got := sideOf(t, r2.host, r2.log); got != SideTop {
		t.Errorf("after loading, the log is at %v, wanted top", got)
	}
}

// A first run has no file: nothing is applied, the host is left alone,
// and the error says why so an app can tell a missing layout from a
// broken one.
func TestLoadLayoutFileFirstRun(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	r := newRig(t)
	ok, err := r.host.LoadLayoutFile("nobody")
	if ok {
		t.Error("it claimed to have applied a layout that is not there")
	}
	if !isNotExist(err) {
		t.Errorf("error %v, want one that says the file is not there", err)
	}
	if got := sideOf(t, r.host, r.log); got != SideBottom {
		t.Errorf("the host moved anyway: the log is at %v", got)
	}
}

// A file that is not a layout leaves the host in its default arrangement
// rather than half-applied.
func TestLoadLayoutFileBroken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "uitoolkit"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LayoutFile("broken"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := newRig(t)
	ok, err := r.host.LoadLayoutFile("broken")
	if ok || err == nil {
		t.Fatalf("LoadLayoutFile = %v, %v; want a refusal", ok, err)
	}
	if isNotExist(err) {
		t.Error("a broken file must not look like a missing one")
	}
	for _, p := range []*Panel{r.tree, r.prop, r.log} {
		if p.Stack() == nil {
			t.Errorf("%s was left out of the tree", p.Name())
		}
	}
}

// A layout name is an app's own name, never a path: nothing may write
// outside the toolkit's config directory, least of all over look.json.
func TestLayoutNameIsNotAPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	r := newRig(t)
	for _, name := range []string{"", ".", "..", "../look", "a/b", `a\b`, ".hidden"} {
		if ValidLayoutName(name) {
			t.Errorf("%q was accepted as a layout name", name)
		}
		if err := r.host.SaveLayoutFile(name); err == nil {
			t.Errorf("SaveLayoutFile(%q) wrote something", name)
		}
		if _, err := r.host.LoadLayoutFile(name); err == nil {
			t.Errorf("LoadLayoutFile(%q) read something", name)
		}
	}
	if !ValidLayoutName("inspector") {
		t.Error("a plain app name was refused")
	}
}

// sideOf is the area a panel is in, found the way an application would:
// by looking through the four areas for it.
func sideOf(t *testing.T, h *Host, p *Panel) Side {
	t.Helper()
	for _, s := range []Side{SideLeft, SideRight, SideTop, SideBottom} {
		found := false
		widget.Walk(h.Area(s), func(c widget.Component) {
			if c == widget.Component(p) {
				found = true
			}
		})
		if found {
			return s
		}
	}
	t.Fatalf("%s is in no area", p.Name())
	return SideLeft
}

func isNotExist(err error) bool {
	pe, ok := err.(*fs.PathError)
	return ok && os.IsNotExist(pe)
}
