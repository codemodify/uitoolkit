package uitoolkit_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
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
