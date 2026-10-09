package uitoolkit_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The sample applications are the toolkit's first customer: what they can
// build against is exactly what somebody who runs `go get` can build
// against. A sample that reaches into internal/ can do something no
// reader of it can, so it stops being an example and becomes a lie — and
// the missing piece of public API it papers over never gets built.
//
// The rule has been broken twice, so it is a test rather than a memory.
// If this fails, do not add the directory to the list below: either use
// the public API, or make the thing the sample needed public.
var publicOnly = []string{
	"examples", // the samples a newcomer reads and copies
	"cmd",      // the shipped commands, minus the tooling exempted below
	"showcase", // the widget gallery, public because two apps want it
	"tools",    // the helper programs beside the shell scripts
}

// Toolkit tooling may use the toolkit's own test infrastructure: these
// two commands are not samples and nobody is meant to copy them out.
// uitest-driver is the end-to-end rig's other half and drives a window
// from the outside with internal/uitest and internal/apptest;
// uitk-themesheet renders the theme atlas with internal/themesheet.
// Nothing else goes on this list.
var toolingAllowed = map[string]bool{
	"cmd/uitest-driver":   true,
	"cmd/uitk-themesheet": true,
}

func TestSamplesUseOnlyThePublicAPI(t *testing.T) {
	const internalPrefix = "github.com/codemodify/uitoolkit/internal/"
	bad := map[string][]string{}
	for _, root := range publicOnly {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if toolingAllowed[filepath.ToSlash(path)] {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range f.Imports {
				p, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if strings.HasPrefix(p, internalPrefix) {
					bad[path] = append(bad[path], p)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(bad) == 0 {
		return
	}
	files := make([]string, 0, len(bad))
	for f := range bad {
		files = append(files, f)
	}
	sort.Strings(files)
	var msg strings.Builder
	msg.WriteString("these reach into the toolkit's internal packages, which nobody outside the module can import:\n")
	for _, f := range files {
		msg.WriteString("\t" + f + ": " + strings.Join(bad[f], ", ") + "\n")
	}
	msg.WriteString("A sample must build on the published API. If it needs something that is not there, make that thing public.")
	t.Error(msg.String())
}

// A sample's window carries the toolkit's name, because a desktop with
// several of them open has to say which toolkit they belong to — and
// because the title is what the panel, the task switcher and the window
// list call the window, which is the one place the look's own caption
// cannot stand in for it.
//
// Five of the look-alike samples set the title to their own binary name,
// so an Alt-Tab across a desktop of samples read
// "uitoolkit-sample-look-like-chromium". They were the five that draw
// their own caption, which is exactly why nobody noticed: the title
// never appears in the window, only outside it. The convention lived in
// a copied helper in twelve separate main packages and nothing held it,
// so this test holds it instead.
func TestEverySampleWindowIsNamedForTheToolkit(t *testing.T) {
	const prefix = "uitoolkit - "
	var bad []string
	err := filepath.WalkDir("examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		// A test may open a window called anything; it is not what a
		// reader copies and no desktop shows it.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isWindowOptions(lit.Type) {
				return true
			}
			for _, el := range lit.Elts {
				kv, ok := el.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := kv.Key.(*ast.Ident); !ok || key.Name != "Title" {
					continue
				}
				if titleIsNamedForTheToolkit(kv.Value, prefix) {
					continue
				}
				pos := fset.Position(kv.Pos())
				bad = append(bad, pos.Filename+":"+strconv.Itoa(pos.Line))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) == 0 {
		return
	}
	sort.Strings(bad)
	var msg strings.Builder
	msg.WriteString("these sample windows are not named for the toolkit:\n")
	for _, b := range bad {
		msg.WriteString("\t" + b + "\n")
	}
	msg.WriteString("A window title goes through the sample's windowTitle/WindowTitle helper, " +
		"which puts it under " + strconv.Quote(prefix) + ". The title is what the panel and the " +
		"task switcher show, so a sample that draws its own caption needs it too.")
	t.Error(msg.String())
}

// isWindowOptions reports whether t is the platform.WindowOptions type —
// the only struct here whose Title names a window. The samples' other
// Title fields are columns, tabs, wizard pages, tray items and
// notifications, which are named for what they are.
func isWindowOptions(t ast.Expr) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "WindowOptions" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "platform"
}

// titleIsNamedForTheToolkit accepts the prefix spelled out, or any call
// to the samples' helper — windowTitle in a main package, WindowTitle
// where an app package exports it. The helper is where the prefix is
// written once, so a title computed while the app runs carries it too;
// that is why a call is accepted without reading what it returns, and
// why anything else (a bare string, a variable) is not.
func titleIsNamedForTheToolkit(v ast.Expr, prefix string) bool {
	switch e := v.(type) {
	case *ast.BasicLit:
		s, err := strconv.Unquote(e.Value)
		return err == nil && strings.HasPrefix(s, prefix)
	case *ast.CallExpr:
		switch fn := e.Fun.(type) {
		case *ast.Ident:
			return fn.Name == "windowTitle" || fn.Name == "WindowTitle"
		case *ast.SelectorExpr:
			return fn.Sel.Name == "WindowTitle"
		}
	}
	return false
}
