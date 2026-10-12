package uitoolkit_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/codemodify/uitoolkit/style"
)

// The documentation names APIs; this checks they exist.
//
// `contracts.md` once listed the places an application may put an icon —
// `Button.Icon`, `MenuItem.Icon`, `TreeNode.Icon` — as the answer to the
// toolkit's own first rule, that a mark is an icon and never a character.
// A status bar was not on that list and could not be: no widget could
// draw a mark unless you clicked it. The rule was unkeepable and the page
// said nothing, because nothing checked the page against the toolkit.
//
// So every `package.Symbol` written in a documentation page has to
// resolve. A doc that names something that was renamed, or that was never
// built, fails here rather than wasting a reader's afternoon.
func TestEveryAPITheDocsNameExists(t *testing.T) {
	pkgs := map[string]map[string]bool{}
	for _, p := range []string{"app", "widget", "widgets", "style", "platform", "layout", "diag", "dock", "richtext", "."} {
		pkgs[pkgName(p)] = exportedOf(t, p)
	}
	// The root package is written as uitoolkit.X.
	pkgs["uitoolkit"] = pkgs["."]

	// pkg.Symbol, with the package one this repository actually has.
	ref := regexp.MustCompile(`\b(app|widget|widgets|style|platform|layout|diag|dock|richtext|uitoolkit)\.([A-Z][A-Za-z0-9_]*)\b`)

	files, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "README.md", "icons/README.md")

	var missing []string
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(src), "\n") {
			// A comparison table's other columns are other toolkits'
			// APIs, and two of fyne's packages are called `widget` and
			// `layout` exactly as ours are — `widget.Label` in a table
			// row is fyne's, not this repository's. There is no way to
			// tell them apart by spelling, so the ambiguous two are not
			// checked inside a table. Everything else is, everywhere.
			inTable := strings.HasPrefix(strings.TrimSpace(line), "|")
			for _, m := range ref.FindAllStringSubmatch(line, -1) {
				pkg, sym := m[1], m[2]
				if inTable && (pkg == "widget" || pkg == "layout") {
					continue
				}
				known, ok := pkgs[pkg]
				if !ok || known[sym] {
					continue
				}
				if docAllowed[pkg+"."+sym] {
					continue
				}
				missing = append(missing, f+": "+pkg+"."+sym)
			}
		}
	}
	sort.Strings(missing)
	seen := ""
	for _, m := range missing {
		if m == seen {
			continue
		}
		seen = m
		t.Errorf("the docs name something that does not exist: %s", m)
	}
}

// docAllowed is for names a page mentions that are deliberately not this
// repository's API: another toolkit's type in a comparison, a field of a
// struct rather than the struct, an example's own identifier. Each one is
// a decision, which is why they are written down rather than filtered by
// a pattern.
var docAllowed = map[string]bool{
	// Struct fields and methods, written package-qualified for the
	// reader's sake; the test only knows top-level names.
	"style.Classic.Pack":              true,
	"style.DecorationSpec.Stacked":    true,
	"style.Palette.Ink":               true,
	"style.DecorationSpec.ButtonSide": true,
	"platform.FrameCapsOf":            true,
	"widget.SetTheme":                 true,

	// Prose that names a function in order to say it does not exist:
	// platform.md explains that a client may not place a toplevel on
	// Wayland, so there is no CenterAvailable to ask — you ask the window
	// what happened instead. The sentence is right and the name must stay
	// unresolvable; that is its point.
	"platform.CenterAvailable": true,
}

func pkgName(dir string) string {
	if dir == "." {
		return "."
	}
	return dir
}

// exportedOf is every exported top-level name a package declares.
func exportedOf(t *testing.T, dir string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			continue // a file this build excludes; another will declare it
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					out[d.Name.Name] = true
				}
				if d.Recv != nil && d.Name.IsExported() {
					// Methods are named on their own in prose too.
					out[d.Name.Name] = true
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							out[s.Name.Name] = true
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								out[n.Name] = true
							}
						}
					}
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("%s declared nothing exported — the parser is not finding the package", dir)
	}
	return out
}

// **Every pack the toolkit ships has a row in the table that lists the
// packs.** The table is read as the list of what there is — `themes.md`
// opens by saying so — and it had fallen seven behind: Lumen's four,
// Nocturne, Cassette and Deck all shipped, were in Settings' theme list
// and in nobody's documentation. Nothing compared the page with the
// toolkit, so the page could only be right by somebody remembering.
//
// The counts in its first line are checked only in the build that has
// every engine, because they are counts of everything there is and a
// build with one engine has eight packs. `tools/test.sh` is that build.
func TestEveryPackIsInTheThemesTable(t *testing.T) {
	src, err := os.ReadFile("docs/themes.md")
	if err != nil {
		t.Fatal(err)
	}
	page := string(src)
	packs := style.AllBuiltinThemeNames()
	var missing []string
	for _, n := range packs {
		if !strings.Contains(page, "| `"+n+"` |") {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		t.Errorf("docs/themes.md lists no row for %d of the %d built-in packs: %s",
			len(missing), len(packs), strings.Join(missing, " "))
	}

	engines := style.EngineIDs()
	if len(engines) < 30 {
		t.Skipf("built with %d engines; the counts are only checked under theme_engine_all", len(engines))
	}
	want := fmt.Sprintf("%d packs from %d engines", len(packs), len(engines))
	if !strings.Contains(page, want) {
		first := page
		if i := strings.Index(page, "## Built-in packs"); i >= 0 {
			first = page[i:]
		}
		if i := strings.Index(first, "engines"); i >= 0 {
			first = first[:i+len("engines")]
		}
		t.Errorf("docs/themes.md does not say %q; it opens the table with %q", want,
			strings.TrimSpace(strings.TrimPrefix(first, "## Built-in packs")))
	}
}

// **A counted claim is counted again here.** Three places say how many
// packs leave a latched button looking like an ordinary one — two pages
// and the comment over the code that measures it — and the number drifts
// every time a pack is added. It had: the pages said 53 of 135 when the
// toolkit had 132 packs and 49 of them drew it the same way.
//
// Only checked in the build that has every engine, because it is a count
// of every pack there is.
func TestTheCountedClaimAboutLatchedButtonsIsTheCount(t *testing.T) {
	if n := len(style.EngineIDs()); n < 30 {
		t.Skipf("built with %d engines; the count is only checked under theme_engine_all", n)
	}
	packs := style.AllBuiltinThemeNames()
	same := 0
	for _, n := range packs {
		p, ok := style.LoadTheme(n)
		if !ok {
			t.Fatalf("%s is in the built-in list and does not load", n)
		}
		if !style.CheckedFaceOf(p.Look()) {
			same++
		}
	}

	claim := regexp.MustCompile(`(\d+) of the\s+(\d+) (?:shipped )?packs\s+(?://\s+)?draw a checked button`)
	for _, f := range []string{"docs/recipes.md", "docs/widgets.md", "style/checkedface.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		m := claim.FindStringSubmatch(strings.ReplaceAll(string(src), "\n", " "))
		if m == nil {
			t.Errorf("%s no longer makes the claim this counts; drop it from the list or put the sentence back", f)
			continue
		}
		if m[1] != itoa(same) || m[2] != itoa(len(packs)) {
			t.Errorf("%s says %s of %s packs draw a checked button like an ordinary one; it is %d of %d",
				f, m[1], m[2], same, len(packs))
		}
	}
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
