package style

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Installed fonts. A pack names the typefaces of its era, most wanted first
// (XP's Tahoma, Vista's Segoe UI, GNOME's Cantarell, NeXT's Helvetica), then
// open look-alikes; the first one installed wins, and the bundled Titillium
// Web and JetBrains Mono stay the last resort, so no pack needs a font to be
// present. Fonts are indexed once per process.
//
// Through fontconfig where there is one, because it is the machine's own
// answer: it knows the configured directories, the user's additions and the
// aliases. macOS has no fontconfig and Windows has none either, and there
// the directories are walked and each file read instead — see
// sysfont_scan.go, which is what made an era's typeface appear on those two
// platforms at all.
//
// Through fc-list, and never through fc-match: that is the whole of why a
// pack that asks for a face nobody has falls through to the next name on
// its list instead of drawing in something else entirely. fc-match is a
// matcher — it is *required* to answer, and on a machine with the usual
// Arch fontconfig it answers "Noto Sans" for every family on earth,
// including families that are not installed and including the bundled
// Titillium Web. A toolkit that asked it "have you got Lucida Sans?" would
// be told yes, 131 times over, and every era would quietly read in one
// face. fc-list enumerates instead: it says what is actually on the disk,
// keyed by the family the file itself declares, and [ResolveFont] compares
// the names. See TestLookupNeverAsksFontconfigToMatch.

// SystemFontsEnv set to "0" turns installed-font lookup off (reproducible
// screenshots, CI): every look then reads in the bundled faces.
const SystemFontsEnv = "UITK_SYSTEM_FONTS"

// sysFace is one installed upright face.
type sysFace struct {
	file   string
	index  int  // face in a collection file
	weight int  // fontconfig weight: 80 regular, 200 bold
	named  bool // a variable font's named instance (sfnt draws only the default outline)
}

var sysIndex struct {
	once  sync.Once
	faces map[string][]sysFace // lower-case family → upright faces
	// names are the families a chooser lists: the name each file
	// declares first, in its own spelling, sorted and deduplicated. The
	// rest of a font's comma-separated names are aliases of the same
	// file — the weight-suffixed spellings fontconfig invents ("Noto
	// Sans Bengali UI Thin") and localized names — and they stay
	// resolvable without standing in a list of 600 as if they were
	// typefaces of their own.
	names []string
}

// fcList lists the installed fonts through fontconfig. Tests replace it.
var fcList = func() ([]byte, error) {
	path, err := exec.LookPath("fc-list")
	if err != nil {
		return nil, err
	}
	return exec.Command(path, "--format", "%{family}\t%{weight}\t%{slant}\t%{index}\t%{file}\n").Output()
}

// PrefetchSystemFonts builds the installed-font index in the background
// (fontconfig takes a dozen milliseconds), so the first look does not wait
// for it. The app package calls it while the display connection comes up.
func PrefetchSystemFonts() { go systemFaces() }

func systemFaces() map[string][]sysFace {
	sysIndex.once.Do(func() {
		sysIndex.faces, sysIndex.names = map[string][]sysFace{}, nil
		if os.Getenv(SystemFontsEnv) == "0" {
			return
		}
		if out, err := fcList(); err == nil && len(out) > 0 {
			sysIndex.names = parseFCList(out, sysIndex.faces)
			return
		}
		// No fontconfig, or it answered nothing: macOS has none at all
		// and Windows has none either, and without this every pack on
		// both falls through to the bundled faces. Walking the font
		// directories and reading each file is what fc-list does
		// underneath (sysfont_scan.go).
		sysIndex.names = scanFamilies(systemFontDirs(), sysIndex.faces)
	})
	return sysIndex.faces
}

// systemFamilies is the sorted list of installed families a chooser
// offers (the index is built if it has not been).
func systemFamilies() []string {
	systemFaces()
	return sysIndex.names
}

// faceRecord is one upright face, as either source found it: fc-list
// (parseFCList) or a walk of the font directories (sysfont_scan.go).
// Both produce these and indexFaceRecords does the rest, so the two
// paths cannot drift in how a family is keyed or listed.
type faceRecord struct {
	// names are the family's spellings, the one to list first. fc-list
	// gives a comma-separated set; a file gives its typographic family
	// and its family, which differ for a face like "Helvetica Neue
	// Bold" that names itself a family of its own.
	names  []string
	weight int  // fontconfig's scale: 80 regular, 200 bold
	index  int  // face within a collection file
	named  bool // a variable font's named instance
	file   string
}

// indexFaceRecords files records under every spelling of their family
// and returns the primary names a chooser lists, sorted and
// deduplicated.
func indexFaceRecords(records []faceRecord, into map[string][]sysFace) []string {
	primary := map[string]string{}
	for _, r := range records {
		face := sysFace{file: r.file, index: r.index, weight: r.weight, named: r.named}
		for i, fam := range r.names {
			fam = strings.TrimSpace(fam)
			key := strings.ToLower(fam)
			if key == "" {
				continue
			}
			into[key] = append(into[key], face)
			if i == 0 {
				if _, seen := primary[key]; !seen {
					primary[key] = fam
				}
			}
		}
	}
	return sortedFamilies(primary)
}

// parseFCList reads fc-list lines "family[,alias…]\tweight\tslant\tindex\tfile"
// into the lookup index, and returns the primary family names — the first
// name on each line, which is the one the file declares — sorted and
// deduplicated.
func parseFCList(out []byte, into map[string][]sysFace) []string {
	var records []faceRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 5 {
			continue
		}
		file := f[4]
		switch strings.ToLower(filepath.Ext(file)) {
		case ".ttf", ".otf", ".ttc", ".otc":
		default:
			continue // bitmap and Type 1 faces: sfnt reads OpenType only
		}
		if slant, err := strconv.Atoi(f[2]); err != nil || slant != 0 {
			continue
		}
		// A variable font's own line carries a weight range ("[0 210]");
		// its named instances follow as lines of their own.
		w, err := strconv.Atoi(f[1])
		if err != nil {
			continue
		}
		idx, _ := strconv.Atoi(f[3])
		records = append(records, faceRecord{
			names:  strings.Split(f[0], ","),
			weight: w,
			index:  idx & 0xffff,
			named:  idx>>16 != 0,
			file:   file,
		})
	}
	return indexFaceRecords(records, into)
}

// fontconfig weights.
const (
	fcRegular  = 80
	fcMedium   = 100
	fcSemibold = 180
	fcBold     = 200
)

// fcWeight is fontconfig's number for a CSS weight (80 regular, 100
// medium, 180 demibold, 200 bold); cssWeight goes back.
func fcWeight(w Weight) int {
	switch {
	case w >= WeightBold:
		return fcBold
	case w >= WeightSemibold:
		return fcSemibold
	case w >= WeightMedium:
		return fcMedium
	}
	return fcRegular
}

func cssWeight(fc int) Weight {
	switch {
	case fc >= fcBold:
		return WeightBold
	case fc >= fcSemibold:
		return WeightSemibold
	case fc >= fcMedium:
		return WeightMedium
	}
	return WeightRegular
}

// pickSystemFace is family's installed face for w: a static face of that
// weight when there is one; otherwise the nearest face, drawn heavier
// (synth, a fraction of the size) when it is lighter than asked for.
func pickSystemFace(family string, w Weight) (face sysFace, synth float32, ok bool) {
	faces := systemFaces()[strings.ToLower(strings.TrimSpace(family))]
	want := fcWeight(w)
	best, bestD := -1, 1<<30
	for i, f := range faces {
		if f.named && f.weight != fcRegular {
			continue // sfnt cannot draw a variable font's instances
		}
		d := f.weight - want
		if d < 0 {
			d = -d
		}
		if d < bestD {
			best, bestD = i, d
		}
	}
	if best < 0 {
		return sysFace{}, 0, false
	}
	face = faces[best]
	if s := synthFor(w, cssWeight(face.weight)); s >= 0.005 {
		synth = s
	}
	return face, synth, true
}

// FontInstalled reports whether family is bundled or installed.
func FontInstalled(family string) bool {
	if bundledFamily(family) != "" {
		return true
	}
	_, ok := systemFaces()[strings.ToLower(strings.TrimSpace(family))]
	return ok
}

// ResolveFont is the first of families that is bundled or installed, or ""
// when none is (the caller keeps the bundled face).
func ResolveFont(families []string) string {
	for _, f := range families {
		if b := bundledFamily(f); b != "" {
			return b
		}
		if FontInstalled(f) {
			return strings.TrimSpace(f)
		}
	}
	return ""
}

// bundledFamily maps the names of the bundled faces onto their family, or "".
func bundledFamily(family string) string {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case strings.ToLower(FamilyUI), "ui":
		return FamilyUI
	case strings.ToLower(FamilyMono), "mono":
		return FamilyMono
	}
	return ""
}

// systemOTFaces caches parsed installed faces by file, face index and synth.
var systemOTFaces = struct {
	mu sync.Mutex
	m  map[sysFaceKey]*otFace
}{m: map[sysFaceKey]*otFace{}}

type sysFaceKey struct {
	file  string
	index int
	synth float32
}

// synthBold is how much heavier a synthesized bold draws, as a fraction of
// the size (FreeType emboldens by about 1/24 em).
const synthBold = 0.04

// systemOTFace loads family's installed face for w, or nil.
func systemOTFace(family string, w Weight) *otFace {
	face, synth, ok := pickSystemFace(family, w)
	if !ok {
		return nil
	}
	key := sysFaceKey{face.file, face.index, synth}
	systemOTFaces.mu.Lock()
	defer systemOTFaces.mu.Unlock()
	if f, ok := systemOTFaces.m[key]; ok {
		return f
	}
	src, err := readFontFile(face.file)
	if err != nil {
		systemOTFaces.m[key] = nil
		return nil
	}
	f, err := parseFaceBytes(face.file, src, face.index)
	if err != nil {
		systemOTFaces.m[key] = nil
		return nil
	}
	f.embolden = synth
	systemOTFaces.m[key] = f
	return f
}

// ---- what a font chooser offers -------------------------------------------

// BundledFamilies are the two typefaces the toolkit carries in its own
// binary, in the order a chooser lists them: the UI face first, the
// monospaced one second. They are the only two families that are on every
// machine, and they are what every pack falls back to, so they lead the
// list rather than standing among the installed families under T and J.
func BundledFamilies() []string { return []string{FamilyUI, FamilyMono} }

// ListFontFamilies is what a font chooser offers: the two bundled
// families, then every upright family fontconfig reports installed,
// sorted, without repeating the bundled two.
//
// Both choosers are handed the same list. The toolkit has two font roles
// and a pack names a typeface for each, so there are two choosers — one
// list cannot set both roles honestly — but neither list is filtered to
// "sans" or "mono", because fontconfig's spacing field is not reliable
// enough to hide a family behind it and because a person who wants
// JetBrains Mono for the interface, or Titillium Web in a code view, is
// not making a mistake the toolkit should correct.
//
// With UITK_SYSTEM_FONTS=0 it is the bundled two and nothing else, which
// is what makes a screenshot of the chooser reproducible.
func ListFontFamilies() []string {
	bundled := BundledFamilies()
	skip := map[string]bool{}
	for _, f := range bundled {
		skip[strings.ToLower(f)] = true
	}
	installed := systemFamilies()
	out := make([]string, 0, len(bundled)+len(installed))
	out = append(out, bundled...)
	for _, fam := range installed {
		if !skip[strings.ToLower(fam)] {
			out = append(out, fam)
		}
	}
	return out
}
