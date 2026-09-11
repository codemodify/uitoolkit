package widgets

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// FileInfo is one row in a FileDialog listing.
type FileInfo struct {
	Name string
	Dir  bool
	Size int64
}

// FileDialogMode selects Open vs Save chrome.
type FileDialogMode int

const (
	FileOpen FileDialogMode = iota
	FileSave
)

// FileDialogOptions configures ShowFileDialog. Entries and OnNavigate are the
// stub seam — a later backend can swap this for a native picker.
type FileDialogOptions struct {
	Title      string
	Path       string
	Filter     string
	Mode       FileDialogMode
	Entries    []FileInfo
	OnNavigate func(path string) []FileInfo
	OnPick     func(path string)
	OnCancel   func()
}

// FileDialog is a modal list + path field. It does not open a platform dialog.
type FileDialog struct {
	widget.Base
	opts    FileDialogOptions
	path    *TextField
	table   *TableView
	dir     string
	entries []FileInfo
	picked  string
	done    bool
	overlay *Overlay
}

// NewFileDialog builds the overlay + card. Call Show, or use ShowFileDialog.
func NewFileDialog(opts FileDialogOptions) *FileDialog {
	if opts.Title == "" {
		if opts.Mode == FileSave {
			opts.Title = "Save file"
		} else {
			opts.Title = "Open file"
		}
	}
	if opts.Path == "" {
		opts.Path = "."
	}
	fd := &FileDialog{opts: opts}
	fd.Init(fd)

	fd.dir = opts.Path
	fd.path = NewTextField(opts.Path, "Path", nil)
	fd.path.OnSubmit = func(s string) { fd.setPath(s) }

	fd.table = NewTableView([]TableColumn{
		{Title: "Name", Width: 0, Sortable: true},
		{Title: "Kind", Width: 88, Sortable: true},
	}, 0, fd.cell, fd.onRow)
	fd.table.OnSort = fd.sortEntries
	fd.table.RowHeight = 26

	hint := NewLabel("Stub picker — wire OnNavigate or Entries; native dialog later.")
	if opts.Filter != "" {
		hint.SetText("Filter  " + opts.Filter + "  ·  stub picker (no platform dialog)")
	}

	openLbl := "Open"
	if opts.Mode == FileSave {
		openLbl = "Save"
	}
	open := NewButton(openLbl, func() { fd.finish(true) })
	open.Primary = true
	cancel := NewButton("Cancel", func() { fd.finish(false) })

	browse := NewColumn(
		NewLabel("Path"),
		fd.path,
		hint,
		fd.table,
		NewRow(cancel, open).WithGap(8).WithJustify(layout.JustifyEnd),
	).WithGap(8).WithPad(4)
	browse.AddFlex(fd.table, 1)

	card := NewPanel(opts.Title, browse)
	card.Raised = true
	fd.overlay = NewOverlay(card)
	fd.overlay.MinCardW = 520
	fd.overlay.MinCardH = 380
	fd.overlay.OnClose = func() { fd.finish(false) }
	fd.reload(opts.Path)
	return fd
}

func (fd *FileDialog) cell(row, col int) string {
	if row < 0 || row >= len(fd.entries) {
		return ""
	}
	e := fd.entries[row]
	if col == 1 {
		if e.Dir {
			return "Folder"
		}
		return "File"
	}
	if e.Dir {
		return e.Name + "/"
	}
	return e.Name
}

func (fd *FileDialog) onRow(i int) {
	if i < 0 || i >= len(fd.entries) {
		return
	}
	e := fd.entries[i]
	next := filepath.Join(fd.dir, e.Name)
	if e.Dir {
		fd.setPath(next)
		return
	}
	fd.path.SetText(next)
}

func (fd *FileDialog) setPath(p string) {
	if p == "" {
		return
	}
	fd.dir = p
	if fd.path != nil {
		fd.path.SetText(p)
	}
	fd.reload(p)
}

func (fd *FileDialog) reload(p string) {
	fd.entries = fd.list(p)
	fd.table.RowCount = len(fd.entries)
	fd.table.Selected = -1
	fd.table.OffsetY = 0
	fd.table.Invalidate()
}

func (fd *FileDialog) list(p string) []FileInfo {
	if fd.opts.OnNavigate != nil {
		if ents := fd.opts.OnNavigate(p); ents != nil {
			return filterEntries(ents, fd.opts.Filter)
		}
	}
	if ents, err := ReadDirEntries(p); err == nil {
		return filterEntries(ents, fd.opts.Filter)
	}
	if len(fd.opts.Entries) > 0 {
		return filterEntries(append([]FileInfo(nil), fd.opts.Entries...), fd.opts.Filter)
	}
	return filterEntries(stubEntries(), fd.opts.Filter)
}

func (fd *FileDialog) sortEntries(col int, asc bool) {
	sort.SliceStable(fd.entries, func(i, j int) bool {
		a, b := fd.entries[i], fd.entries[j]
		var less bool
		if col == 1 {
			if a.Dir != b.Dir {
				less = a.Dir
			} else {
				less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		} else {
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if !asc {
			return !less
		}
		return less
	})
	fd.table.Invalidate()
}

func (fd *FileDialog) finish(ok bool) {
	if fd.done {
		return
	}
	fd.done = true
	if ok {
		fd.picked = fd.chosen()
	}
	if fd.overlay != nil {
		widget.DismissOverlay(fd.overlay)
	}
	if ok {
		if fd.opts.OnPick != nil {
			fd.opts.OnPick(fd.picked)
		}
		return
	}
	if fd.opts.OnCancel != nil {
		fd.opts.OnCancel()
	}
}

func (fd *FileDialog) chosen() string {
	if fd.table != nil && fd.table.Selected >= 0 && fd.table.Selected < len(fd.entries) {
		e := fd.entries[fd.table.Selected]
		return filepath.Join(fd.dir, e.Name)
	}
	if fd.path != nil && fd.path.Text != "" {
		return fd.path.Text
	}
	return fd.dir
}

// Overlay is the dimmed host layer.
func (fd *FileDialog) Overlay() *Overlay { return fd.overlay }

// Path is the last confirmed path (empty if cancelled).
func (fd *FileDialog) Path() string { return fd.picked }

// Entries is the visible listing.
func (fd *FileDialog) Entries() []FileInfo { return fd.entries }

// Show mounts the modal on the window that hosts from.
func (fd *FileDialog) Show(from widget.Component) bool {
	if fd.overlay == nil {
		return false
	}
	return widget.ShowOverlay(from, fd.overlay)
}

// ShowFileDialog mounts a stub file picker on the window that hosts from.
func ShowFileDialog(from widget.Component, opts FileDialogOptions) *FileDialog {
	fd := NewFileDialog(opts)
	fd.Show(from)
	return fd
}

// ReadDirEntries lists a real directory for apps that want to wire the stub.
func ReadDirEntries(path string) ([]FileInfo, error) {
	ents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]FileInfo, 0, len(ents))
	for _, e := range ents {
		info := FileInfo{Name: e.Name(), Dir: e.IsDir()}
		if fi, err := e.Info(); err == nil && !e.IsDir() {
			info.Size = fi.Size()
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func stubEntries() []FileInfo {
	return []FileInfo{
		{Name: "docs", Dir: true},
		{Name: "examples", Dir: true},
		{Name: "widgets", Dir: true},
		{Name: "LICENSE"},
		{Name: "README.md"},
		{Name: "export.go"},
		{Name: "go.mod"},
		{Name: "version.go"},
	}
}

func filterEntries(ents []FileInfo, filter string) []FileInfo {
	filter = strings.TrimSpace(filter)
	if filter == "" || filter == "*" || filter == "*.*" {
		return ents
	}
	ext := strings.TrimPrefix(filter, "*")
	if !strings.HasPrefix(filter, "*.") {
		return ents
	}
	out := make([]FileInfo, 0, len(ents))
	for _, e := range ents {
		if e.Dir || strings.HasSuffix(strings.ToLower(e.Name), strings.ToLower(ext)) {
			out = append(out, e)
		}
	}
	return out
}

// Preferred size hint so the overlay card is wide enough for a table.
func (fd *FileDialog) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(520, 420))
}
