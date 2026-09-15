package style

import (
	"os"
	"strings"
	"sync"
	"testing"
)

// fakeFonts swaps the fontconfig index for lines of fc-list output.
func fakeFonts(t *testing.T, lines ...string) {
	t.Helper()
	old := fcList
	fcList = func() ([]byte, error) { return []byte(strings.Join(lines, "\n") + "\n"), nil }
	sysIndex.once = sync.Once{}
	sysIndex.faces = nil
	t.Cleanup(func() {
		fcList = old
		sysIndex.once = sync.Once{}
		sysIndex.faces = nil
	})
}

func TestFCListIndex(t *testing.T) {
	fakeFonts(t,
		"Tahoma\t80\t0\t0\t/f/tahoma.ttf",
		"Tahoma\t200\t0\t0\t/f/tahomabd.ttf",
		"Noto Sans,Noto Sans Display\t80\t0\t0\t/f/NotoSans-Regular.ttf",
		"Noto Sans\t80\t100\t0\t/f/NotoSans-Italic.ttf",   // italic: skipped
		"Fixed\t80\t0\t0\t/f/6x13.pcf.gz",                 // bitmap: skipped
		"Adwaita Sans\t[0 210]\t0\t0\t/f/AdwaitaSans.ttf", // variable range line: skipped
		"Adwaita Sans\t80\t0\t0\t/f/AdwaitaSans.ttf",
		"Adwaita Sans\t200\t0\t458752\t/f/AdwaitaSans.ttf", // named instance
		"Cantarell\t80\t0\t2\t/f/fonts.ttc",
	)
	faces := systemFaces()
	if n := len(faces["tahoma"]); n != 2 {
		t.Fatalf("tahoma faces %d, want 2", n)
	}
	if n := len(faces["noto sans"]); n != 1 {
		t.Fatalf("noto sans faces %d, want the upright one", n)
	}
	if _, ok := faces["noto sans display"]; !ok {
		t.Fatal("a line's second family name is an alias of the same face")
	}
	if _, ok := faces["fixed"]; ok {
		t.Fatal("bitmap fonts cannot be drawn and must not resolve")
	}
	if f := faces["cantarell"]; len(f) != 1 || f[0].index != 2 {
		t.Fatalf("collection index lost: %+v", f)
	}

	// Bold: a static bold face when there is one…
	if f, synth, ok := pickSystemFace("Tahoma", WeightBold); !ok || synth != 0 || f.file != "/f/tahomabd.ttf" {
		t.Fatalf("Tahoma bold: %+v synth=%v ok=%v", f, synth, ok)
	}
	// …else the regular outline drawn heavier (sfnt cannot draw a variable
	// font's bold instance).
	if f, synth, ok := pickSystemFace("Adwaita Sans", WeightBold); !ok || synth != synthBold || f.named {
		t.Fatalf("Adwaita Sans bold: %+v synth=%v ok=%v", f, synth, ok)
	}
	if _, synth, _ := pickSystemFace("Noto Sans", WeightRegular); synth != 0 {
		t.Fatal("a regular face is never synthesized")
	}
	// Semibold with only regular and bold installed: the bold face, as CSS
	// matches upward; medium: the regular a little heavier.
	if f, synth, ok := pickSystemFace("Tahoma", WeightSemibold); !ok || synth != 0 || f.file != "/f/tahomabd.ttf" {
		t.Fatalf("Tahoma semibold: %+v synth=%v", f, synth)
	}
	if f, synth, ok := pickSystemFace("Tahoma", WeightMedium); !ok || synth <= 0 || synth >= synthBold || f.weight != fcRegular {
		t.Fatalf("Tahoma medium: %+v synth=%v", f, synth)
	}

	if got := ResolveFont([]string{"Segoe UI", "Tahoma", "Noto Sans"}); got != "Tahoma" {
		t.Fatalf("first installed family: %q", got)
	}
	if got := ResolveFont([]string{"Segoe UI", "Lucida Grande"}); got != "" {
		t.Fatalf("nothing installed should resolve to nothing, got %q", got)
	}
	if got := ResolveFont([]string{"Segoe UI", "Titillium Web"}); got != FamilyUI {
		t.Fatalf("the bundled face counts as installed, got %q", got)
	}
}

// UITK_SYSTEM_FONTS=0 keeps every look on the bundled faces.
func TestSystemFontsOff(t *testing.T) {
	t.Setenv(SystemFontsEnv, "0")
	fakeFonts(t, "Tahoma\t80\t0\t0\t/f/tahoma.ttf")
	if FontInstalled("Tahoma") {
		t.Fatal("installed fonts should be off")
	}
}

// A pack reads in its era's typeface when installed: the pack's own entry
// first, then its engine's; a theme's own "fonts" win over both; a family
// whose file cannot be read falls back to the bundled face.
func TestLooksReadInTheirErasTypeface(t *testing.T) {
	fakeFonts(t,
		"Tahoma\t80\t0\t0\t/nonexistent/tahoma.ttf",
		"Nimbus Sans\t80\t0\t0\t/nonexistent/NimbusSans.otf",
	)
	cases := map[string]string{
		"luna":    "Tahoma",      // XP
		"win2000": "Tahoma",      // the pack's own entry
		"next":    "Nimbus Sans", // Helvetica's metric twin
		"motif":   "Nimbus Sans",
		"aero":    FamilyUI, // Segoe UI and its look-alikes: none installed
		"dark":    FamilyUI, // the toolkit's own look keeps the bundled face
	}
	for pack, want := range cases {
		lk := mustLook(t, pack)
		if got := lk.UIFamily(); got != want {
			t.Errorf("%s reads in %q, want %q", pack, got, want)
		}
		// The file is missing: the face falls back and still draws.
		if f := lk.Font(); f == nil || f.Advance("Hello") <= 0 {
			t.Errorf("%s: no usable font", pack)
		}
	}

	tok := withEraFonts(ThemeTokens{Engine: "luna", Fonts: FontPrefs{UI: []string{"Nimbus Sans"}}}, "luna")
	if ui, _ := lookFamilies(tok); ui != "Nimbus Sans" {
		t.Fatalf("a theme's own fonts should win, got %q", ui)
	}
	if tok.Fonts.Mono == nil {
		t.Fatal("unset mono should still come from the era")
	}
}

// theme.json carries "fonts" both ways.
func TestThemeJSONFonts(t *testing.T) {
	raw := `{"label":"Mine","palette":"light","engine":"luna","fonts":{"ui":["  Verdana ", ""],"mono":["Consolas"]}}`
	p, err := parseThemeFile("mine", []byte(raw), ThemeSourceUser)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Tokens.Fonts; len(got.UI) != 1 || got.UI[0] != "Verdana" || len(got.Mono) != 1 || got.Mono[0] != "Consolas" {
		t.Fatalf("fonts %+v", got)
	}
	doc := themeDoc("Mine", "", p.Tokens)
	if doc.Fonts == nil || doc.Fonts.UI[0] != "Verdana" {
		t.Fatalf("export lost the fonts: %+v", doc.Fonts)
	}
	// A pack that names no fonts writes none.
	if d := themeDoc("x", "", ThemeTokens{}); d.Fonts != nil {
		t.Fatalf("empty fonts exported: %+v", d.Fonts)
	}
}

// On a machine with fontconfig, a NeXT look really reads in Helvetica or
// its twin, and its bold is heavier than its regular.
func TestInstalledHelveticaDraws(t *testing.T) {
	if os.Getenv(SystemFontsEnv) == "0" || !FontInstalled("Nimbus Sans") && !FontInstalled("Helvetica") {
		t.Skip("no Helvetica or Nimbus Sans installed")
	}
	lk := mustLook(t, "next")
	if lk.UIFamily() == FamilyUI {
		t.Fatalf("NeXT should read in an installed Helvetica, got the bundled face")
	}
	reg, bold := lk.Font(), lk.BoldFont()
	if bold.Advance("Workspace") <= reg.Advance("Workspace") {
		t.Fatalf("bold %v is not wider than regular %v", bold.Advance("Workspace"), reg.Advance("Workspace"))
	}
}

// Windows draws selected text white (HighlightText) even on its light
// blues; other looks keep the most readable colour.
func TestSelectionTextPinnedByPack(t *testing.T) {
	white := Hex("#ffffff")
	for _, pack := range []string{"aero", "win8", "win10"} {
		lk := mustLook(t, pack)
		if got := lk.selectedText(lk.palette.Selection); got != white {
			t.Errorf("%s: selected text %s, want white", pack, colorHexPadded(got))
		}
	}
	lk := mustLook(t, "light")
	sel := lk.palette.Selection
	if got := lk.selectedText(sel); ContrastRatio(got, Mix(lk.palette.Field, sel, sel.A)) < 4.5 {
		t.Errorf("light: selected text %s does not read on its selection", colorHexPadded(got))
	}
}

// "Reduce motion" is saved in look.json and, with UITK_ANIMATIONS, decides
// whether controls animate.
func TestReduceMotionPref(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(AnimationsEnv, "")
	defer SetReduceMotion(false)
	a := DefaultAppearance()
	a.ReduceMotion = true
	if err := SaveAppearance(a); err != nil {
		t.Fatal(err)
	}
	if !LoadAppearance().ReduceMotion {
		t.Fatal("reduce motion did not survive look.json")
	}
	SetReduceMotion(false)
	if !Animations() {
		t.Fatal("animations should be on by default")
	}
	SetReduceMotion(true)
	if Animations() {
		t.Fatal("reduce motion should turn animations off")
	}
	SetReduceMotion(false)
	t.Setenv(AnimationsEnv, "0")
	if Animations() {
		t.Fatal("UITK_ANIMATIONS=0 should turn animations off")
	}
}
