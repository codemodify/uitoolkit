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
// present. Fonts are found through fontconfig, indexed once per process.

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
		sysIndex.faces = map[string][]sysFace{}
		if os.Getenv(SystemFontsEnv) == "0" {
			return
		}
		if out, err := fcList(); err == nil {
			parseFCList(out, sysIndex.faces)
		}
	})
	return sysIndex.faces
}

// parseFCList reads fc-list lines "family[,alias…]\tweight\tslant\tindex\tfile".
func parseFCList(out []byte, into map[string][]sysFace) {
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
		face := sysFace{file: file, index: idx & 0xffff, weight: w, named: idx>>16 != 0}
		for _, fam := range strings.Split(f[0], ",") {
			if fam = strings.ToLower(strings.TrimSpace(fam)); fam != "" {
				into[fam] = append(into[fam], face)
			}
		}
	}
}

// fontconfig weights.
const (
	fcRegular = 80
	fcBold    = 200
)

// pickSystemFace is family's installed face for w: a static face of that
// weight when there is one; otherwise, for bold, the regular outline drawn
// heavier (synth), and for regular the nearest static face.
func pickSystemFace(family string, w Weight) (face sysFace, synth, ok bool) {
	faces := systemFaces()[strings.ToLower(strings.TrimSpace(family))]
	want := fcRegular
	if w >= WeightBold {
		want = fcBold
	}
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
		return sysFace{}, false, false
	}
	face = faces[best]
	synth = want == fcBold && face.weight < 150
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
	synth bool
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
	src, err := os.ReadFile(face.file)
	if err != nil {
		systemOTFaces.m[key] = nil
		return nil
	}
	f, err := parseFaceBytes(face.file, src, face.index)
	if err != nil {
		systemOTFaces.m[key] = nil
		return nil
	}
	if synth {
		f.embolden = synthBold
	}
	systemOTFaces.m[key] = f
	return f
}
