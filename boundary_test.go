package uitoolkit_test

import (
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
