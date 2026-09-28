package style

import (
	"os"
	"strings"
	"testing"
)

// The reason a pack that asks for a face nobody has does not quietly
// draw in something else.
//
// fc-match is a *matcher*: it is required to answer, and on a machine
// with the usual fontconfig it answers "Noto Sans" for every family
// asked for, installed or not — including the bundled Titillium Web. A
// lookup built on it would tell every one of the 131 packs that its
// era's typeface was present, and all 131 would read in one face while
// believing they had found Lucida, Tahoma and Chicago.
//
// The lookup is built on fc-list instead: it enumerates what is on the
// disk, keyed by the family each file declares, and the comparison is
// this package's. This pins that, in the only way that survives a
// refactor — by handing the index a machine that has three families and
// asking for the ones it has not got.
func TestLookupNeverAsksFontconfigToMatch(t *testing.T) {
	fakeFonts(t,
		"Noto Sans\t80\t0\t0\t/f/NotoSans-Regular.ttf",
		"Noto Sans\t200\t0\t0\t/f/NotoSans-Bold.ttf",
		"Liberation Sans\t80\t0\t0\t/f/LiberationSans-Regular.ttf",
		"Liberation Mono\t80\t0\t0\t/f/LiberationMono-Regular.ttf",
	)
	for _, absent := range []string{"Lucida Grande", "Lucida Sans", "Tahoma", "Chicago", "Segoe UI", "Helvetica"} {
		if FontInstalled(absent) {
			t.Errorf("%q is not on this machine and the index says it is", absent)
		}
		if _, _, ok := pickSystemFace(absent, WeightRegular); ok {
			t.Errorf("%q resolved to a face: a matcher answered where an index should have said no", absent)
		}
	}
	// And a pack's list falls through to the look-alike that *is* here,
	// which is the whole mechanism: Aqua's Lucidas are absent, its
	// DejaVu is absent, and its Noto Sans is not.
	if got := ResolveFont(lucida); got != "Noto Sans" {
		t.Errorf("Aqua's typefaces came down to %q, want the first one installed", got)
	}
	// The bundled faces resolve without fontconfig having heard of them.
	if got := ResolveFont([]string{"Titillium Web"}); got != FamilyUI {
		t.Errorf("the bundled UI face resolved to %q", got)
	}
	if got := ResolveFont([]string{"Nothing At All"}); got != "" {
		t.Errorf("a family nobody has resolved to %q; the caller keeps the bundled face", got)
	}
}

// What the two choosers list: the bundled faces first, then what is
// installed, each family once.
func TestListFontFamilies(t *testing.T) {
	fakeFonts(t,
		"Zapf Dingbats\t80\t0\t0\t/f/zapf.ttf",
		"Noto Sans,Noto Sans Display\t80\t0\t0\t/f/NotoSans-Regular.ttf",
		"Adwaita Sans\t80\t0\t0\t/f/AdwaitaSans.ttf",
		"Adwaita Sans\t200\t0\t0\t/f/AdwaitaSans-Bold.ttf",
		"Titillium Web\t80\t0\t0\t/f/elsewhere.ttf",
	)
	got := ListFontFamilies()
	if len(got) < 4 || got[0] != FamilyUI || got[1] != FamilyMono {
		t.Fatalf("the list must lead with what is embedded: %v", got)
	}
	rest := got[2:]
	want := []string{"Adwaita Sans", "Noto Sans", "Zapf Dingbats"}
	if strings.Join(rest, "|") != strings.Join(want, "|") {
		t.Errorf("installed families are %v, want %v sorted", rest, want)
	}
	// "Noto Sans Display" is the same file's second name, not a typeface
	// of its own, and does not stand in the list as if it were — while
	// staying perfectly resolvable.
	if !FontInstalled("Noto Sans Display") {
		t.Error("an alias must still resolve")
	}
	// A family the toolkit already carries is not offered twice.
	seen := map[string]int{}
	for _, f := range got {
		seen[strings.ToLower(f)]++
	}
	if seen[strings.ToLower(FamilyUI)] != 1 {
		t.Errorf("%s is in the list %d times", FamilyUI, seen[strings.ToLower(FamilyUI)])
	}
}

func TestSystemFontsOffLeavesTheBundledTwo(t *testing.T) {
	fakeFonts(t, "Noto Sans\t80\t0\t0\t/f/NotoSans-Regular.ttf")
	t.Setenv(SystemFontsEnv, "0") // after fakeFonts: it is what is under test
	if got := ListFontFamilies(); len(got) != 2 || got[0] != FamilyUI || got[1] != FamilyMono {
		t.Errorf("%s=0 lists %v", SystemFontsEnv, got)
	}
}

// A family chosen by the user beats the pack's era, and the pack's era
// is still the fallback when that family is not on this machine.
func TestChosenFontBeatsThePack(t *testing.T) {
	fakeFonts(t,
		"Noto Sans\t80\t0\t0\t/f/NotoSans-Regular.ttf",
		"Liberation Sans\t80\t0\t0\t/f/LiberationSans-Regular.ttf",
		"Liberation Mono\t80\t0\t0\t/f/LiberationMono-Regular.ttf",
	)
	// aqua asks for Lucida Grande and comes down to Noto Sans here.
	needEngine(t, "aqua")
	base := Appearance{Name: "aqua"}.Normalize()
	look := base.Look()
	if look.UIFamily() != "Noto Sans" {
		t.Fatalf("the pack's own typeface resolved to %q", look.UIFamily())
	}
	chosen := base
	chosen.FontUI = "Liberation Sans"
	chosen.FontMono = "Liberation Mono"
	if got := chosen.Look().UIFamily(); got != "Liberation Sans" {
		t.Errorf("the chosen interface face lost to the pack: %q", got)
	}
	if got := chosen.Look().MonoFamily(); got != "Liberation Mono" {
		t.Errorf("the chosen code face lost to the pack: %q", got)
	}
	// A family that is not installed here — a look.json carried over
	// from another machine — leaves the pack's era underneath rather
	// than a hole.
	gone := base
	gone.FontUI = "Lucida Grande"
	if got := gone.Look().UIFamily(); got != "Noto Sans" {
		t.Errorf("an uninstalled choice left %q; the pack's list is still underneath", got)
	}
	// And the way back is the empty name, which is what the chooser's
	// first item sets.
	back := chosen
	back.FontUI, back.FontMono = "", ""
	if got := back.Look().UIFamily(); got != "Noto Sans" {
		t.Errorf("clearing the choice left %q", got)
	}
}

func TestFontChoiceSurvivesLookJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := DefaultAppearance()
	a.FontUI = "Liberation Sans"
	a.FontMono = "Liberation Mono"
	if err := SaveAppearance(a); err != nil {
		t.Fatal(err)
	}
	got := LoadAppearance()
	if got.FontUI != a.FontUI || got.FontMono != a.FontMono {
		t.Errorf("look.json gave back %q / %q", got.FontUI, got.FontMono)
	}
	raw, err := os.ReadFile(AppearancePath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"fontUI": "Liberation Sans"`) {
		t.Errorf("look.json says %s", raw)
	}
	// The default is no choice at all, and it is left out of the file
	// the way every other default is.
	if err := SaveAppearance(DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(AppearancePath())
	if strings.Contains(string(raw), "fontUI") {
		t.Errorf("the pack's own typefaces are written out: %s", raw)
	}
}

// UITK_FONT overrides the saved typeface for one process, the way
// UITK_THEME overrides the saved pack; "theme" puts a role back on the
// pack's era without the file being touched.
func TestFontEnvOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := DefaultAppearance()
	a.FontUI = "Liberation Sans"
	if err := SaveAppearance(a); err != nil {
		t.Fatal(err)
	}
	t.Setenv(FontEnv, "Noto Sans")
	t.Setenv(MonoFontEnv, "Hack")
	got := LoadAppearance()
	if got.FontUI != "Noto Sans" || got.FontMono != "Hack" {
		t.Errorf("the environment did not override: %q / %q", got.FontUI, got.FontMono)
	}
	t.Setenv(FontEnv, "theme")
	if got := LoadAppearance().FontUI; got != "" {
		t.Errorf("%s=theme left %q", FontEnv, got)
	}
	// It is not written back on its own: the file still says what it
	// said until something saves.
	os.Unsetenv(FontEnv)
	os.Unsetenv(MonoFontEnv)
	if got := LoadAppearance().FontUI; got != "Liberation Sans" {
		t.Errorf("the file says %q; the environment must not have rewritten it", got)
	}
}

func TestNormalizeFontChoice(t *testing.T) {
	for in, want := range map[string]string{
		"":                "",
		"   ":             "",
		"theme":           "",
		"Theme font":      "",
		"default":         "",
		"  Noto Sans   ":  "Noto Sans",
		"JetBrains Mono ": "JetBrains Mono",
	} {
		if got := NormalizeFontChoice(in); got != want {
			t.Errorf("%q normalized to %q, want %q", in, got, want)
		}
	}
}
