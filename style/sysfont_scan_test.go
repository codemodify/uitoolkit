package style

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/fonts"
)

// fontDir writes the bundled faces into a directory of their own, so
// the scanner can be pointed at a known set rather than at whatever
// the machine running the test happens to have installed.
func fontDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		b, err := fonts.Bytes(n)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, n), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// The scan reads a directory of fonts and gets each one's family and
// weight from the file, which is the whole of what it is for.
func TestScanReadsFamiliesAndWeights(t *testing.T) {
	dir := fontDir(t, fonts.FileTitilliumRegular, fonts.FileTitilliumBold,
		fonts.FileJetBrainsRegular, fonts.FileJetBrainsBold)
	faces := map[string][]sysFace{}
	names := scanFamilies([]string{dir}, faces)

	for _, want := range []string{"Titillium Web", "JetBrains Mono"} {
		if !hasFold(names, want) {
			t.Errorf("the scan did not list %q; it listed %v", want, names)
		}
		got := faces[strings.ToLower(want)]
		if len(got) != 2 {
			t.Errorf("%s has %d faces, want a regular and a bold", want, len(got))
			continue
		}
		var reg, bold bool
		for _, f := range got {
			switch f.weight {
			case fcRegular:
				reg = true
			case fcBold:
				bold = true
			}
		}
		if !reg || !bold {
			t.Errorf("%s: regular=%v bold=%v, from weights %v", want, reg, bold, weightsOf(got))
		}
	}
}

// A weight comes from the file's OS/2 table, not from its name: the
// subfamily string is localized, so a French system would have "Gras"
// where a name reader looks for "Bold".
func TestScanWeightComesFromOS2(t *testing.T) {
	dir := fontDir(t, fonts.FileTitilliumBold)
	// Renamed, so nothing about the weight can be read off the path.
	old := filepath.Join(dir, fonts.FileTitilliumBold)
	if err := os.Rename(old, filepath.Join(dir, "anonymous.ttf")); err != nil {
		t.Fatal(err)
	}
	faces := map[string][]sysFace{}
	scanFamilies([]string{dir}, faces)
	got := faces["titillium web"]
	if len(got) != 1 {
		t.Fatalf("faces %v", got)
	}
	if got[0].weight != fcBold {
		t.Errorf("a file called anonymous.ttf came out weight %d, want %d", got[0].weight, fcBold)
	}
}

// Whatever the source, the index is keyed the same way and a lookup
// finds the same face. This is what indexFaceRecords is for: fc-list
// and the scan cannot drift apart in how a family is spelled.
func TestScanAndFontconfigIndexAlike(t *testing.T) {
	dir := fontDir(t, fonts.FileTitilliumRegular)
	scanned := map[string][]sysFace{}
	scanFamilies([]string{dir}, scanned)

	file := filepath.Join(dir, fonts.FileTitilliumRegular)
	parsed := map[string][]sysFace{}
	parseFCList([]byte("Titillium Web\t80\t0\t0\t"+file+"\n"), parsed)

	for k, want := range parsed {
		got, ok := scanned[k]
		if !ok {
			t.Errorf("fc-list keyed %q and the scan did not", k)
			continue
		}
		if len(got) != len(want) || got[0].weight != want[0].weight || got[0].file != want[0].file {
			t.Errorf("%q: the scan has %+v, fc-list %+v", k, got, want)
		}
	}
}

// A directory of things that are not fonts costs nothing and breaks
// nothing: a truncated file among six hundred must not stop a toolkit
// from starting.
func TestScanSurvivesRubbish(t *testing.T) {
	dir := t.TempDir()
	good, err := fonts.TitilliumRegular()
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("good.ttf", good)
	write("truncated.ttf", good[:64])
	write("empty.otf", nil)
	write("notafont.ttf", []byte("this is not a font at all, it is a sentence"))
	write("readme.txt", []byte("ignored: the extension is not one sfnt reads"))
	if err := os.Mkdir(filepath.Join(dir, "sub.ttf"), 0o755); err != nil {
		t.Fatal(err)
	}

	faces := map[string][]sysFace{}
	names := scanFamilies([]string{dir}, faces)
	if len(names) != 1 || !hasFold(names, "Titillium Web") {
		t.Fatalf("the scan came back with %v, want the one good font", names)
	}
}

// A directory that is not there is not an error. The lists in
// systemFontDirs name every usual place without checking, because
// checking and then walking is two answers to one question.
func TestScanIgnoresMissingDirs(t *testing.T) {
	faces := map[string][]sysFace{}
	names := scanFamilies([]string{"", "/nonexistent/fonts", filepath.Join(t.TempDir(), "nope")}, faces)
	if len(names) != 0 || len(faces) != 0 {
		t.Fatalf("names %v faces %v", names, faces)
	}
}

// The same file reached two ways is one font, not two. A symlink farm
// is how several distributions arrange /usr/share/fonts, and on macOS
// /Library/Fonts shadows the system one.
func TestScanDeduplicatesByRealPath(t *testing.T) {
	dir := fontDir(t, fonts.FileTitilliumRegular)
	link := filepath.Join(dir, "alias.ttf")
	if err := os.Symlink(filepath.Join(dir, fonts.FileTitilliumRegular), link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	faces := map[string][]sysFace{}
	scanFamilies([]string{dir}, faces)
	if got := faces["titillium web"]; len(got) != 1 {
		t.Errorf("the same font twice: %+v", got)
	}
}

// The directories are the platform's own, and each platform's list
// holds the place that platform actually keeps its fonts.
func TestSystemFontDirs(t *testing.T) {
	dirs := systemFontDirs()
	if len(dirs) == 0 {
		t.Fatal("no font directories at all")
	}
	want := map[string]string{
		"darwin":  "/System/Library/Fonts",
		"windows": "Fonts",
		"linux":   "/usr/share/fonts",
	}[runtime.GOOS]
	if want == "" {
		return
	}
	for _, d := range dirs {
		if strings.Contains(d, want) {
			return
		}
	}
	t.Errorf("%s: %v holds nothing like %q", runtime.GOOS, dirs, want)
}

// macOS keeps Helvetica, Times and Courier — the faces the older packs
// ask for by name — in Supplemental, and a list without it would find
// the system UI faces and none of the ones a theme wants.
func TestDarwinDirsIncludeSupplemental(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS only")
	}
	for _, d := range systemFontDirs() {
		if d == "/System/Library/Fonts/Supplemental" {
			return
		}
	}
	t.Errorf("no Supplemental in %v", systemFontDirs())
}

// sfntTable walks the table directory rather than reading the file, so
// it must find a table in a plain font and refuse an index a single
// font does not have.
func TestSfntTableFindsOS2(t *testing.T) {
	b, err := fonts.TitilliumRegular()
	if err != nil {
		t.Fatal(err)
	}
	r := strings.NewReader(string(b))
	os2, ok := sfntTable(r, 0, "OS/2")
	if !ok {
		t.Fatal("no OS/2 table in a font that has one")
	}
	if len(os2) < 64 {
		t.Fatalf("OS/2 is %d bytes, too short to hold fsSelection", len(os2))
	}
	if _, ok := sfntTable(r, 1, "OS/2"); ok {
		t.Error("a single font answered for face index 1")
	}
	if _, ok := sfntTable(r, 0, "nope"); ok {
		t.Error("a table that is not there was found")
	}
}

// The style reader: OS/2 wins, the name is the fallback, and an italic
// is left out however it says so.
func TestFaceStyle(t *testing.T) {
	os2 := func(usWeight, fsSelection uint16) []byte {
		b := make([]byte, 78)
		b[4], b[5] = byte(usWeight>>8), byte(usWeight)
		b[62], b[63] = byte(fsSelection>>8), byte(fsSelection)
		return b
	}
	for _, c := range []struct {
		name       string
		os2        []byte
		sub        string
		wantWeight int
		wantItalic bool
	}{
		{"regular from OS/2", os2(400, 0), "", fcRegular, false},
		{"bold from OS/2", os2(700, 0x20), "", fcBold, false},
		// The bundled JetBrains Mono Bold, which declares 558 because
		// it was cut from a variable font: the BOLD bit is what makes
		// it the family's bold face.
		{"the BOLD bit beats an odd weight class", os2(558, 0x00a0), "", fcBold, false},
		{"a regular that sets USE_TYPO_METRICS", os2(400, 0x00c0), "", fcRegular, false},
		{"semibold from OS/2", os2(600, 0), "", fcSemibold, false},
		{"medium from OS/2", os2(500, 0), "", fcMedium, false},
		{"light counts as regular", os2(300, 0), "", fcRegular, false},
		{"italic bit", os2(400, 0x0001), "", fcRegular, true},
		{"oblique bit", os2(400, 0x0200), "", fcRegular, true},
		{"OS/2 beats the name", os2(700, 0), "Regular", fcBold, false},
		{"no OS/2: the name", nil, "Bold", fcBold, false},
		{"no OS/2: italic by name", nil, "Bold Italic", fcBold, true},
		{"no OS/2: oblique by name", nil, "Oblique", fcRegular, true},
		{"no OS/2: nothing", nil, "", fcRegular, false},
		// A weight class of zero is a broken font, not a thin one.
		{"zero weight falls back", os2(0, 0), "Bold", fcBold, false},
	} {
		w, it := faceStyle(c.os2, c.sub)
		if w != c.wantWeight || it != c.wantItalic {
			t.Errorf("%s: weight %d italic %v, want %d %v", c.name, w, it, c.wantWeight, c.wantItalic)
		}
	}
}

func hasFold(xs []string, want string) bool {
	for _, x := range xs {
		if strings.EqualFold(x, want) {
			return true
		}
	}
	return false
}

func weightsOf(fs []sysFace) []int {
	out := make([]int, len(fs))
	for i, f := range fs {
		out[i] = f.weight
	}
	return out
}
