package filesapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// Dragging in Files: rows drag out as real files, and files dropped on a
// folder in the tree or on a folder tab move there.
//
// The demo's folders are made up, so a row has no file behind it until
// something asks for one. The first drag writes the rows out to a
// directory of the process's own — a file manager on the other end of the
// drag has to be handed a path that is really there, or the drop does
// nothing and it is impossible to tell a broken protocol from a broken
// demo.

// filesScratch is where dragged rows are written, made once per process
// and removed when it exits.
var filesScratch struct {
	sync.Once
	dir string
}

func filesScratchDir() string {
	filesScratch.Do(func() {
		dir, err := os.MkdirTemp("", "uitk-files-demo-")
		if err != nil {
			return
		}
		filesScratch.dir = dir
	})
	return filesScratch.dir
}

// filesMaterialize writes one row out and reports the path it is at. A
// folder becomes a directory with its files in it, so dragging one to a
// file manager copies something recognisable.
func filesMaterialize(places []filePlace, place int, r fileRow) (string, bool) {
	dir := filesScratchDir()
	if dir == "" {
		return "", false
	}
	// Under the place's own name, so two folders with a README each do
	// not write over one another.
	base := filepath.Join(dir, strings.TrimPrefix(places[place].Path, "/"))
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", false
	}
	full := filepath.Join(base, r.Name)
	if !r.Dir {
		if err := os.WriteFile(full, []byte(r.Body), 0o644); err != nil {
			return "", false
		}
		return full, true
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return "", false
	}
	// A folder that is itself one of the places gets that place's files.
	for _, p := range places {
		if p.Label != r.Name {
			continue
		}
		for _, child := range p.Rows {
			if child.Dir {
				continue
			}
			_ = os.WriteFile(filepath.Join(full, child.Name), []byte(child.Body), 0o644)
		}
	}
	return full, true
}

// filesDragRows is the drag of the selected rows: real paths, so a file
// manager or a mail client on the other end gets files it can open, and
// the rows themselves as the payload, so a drop back into this same
// application needs no files at all.
func filesDragRows(win *app.Window, places []filePlace, place int, rows []int) *widget.Drag {
	if place < 0 || place >= len(places) {
		return nil
	}
	all := places[place].Rows
	var paths []string
	var names []string
	for _, i := range rows {
		if i < 0 || i >= len(all) {
			continue
		}
		if p, ok := filesMaterialize(places, place, all[i]); ok {
			paths = append(paths, p)
		}
		names = append(names, all[i].Name)
	}
	if len(paths) == 0 {
		return nil
	}
	d := widget.DragFiles(paths...)
	if d == nil {
		return nil
	}
	d.Payload = filesPayload{From: place, Names: names}
	// Files move between folders and copy out to other applications; the
	// target picks, and a file manager offers the user the choice.
	d.Actions = platform.DragCopy | platform.DragMove | platform.DragLink
	d.Preferred = platform.DragCopy
	if win != nil {
		d.Image, d.Hotspot = widget.DragLabel(win.Look(), filesDragLabel(names), win.Scale())
	}
	return d
}

// filesPayload rides along inside the application: which folder the rows
// came from and which rows they were, so a drop in another folder can
// move them without reading a single byte back.
type filesPayload struct {
	From  int
	Names []string
}

// filesDragLabel is what the drag's picture says: the one name, or how
// many there are.
func filesDragLabel(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return fmt.Sprintf("%d items", len(names))
}

// filesDropInto moves the dragged rows into place dst. It reports whether
// anything moved, which is what tells the drag's source the move ran.
func filesDropInto(places []filePlace, dst int, e widget.DropEvent) bool {
	if dst < 0 || dst >= len(places) {
		return false
	}
	p, ok := e.Payload.(filesPayload)
	if !ok {
		// From another application: the files are named by path, and the
		// demo lists them in the folder without copying anything.
		return filesDropPaths(places, dst, e.Paths)
	}
	if p.From == dst {
		return false // already there
	}
	src := &places[p.From]
	moved := 0
	for _, name := range p.Names {
		for i, r := range src.Rows {
			if r.Name != name {
				continue
			}
			if e.Action == platform.DragMove {
				src.Rows = append(src.Rows[:i:i], src.Rows[i+1:]...)
			}
			places[dst].Rows = append(places[dst].Rows, r)
			moved++
			break
		}
	}
	return moved > 0
}

// filesDropPaths lists files dropped in from another application.
func filesDropPaths(places []filePlace, dst int, paths []string) bool {
	added := 0
	for _, p := range paths {
		name := filepath.Base(p)
		if name == "" || name == "." || name == "/" {
			continue
		}
		row := fileRow{Name: name, Kind: filesKindOf(name), Size: "—", Modified: "now"}
		if fi, err := os.Stat(p); err == nil {
			if fi.IsDir() {
				row.Dir, row.Kind = true, "Folder"
			} else {
				row.Size = fmt.Sprintf("%d B", fi.Size())
			}
		}
		if !row.Dir {
			if b, err := os.ReadFile(p); err == nil && len(b) < 64*1024 {
				row.Body = string(b)
			}
		}
		places[dst].Rows = append(places[dst].Rows, row)
		added++
	}
	return added > 0
}

// filesKindOf names a dropped file's kind from its extension, the way the
// sample rows are named.
func filesKindOf(name string) string {
	switch ext := strings.ToLower(filepath.Ext(name)); ext {
	case "":
		return "File"
	case ".go", ".c", ".h", ".py", ".sh", ".js", ".ts":
		return "Source"
	case ".md", ".txt":
		return "Text"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return "Image"
	default:
		return strings.ToUpper(strings.TrimPrefix(ext, ".")) + " file"
	}
}

// filesDropMessage is what the status bar says about a drop: whether the
// files moved or were copied, and where they landed.
func filesDropMessage(e widget.DropEvent, where string) string {
	n := len(e.Paths)
	if p, ok := e.Payload.(filesPayload); ok {
		n = len(p.Names)
	}
	verb := "Copied"
	if e.Action == platform.DragMove {
		verb = "Moved"
	}
	return fmt.Sprintf("%s %d item(s) to %s", verb, n, where)
}
