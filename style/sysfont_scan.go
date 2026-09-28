package style

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/image/font/sfnt"
)

// Finding the installed fonts where there is no fontconfig.
//
// fc-list is Linux's, and [systemFaces] asks it first because it is
// the machine's own answer — it knows the configured directories, the
// user's additions and the aliases. macOS has no fontconfig at all and
// Windows has none either, so on both of them the index came back
// empty and every pack fell through to the bundled Titillium Web: a
// toolkit that ships 131 packs spanning four decades drew all of them
// in one face.
//
// This is the fallback. It walks the directories the platform keeps
// its fonts in and reads each file's own name and OS/2 tables, which
// is what fc-list does underneath. It is pure Go and has no build tag,
// so the same code answers on all three platforms and can be tested on
// any of them.
//
// Reading the file itself, and never a registry or a font manager, is
// the same decision [parseFCList] documents: the family a file
// declares is the family the toolkit will draw, because the toolkit
// draws the file. A name that came from somewhere else could name a
// face that is not there.

// systemFontDirs is where this platform keeps its fonts, most general
// first. A directory that does not exist is skipped, so the list can
// name every usual place without checking.
func systemFontDirs() []string {
	home, _ := os.UserHomeDir()
	join := func(base string, rest ...string) string {
		if base == "" {
			return ""
		}
		return filepath.Join(append([]string{base}, rest...)...)
	}
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/System/Library/Fonts",
			// Where macOS keeps Helvetica, Times, Courier and the rest
			// of the faces the older packs ask for by name.
			"/System/Library/Fonts/Supplemental",
			"/Library/Fonts",
			join(home, "Library", "Fonts"),
		}
	case "windows":
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		return []string{
			filepath.Join(windir, "Fonts"),
			// Fonts a user installed without administrator rights, which
			// is where anything added since Windows 10 1803 tends to be.
			join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts"),
		}
	default:
		return []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			join(home, ".local", "share", "fonts"),
			join(home, ".fonts"),
		}
	}
}

// scanFontDirs walks dirs and returns one record per upright face.
//
// It is deliberately forgiving: a font it cannot read is skipped, not
// reported. There is no useful thing to do about one bad file among
// six hundred, and a toolkit that refused to start because a stray
// .ttf in ~/.fonts was truncated would be worse than one that draws
// the other five hundred and ninety-nine.
func scanFontDirs(dirs []string) []faceRecord {
	files := fontFilesIn(dirs)
	if len(files) == 0 {
		return nil
	}
	// One goroutine per core over the files. Reading a font is all
	// parsing and the files are independent, and it is worth doing:
	// a Mac keeps its system faces in large .ttc collections, and
	// eight hundred milliseconds in one thread is long enough for the
	// first window to want a font before the index has one.
	//
	// The results go into a slot per file rather than a shared slice,
	// so the order is the walk's and two runs on one machine agree.
	per := make([][]faceRecord, len(files))
	workers := min(runtime.NumCPU(), len(files))
	var wg sync.WaitGroup
	next := make(chan int, len(files))
	for i := range files {
		next <- i
	}
	close(next)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				per[i] = scanFontFile(files[i])
			}
		}()
	}
	wg.Wait()

	var out []faceRecord
	for _, r := range per {
		out = append(out, r...)
	}
	return out
}

// fontFilesIn is every font file under dirs, each one once.
func fontFilesIn(dirs []string) []string {
	var files []string
	seen := map[string]bool{}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // an unreadable subtree is skipped
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".otf", ".ttc", ".otc":
			default:
				return nil // sfnt reads OpenType only, as fc-list is filtered to
			}
			// The same file reachable two ways — a symlink farm, or
			// /Library/Fonts shadowing a system one — is one font.
			real, rerr := filepath.EvalSymlinks(path)
			if rerr != nil {
				real = path
			}
			if seen[real] {
				return nil
			}
			seen[real] = true
			files = append(files, path)
			return nil
		})
	}
	return files
}

// scanFontFile reads one font or collection; nil for anything it
// cannot make sense of.
func scanFontFile(path string) []faceRecord {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	// ReaderAt rather than the whole file: the system fonts of a Mac
	// run to hundreds of megabytes, and all that is wanted from each
	// is two small tables.
	c, err := sfnt.ParseCollectionReaderAt(f)
	if err != nil {
		return nil
	}
	var out []faceRecord
	for i := range c.NumFonts() {
		font, err := c.Font(i)
		if err != nil {
			continue
		}
		// OS/2 is read from the file rather than from the parsed font:
		// sfnt exposes no raw table, and the two numbers wanted from it
		// are four bytes each.
		os2, _ := sfntTable(f, i, "OS/2")
		if r, ok := faceRecordOf(font, os2, path, i); ok {
			out = append(out, r)
		}
	}
	return out
}

// sfntTable is the bytes of one table of the index'th font in r,
// which may be a single font or a ttcf collection.
//
// Only the table directory is walked, so this reads a few hundred
// bytes of a file that may be ninety megabytes.
func sfntTable(r io.ReaderAt, index int, tag string) ([]byte, bool) {
	at := func(off int64, n int) ([]byte, bool) {
		b := make([]byte, n)
		if _, err := r.ReadAt(b, off); err != nil {
			return nil, false
		}
		return b, true
	}
	head, ok := at(0, 12)
	if !ok {
		return nil, false
	}
	var fontOff int64
	if string(head[:4]) == "ttcf" {
		n := int(binary.BigEndian.Uint32(head[8:]))
		if index < 0 || index >= n {
			return nil, false
		}
		off, ok := at(12+int64(index)*4, 4)
		if !ok {
			return nil, false
		}
		fontOff = int64(binary.BigEndian.Uint32(off))
		if head, ok = at(fontOff, 12); !ok {
			return nil, false
		}
	} else if index != 0 {
		return nil, false
	}
	numTables := int(binary.BigEndian.Uint16(head[4:]))
	if numTables <= 0 || numTables > 512 {
		return nil, false
	}
	dir, ok := at(fontOff+12, numTables*16)
	if !ok {
		return nil, false
	}
	for i := range numTables {
		rec := dir[i*16:]
		if string(rec[:4]) != tag {
			continue
		}
		off := int64(binary.BigEndian.Uint32(rec[8:]))
		length := int(binary.BigEndian.Uint32(rec[12:]))
		if length <= 0 || length > 1<<20 {
			return nil, false
		}
		return at(off, length)
	}
	return nil, false
}

// faceRecordOf is one face's record; ok is false for an italic or for
// a face with no family name, both of which fc-list also leaves out.
func faceRecordOf(font *sfnt.Font, os2 []byte, path string, index int) (faceRecord, bool) {
	var b sfnt.Buffer
	name := func(id sfnt.NameID) string {
		s, err := font.Name(&b, id)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(s)
	}
	// The typographic family is the one a chooser should show and the
	// one a pack names: "Helvetica Neue", where the family alone may be
	// "Helvetica Neue Bold" so that old software sees four families of
	// one face each. Both are registered, as fc-list registers its
	// comma-separated list, so either spelling resolves.
	fam, typo := name(sfnt.NameIDFamily), name(sfnt.NameIDTypographicFamily)
	names := []string{}
	for _, n := range []string{typo, fam} {
		if n != "" && !slicesHas(names, n) {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return faceRecord{}, false
	}
	sub := name(sfnt.NameIDTypographicSubfamily)
	if sub == "" {
		sub = name(sfnt.NameIDSubfamily)
	}
	weight, italic := faceStyle(os2, sub)
	if italic {
		return faceRecord{}, false
	}
	return faceRecord{names: names, weight: weight, index: index, file: path}, true
}

// faceStyle is the face's weight on fontconfig's scale and whether it
// slants.
//
// OS/2 is the authority — usWeightClass is a number and fsSelection
// has a bit for italic — and the subfamily name is the fallback for
// the faces that have no OS/2 table, which are mostly old CFF fonts
// Apple still ships. Reading the name first would be wrong in the
// other direction: it is localized, so a French system would have
// "Gras" where this looks for "Bold".
func faceStyle(os2 []byte, subfamily string) (weight int, italic bool) {
	if len(os2) >= 64 {
		usWeight := int(be16(os2[4:]))
		fsSelection := be16(os2[62:])
		italic = fsSelection&0x0001 != 0 || fsSelection&0x0200 != 0 // ITALIC, OBLIQUE
		switch {
		case fsSelection&0x0020 != 0: // BOLD
			// The bit says "this is the family's bold face" and the
			// number says how heavy it is, and they do not always
			// agree. The toolkit's bundled JetBrains Mono Bold
			// declares usWeightClass 558 — it was cut from a variable
			// font and the axis value came with it — so reading the
			// number alone files the bold face under medium and a
			// pack that asks for bold gets the regular emboldened.
			return fcBold, italic
		case usWeight >= 1 && usWeight <= 1000:
			return fcWeight(nearestWeight(usWeight)), italic
		}
		// A weight class of 0 is a bug in the font, not "thin"; fall
		// through and read the name, but keep the slant, which was a
		// separate bit and is not in doubt.
		return fcWeight(weightFromName(subfamily)), italic
	}
	lower := strings.ToLower(subfamily)
	return fcWeight(weightFromName(subfamily)),
		strings.Contains(lower, "italic") || strings.Contains(lower, "oblique")
}

// nearestWeight snaps an OS/2 usWeightClass to the four weights the
// toolkit draws. Anything below medium is regular: the toolkit has no
// light, and a Light face standing in for a regular one is worse than
// the regular it already has.
func nearestWeight(us int) Weight {
	switch {
	case us >= 700:
		return WeightBold
	case us >= 600:
		return WeightSemibold
	case us >= 500:
		return WeightMedium
	}
	return WeightRegular
}

// weightFromName reads a subfamily string, for the faces with no OS/2.
func weightFromName(sub string) Weight {
	s := strings.ToLower(sub)
	switch {
	case strings.Contains(s, "black"), strings.Contains(s, "heavy"),
		strings.Contains(s, "bold"):
		return WeightBold
	case strings.Contains(s, "semibold"), strings.Contains(s, "demibold"):
		return WeightSemibold
	case strings.Contains(s, "medium"):
		return WeightMedium
	}
	return WeightRegular
}

func slicesHas(xs []string, x string) bool {
	for _, s := range xs {
		if strings.EqualFold(s, x) {
			return true
		}
	}
	return false
}

func be16(b []byte) uint16 { return uint16(b[0])<<8 | uint16(b[1]) }

// scanFamilies indexes the faces found by walking dirs, and returns
// the family names a chooser should list.
func scanFamilies(dirs []string, into map[string][]sysFace) []string {
	return indexFaceRecords(scanFontDirs(dirs), into)
}

// sortedFamilies is the chooser's list: the primary name of each
// family, deduplicated, ordered so the same set always reads the same.
func sortedFamilies(primary map[string]string) []string {
	names := make([]string, 0, len(primary))
	for _, fam := range primary {
		names = append(names, fam)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := strings.ToLower(names[i]), strings.ToLower(names[j])
		if a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	return names
}
