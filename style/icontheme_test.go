package style

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// fixtureThemes builds a search path of its own and puts three themes in
// it: one of PNGs, one of the monochrome SVGs Breeze is made of, and one
// that inherits from the first. The machine's own /usr/share/icons is
// out of reach for the rest of the test, which is the point — a test
// that passed because the box it ran on had Breeze installed would say
// nothing at all.
func fixtureThemes(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv(IconThemeDirsEnv, root)
	resetIconCache()
	t.Cleanup(resetIconCache)

	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	png := func(path string, size int, col paintengine2d.Color) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		img := paintengine2d.NewImage(size, size)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(2, 2, float32(size-4), float32(size-4)), paintengine2d.Fill(col))
		img.Touch()
		if err := img.WritePNGFile(path); err != nil {
			t.Fatal(err)
		}
	}

	// pngtheme: 22- and 32-pixel directories of flat grey squares.
	write(filepath.Join(root, "pngtheme", "index.theme"), `[Icon Theme]
Name=PNG Theme
Directories=actions/22,actions/32

[actions/22]
Size=22
Type=Fixed
Context=Actions

[actions/32]
Size=32
Type=Fixed
Context=Actions
`)
	for _, name := range []string{"document-save", "edit-copy", "document-open"} {
		png(filepath.Join(root, "pngtheme", "actions", "22", name+".png"), 22, paintengine2d.RGB(0.2, 0.2, 0.2))
		png(filepath.Join(root, "pngtheme", "actions", "32", name+".png"), 32, paintengine2d.RGB(0.2, 0.2, 0.2))
	}

	// svgtheme: one scalable directory of currentColor paths.
	write(filepath.Join(root, "svgtheme", "index.theme"), `[Icon Theme]
Name=SVG Theme
Inherits=pngtheme
Directories=scalable/actions

[scalable/actions]
Size=22
MinSize=8
MaxSize=512
Type=Scalable
Context=Actions
`)
	for _, name := range []string{"document-save", "edit-copy"} {
		write(filepath.Join(root, "svgtheme", "scalable", "actions", name+".svg"),
			`<svg viewBox="0 0 22 22"><path style="fill:currentColor" d="M 3 3 L 19 3 L 19 19 L 3 19 Z"/></svg>`)
	}

	// cursors: an index.theme with no icon directories at all, which is
	// what every *_cursors folder on a Linux box is. It is not an icon
	// theme and is not mentioned at all — a chooser that listed the
	// reason every cursor theme on the machine is unavailable would be
	// reading out a list nobody asked about.
	write(filepath.Join(root, "cursors", "index.theme"), "[Icon Theme]\nName=Cursors\n")
	write(filepath.Join(root, "cursors", "cursors", "left_ptr"), "not an icon")

	// empty: a theme with directories and nothing the toolkit asks for
	// in them, which is what hicolor is on most machines.
	write(filepath.Join(root, "empty", "index.theme"), `[Icon Theme]
Name=Empty
Directories=apps/48

[apps/48]
Size=48
Type=Fixed
`)
	png(filepath.Join(root, "empty", "apps", "48", "firefox.png"), 48, paintengine2d.RGB(0.9, 0.3, 0.1))

	// nodraw: an index.theme that promises a directory of files this
	// toolkit cannot read.
	write(filepath.Join(root, "nodraw", "index.theme"), `[Icon Theme]
Name=No Draw
Directories=actions/22

[actions/22]
Size=22
Type=Fixed
`)
	write(filepath.Join(root, "nodraw", "actions", "22", "document-save.svg"),
		`<svg viewBox="0 0 22 22"><use href="#elsewhere"/></svg>`)
	write(filepath.Join(root, "nodraw", "actions", "22", "edit-copy.svg"),
		`<svg viewBox="0 0 22 22"><use href="#elsewhere"/></svg>`)
	return root
}

func TestSystemIconThemesListWhatCanBeDrawn(t *testing.T) {
	fixtureThemes(t)
	listed := ListSystemIconThemes()
	var names []string
	for _, s := range listed {
		names = append(names, string(s.Name)+"="+s.Label)
		if s.Source != ThemeSourceDesktop {
			t.Errorf("%s is sourced %q", s.Name, s.Source)
		}
	}
	want := []string{"desktop:pngtheme=PNG Theme", "desktop:svgtheme=SVG Theme"}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("listed %v, want %v", names, want)
	}
	// And the two that cannot be drawn are named, with the reason,
	// rather than being silently absent.
	bad := UnavailableSystemIconThemes()
	if len(bad) != 2 {
		t.Fatalf("unavailable: %+v", bad)
	}
	byDir := map[string]string{}
	for _, p := range bad {
		byDir[p.Dir] = p.Reason
	}
	if _, listed := byDir["cursors"]; listed {
		t.Error("a folder of cursors is not an icon theme and is not a problem to report")
	}
	if !strings.Contains(byDir["empty"], "no icons") {
		t.Errorf("a theme with none of the toolkit's actions says %q", byDir["empty"])
	}
	if !strings.Contains(byDir["nodraw"], "cannot draw") || !strings.Contains(byDir["nodraw"], "use") {
		t.Errorf("a theme of files we cannot read says %q; it should name the construct", byDir["nodraw"])
	}
}

// The toolkit's actions are looked up under the freedesktop names, not
// under its own: a theme has "document-save", never "save".
func TestSystemIconThemeResolvesFreedesktopNames(t *testing.T) {
	// As above: an installed freedesktop icon theme, found the XDG way.
	if runtime.GOOS != "linux" {
		t.Skip("freedesktop icon themes are a Linux desktop's")
	}
	root := fixtureThemes(t)
	set := SystemIconSetName("pngtheme")
	if !SystemIconThemeHas(set, IconSave) {
		t.Error("the theme has document-save and the toolkit did not find it")
	}
	if ToolIconThemeName(IconSave) != "document-save" {
		t.Errorf("Save is asked for as %q", ToolIconThemeName(IconSave))
	}
	if ToolIconThemeName(IconCut) != "edit-cut" {
		t.Errorf("Cut is asked for as %q", ToolIconThemeName(IconCut))
	}
	// The wanted size wins over the nearest one.
	chain := themeChain("pngtheme")
	if p := findThemeIcon(chain, []string{"document-save"}, 32, 1); !strings.Contains(p, "/32/") {
		t.Errorf("asked for 32 and got %q", p)
	}
	if p := findThemeIcon(chain, []string{"document-save"}, 22, 1); !strings.Contains(p, "/22/") {
		t.Errorf("asked for 22 and got %q", p)
	}
	// A size nobody ships falls back to the nearest directory.
	if p := findThemeIcon(chain, []string{"document-save"}, 24, 1); p == "" {
		t.Error("24 found nothing, with 22 and 32 on the disk")
	}
	_ = root
}

// Inherits is walked, and a name the theme itself has not got comes from
// the theme it inherits.
func TestSystemIconThemeInherits(t *testing.T) {
	fixtureThemes(t)
	chain := themeChain("svgtheme")
	if len(chain) < 2 || chain[0].dir != "svgtheme" || chain[1].dir != "pngtheme" {
		t.Fatalf("chain is %v", chain)
	}
	// svgtheme has document-save of its own…
	if p := findThemeIcon(chain, []string{"document-save"}, 22, 1); !strings.Contains(p, "svgtheme") {
		t.Errorf("its own document-save came from %q", p)
	}
	// …and none of its own document-open, which the parent has.
	if p := findThemeIcon(chain, []string{"document-open"}, 22, 1); !strings.Contains(p, "pngtheme") {
		t.Errorf("document-open came from %q, want the inherited theme", p)
	}
}

// A monochrome file is a shape the look tints; a coloured one is a
// picture drawn as its author drew it. It is the rule the desktops
// follow, and the only one under which a symbolic outline and a
// full-colour floppy disk can both come out right.
func TestSystemIconThemeTintsOnlyMonochrome(t *testing.T) {
	root := fixtureThemes(t)
	mono, err := loadThemeIconFile(filepath.Join(root, "svgtheme", "scalable", "actions", "document-save.svg"), 22)
	if err != nil {
		t.Fatal(err)
	}
	if !mono.tint {
		t.Error("a currentColor path is a shape and must follow the look's foreground")
	}
	grey, err := loadThemeIconFile(filepath.Join(root, "pngtheme", "actions", "22", "document-save.png"), 22)
	if err != nil {
		t.Fatal(err)
	}
	if !grey.tint {
		t.Error("a one-colour PNG is a shape too")
	}
	// Two colours in one file is a picture.
	multi := paintengine2d.NewImage(8, 8)
	ctx := paintengine2d.NewContext(multi)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, 8, 4), paintengine2d.Fill(paintengine2d.RGB(1, 0, 0)))
	ctx.DrawRect(paintengine2d.XYWH(0, 4, 8, 4), paintengine2d.Fill(paintengine2d.RGB(0, 0, 1)))
	multi.Touch()
	if iconImageIsMonochrome(multi) {
		t.Error("a red half and a blue half is not one colour")
	}
	// And a wholly transparent file is not a shape, it is nothing.
	if iconImageIsMonochrome(paintengine2d.NewImage(8, 8)) {
		t.Error("an empty pixmap is not monochrome")
	}
}

// The chooser draws what it listed: every listed theme paints something
// for every one of the toolkit's actions, falling back to the drawn set
// for the names a theme has never heard of rather than to a placeholder.
func TestSystemIconSetsPaint(t *testing.T) {
	fixtureThemes(t)
	for _, set := range ListSystemIconThemes() {
		for _, icon := range AllToolIcons() {
			img := paintengine2d.NewImage(24, 24)
			ctx := paintengine2d.NewContext(img)
			DrawToolIcon(ctx, paintengine2d.XYWH(0, 0, 24, 24), icon, paintengine2d.Black, set.Name)
			img.Touch()
			painted := false
			for y := 0; y < 24 && !painted; y++ {
				for x := 0; x < 24; x++ {
					if _, _, _, a := img.PremulAt(x, y); a > 8 {
						painted = true
						break
					}
				}
			}
			if !painted {
				t.Errorf("%s drew nothing for %s", set.Name, icon.Label())
			}
		}
	}
}

// The id is a namespace of its own and a name in look.json cannot reach
// out of the icon directories it is joined to.
func TestSystemIconSetNamesAreSafe(t *testing.T) {
	for _, bad := range []string{
		"desktop:../../etc",
		"desktop:a/b",
		"desktop:",
		"desktop:.",
		"desktop:..",
		"desktop:x/../../y",
	} {
		if got := ParseIconSet(bad); got != IconSetClassic {
			t.Errorf("%q parsed as %q, want the drawn set", bad, got)
		}
	}
	set := ParseIconSet("desktop:Breeze_Light")
	if set != IconSetName("desktop:Breeze_Light") {
		t.Errorf("a theme directory keeps its own spelling: %q", set)
	}
	if !IsSystemIconSet(set) || IsFileIconSet(set) {
		t.Errorf("%q is a desktop theme, not a folder under icons/", set)
	}
	if dir, ok := SystemIconTheme(set); !ok || dir != "Breeze_Light" {
		t.Errorf("theme directory %q %v", dir, ok)
	}
	// And no folder under icons/ can ever produce one: the colon is
	// rejected by the sanitizer.
	if _, err := SanitizeIconSetName("desktop:breeze"); err == nil {
		t.Error("a user folder named with a colon must not sanitize")
	}
}

// look.json carries the choice, and a theme that is not installed on the
// machine that reads the file survives as a name rather than being
// rewritten to something else.
func TestSystemIconSetSurvivesLookJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := DefaultAppearance()
	a.Icons = SystemIconSetName("breeze-dark")
	if err := SaveAppearance(a); err != nil {
		t.Fatal(err)
	}
	if got := LoadAppearance().Icons; got != a.Icons {
		t.Errorf("look.json gave back %q, want %q", got, a.Icons)
	}
	raw, err := os.ReadFile(AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"icons": "desktop:breeze-dark"`) {
		t.Errorf("look.json says %s", raw)
	}
}

// The search path is the freedesktop one, and the variable that replaces
// it can switch installed themes off entirely — which is what keeps a
// screenshot of this page the same picture on two machines.
func TestIconThemeSearchPath(t *testing.T) {
	// The XDG icon theme specification is a Linux desktop's, and so are
	// the paths this asserts: elsewhere filepath.Join answers with
	// backslashes and filepath.SplitList splits XDG_DATA_DIRS on the
	// wrong character, so the test is about nothing. Skipped rather
	// than given a build tag because the rest of this file is portable
	// and worth running wherever the tests run (tools/test-windows.sh).
	if runtime.GOOS != "linux" {
		t.Skip("the XDG icon theme search path is a Linux desktop's")
	}
	t.Setenv("XDG_DATA_HOME", "/xdg/data")
	t.Setenv("XDG_DATA_DIRS", "/a:/b")
	os.Unsetenv(IconThemeDirsEnv)
	dirs := iconThemeSearchDirs()
	want := []string{"/xdg/data/icons", "/a/icons", "/b/icons"}
	for _, w := range want {
		found := false
		for _, d := range dirs {
			if d == w {
				found = true
			}
		}
		if !found {
			t.Errorf("%q is not on the search path %v", w, dirs)
		}
	}
	if dirs[0] != "/xdg/data/icons" {
		t.Errorf("the user's own directory is not first: %v", dirs)
	}
	t.Setenv(IconThemeDirsEnv, "none")
	if d := iconThemeSearchDirs(); len(d) != 0 {
		t.Errorf("%s=none left %v", IconThemeDirsEnv, d)
	}
	resetIconCache()
	if got := ListSystemIconThemes(); len(got) != 0 {
		t.Errorf("%s=none listed %v", IconThemeDirsEnv, got)
	}
}
