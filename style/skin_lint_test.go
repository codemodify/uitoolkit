package style

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/codemodify/paintengine2d"
)

// The lint command's findings: what a skin that loads would want its author
// to hear about, each keyed like a loader error.

func lintKeys(ws []SkinWarning) string {
	var b strings.Builder
	for _, w := range ws {
		b.WriteString(w.Key)
		b.WriteString(" | ")
		b.WriteString(w.Msg)
		b.WriteString("\n")
	}
	return b.String()
}

func TestLintNamesWhatAnAuthorWouldWantToKnow(t *testing.T) {
	doc := `{
	  "skin": 1,
	  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" },
	              "small": { "1x": "art/sheet.png" } },
	  "sprites": {
	    "face": { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] },
	    "mark": { "sheet": "chrome", "at": [16, 0, 8, 8] },
	    "led.1": { "sheet": "chrome", "at": [16, 0, 4, 4] },
	    "led.2": { "sheet": "chrome", "at": [20, 0, 4, 4] }
	  },
	  "parts": { "button": { "states": { "normal": "face", "hover": "mark" } } },
	  "layouts": { "p": { "size": [40, 20], "slots": {
	    "key": { "at": [0, 0, 8, 8], "art": { "normal": "face", "hover": "mark" } },
	    "lamp": { "at": [10, 0, 8, 8], "art": "mark" } } } }
	}`
	sk := loadTestSkin(t, doc)
	got := lintKeys(LintSkin(sk))
	for _, want := range []string{
		`parts.button.states | no "pressed" art: it shows the "hover" face`,
		`parts.button.states | no "disabled" art: it shows the "normal" face`,
		`layouts.p.slots.key.art | no "pressed" art`,
		`sprites | 2 that no part, window or layout binds`,
		`led* (2)`,
		`sheets.small | no art at 2x or above`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("lint does not say %q; it says:\n%s", want, got)
		}
	}
	// A lamp — one sprite for every state — asks for nothing more, and
	// "mark" and "face" are bound.
	if strings.Contains(got, "slots.lamp") || strings.Contains(got, "mark*") || strings.Contains(got, "face*") {
		t.Errorf("lint warned about something that is fine:\n%s", got)
	}
	// Keyed like a loader error, and marked as a warning.
	w := LintSkin(sk)[0]
	if s := w.String(); !strings.HasPrefix(s, "probe: skin.json: "+w.Key+": warning: ") {
		t.Errorf("a warning prints as %q", s)
	}
}

// A 2× file that is not twice the 1× one, and a pixelated sheet whose 2×
// art is not its 1× art doubled, are both art too small for 2×.
func TestLintCatchesArtTooSmallFor2x(t *testing.T) {
	half := func(w, h int) []byte {
		img := paintengine2d.NewImage(w, h)
		img.Clear(paintengine2d.RGB(0.5, 0.5, 0.5))
		var buf bytes.Buffer
		if err := img.WritePNG(&buf); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	fs := skinTestFS(t, `{"skin": 1, "sheets": {"chrome": {"1x": "art/sheet.png", "2x": "art/short.png"}}}`)
	fs["art/short.png"] = &fstest.MapFile{Data: half(40, 40)}
	sk, err := LoadSkinFS(fs, "probe")
	if err != nil {
		t.Fatal(err)
	}
	if got := lintKeys(LintSkin(sk)); !strings.Contains(got, "sheets.chrome.2x | short.png is 40×40, smaller than twice sheet.png's 32×32") {
		t.Errorf("a 2x file smaller than twice the 1x is not reported:\n%s", got)
	}
	// A pixelated sheet whose 2× file is the right size but not its 1× art
	// doubled: a flat grey is not the fixture's red, green and blue.
	fs = skinTestFS(t, `{"skin": 1, "design": {"pixelated": true}, "sheets": {"chrome": {"1x": "art/sheet.png", "2x": "art/grey.png"}}}`)
	fs["art/grey.png"] = &fstest.MapFile{Data: half(64, 64)}
	if sk, err = LoadSkinFS(fs, "probe"); err != nil {
		t.Fatal(err)
	}
	if got := lintKeys(LintSkin(sk)); !strings.Contains(got, "sheets.chrome.2x | a pixelated sheet's 2x art should be its 1x art with every pixel doubled") {
		t.Errorf("a pixel sheet not doubled is not reported:\n%s", got)
	}
}

// Every skin the toolkit ships lints without a warning about its art: the
// generator draws every sheet at 2× and doubles every pixel sheet exactly.
func TestShippedSkinsHaveArtFor2x(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, p := range ListSkins() {
		sk, _ := LoadSkin(p.Name)
		for _, w := range LintSkin(sk) {
			if strings.HasPrefix(w.Key, "sheets.") {
				t.Errorf("%s", w)
			}
		}
	}
}

// LoadSkinFile reads what the lint command is pointed at: a directory, or
// the same skin as one .uskin file.
func TestLoadSkinFileReadsADirectoryOrAnArchive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := installSkin(t, "linted", skinTestDoc)
	sk, err := LoadSkinFile(filepath.Dir(path))
	if err != nil || sk.Name != "linted" {
		t.Fatalf("directory: %v, %v", sk, err)
	}
	// The same skin, zipped as one file.
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	for _, f := range []string{SkinFile, "art/sheet.png", "art/sheet2.png"} {
		b, err := os.ReadFile(filepath.Join(filepath.Dir(path), f))
		if err != nil {
			t.Fatal(err)
		}
		w, _ := zw.Create(f)
		_, _ = w.Write(b)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "zipped"+SkinArchiveExt)
	if err := os.WriteFile(archive, zbuf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if sk, err := LoadSkinFile(archive); err != nil || sk.Name != "zipped" || len(sk.Sprites) != 2 {
		t.Fatalf("archive: %v, %v", sk, err)
	}
	if _, err := LoadSkinFile(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("a path with nothing there loaded")
	}
	bad := filepath.Join(t.TempDir(), "broken")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, SkinFile), []byte(`{"skin": 1, "parts": {"widget": {}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSkinFile(bad); err == nil || !strings.Contains(err.Error(), "parts.widget") {
		t.Errorf("a broken skin's refusal does not name its key: %v", err)
	}
}
