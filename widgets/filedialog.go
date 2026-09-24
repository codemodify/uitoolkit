package widgets

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
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

// FileDialogOptions configures ShowFileDialog. Entries and OnNavigate feed
// the toolkit's own dialog.
type FileDialogOptions struct {
	Title      string
	Path       string
	Filter     string // glob patterns, separated by spaces or ';' ("*.txt *.md")
	Mode       FileDialogMode
	Entries    []FileInfo
	OnNavigate func(path string) []FileInfo
	OnPick     func(path string)
	OnCancel   func()
	// Native shows the desktop's own file dialog (KDE's, GNOME's) through
	// the XDG portal, as Qt and GTK apps do, instead of the toolkit's
	// themed one; UITK_NATIVE_DIALOGS=1 turns it on for every dialog.
	// Without a portal the toolkit's dialog shows.
	Native bool
}

// NativeDialogsEnv set to 1 makes every file dialog the desktop's own.
const NativeDialogsEnv = "UITK_NATIVE_DIALOGS"

// FileDialog is a modal list + path field. It does not open a platform dialog.
type FileDialog struct {
	widget.Base
	opts    FileDialogOptions
	path    *TextField
	table   *TableView
	dir     string
	entries []FileInfo
	picked  string
	err     string
	done    bool
	hint    *Label
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
	}, 0, fd.cell, fd.onSelect)
	// Selection only moves the path field. Entering a folder is an
	// activation (Return or double click), so arrowing through the list no
	// longer navigates on every keystroke.
	fd.table.OnActivate = fd.onActivate
	fd.table.OnSort = fd.sortEntries
	fd.table.RowHeight = 26

	fd.hint = NewLabel("")

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
		fd.hint,
		fd.table,
		NewButtonBox().AddButton(cancel, RoleReject).AddButton(open, RoleAccept),
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

// hintText is the status line: the read error when the listing failed,
// otherwise the active filter.
func (fd *FileDialog) hintText() string {
	if fd.err != "" {
		return fd.err
	}
	if fd.opts.Filter != "" {
		return "Filter  " + fd.opts.Filter
	}
	return ""
}

// Error is the last directory read failure, or empty.
func (fd *FileDialog) Error() string { return fd.err }

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

// onSelect only mirrors the highlighted row into the path field.
func (fd *FileDialog) onSelect(i int) {
	if i < 0 || i >= len(fd.entries) || fd.path == nil {
		return
	}
	fd.path.SetText(filepath.Join(fd.dir, fd.entries[i].Name))
}

// onActivate enters a folder, or accepts a file (Return / double click).
func (fd *FileDialog) onActivate(i int) {
	if i < 0 || i >= len(fd.entries) {
		return
	}
	e := fd.entries[i]
	next := filepath.Join(fd.dir, e.Name)
	if e.Dir {
		fd.setPath(next)
		return
	}
	if fd.path != nil {
		fd.path.SetText(next)
	}
	fd.finish(true)
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
	ents, err := fd.list(p)
	fd.entries = ents
	fd.err = ""
	if err != nil {
		fd.err = "Cannot open " + p + ": " + err.Error()
	}
	if fd.hint != nil {
		fd.hint.SetText(fd.hintText())
	}
	fd.table.RowCount = len(fd.entries)
	fd.table.Selected = -1
	fd.table.OffsetY = 0
	fd.table.Invalidate()
}

// list reads a directory through the configured seam. A failed read reports
// the error so the dialog can show it — it must never fall back to a
// fabricated listing, which produced paths for files that do not exist.
func (fd *FileDialog) list(p string) ([]FileInfo, error) {
	if fd.opts.OnNavigate != nil {
		if ents := fd.opts.OnNavigate(p); ents != nil {
			return filterEntries(ents, fd.opts.Filter), nil
		}
	}
	ents, err := ReadDirEntries(p)
	if err == nil {
		return filterEntries(ents, fd.opts.Filter), nil
	}
	if len(fd.opts.Entries) > 0 {
		// Explicit caller-supplied listing (the documented stub seam).
		return filterEntries(append([]FileInfo(nil), fd.opts.Entries...), fd.opts.Filter), nil
	}
	return nil, err
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

// ShowFileDialog shows a file dialog for the window that hosts from: the
// toolkit's themed one, or the desktop's own (Native, UITK_NATIVE_DIALOGS)
// when the portal answers, in which case it returns nil and OnPick or
// OnCancel run on the UI goroutine when the user is done.
func ShowFileDialog(from widget.Component, opts FileDialogOptions) *FileDialog {
	if (opts.Native || style.NativeDialogs() || os.Getenv(NativeDialogsEnv) == "1") && showNativeFileDialog(from, opts) {
		return nil
	}
	fd := NewFileDialog(opts)
	fd.Show(from)
	return fd
}

// openNative is the portal call (a seam for tests).
var openNative = platform.OpenFileChooser

func showNativeFileDialog(from widget.Component, opts FileDialogOptions) bool {
	timers, ok := from.Host().(widget.Timers)
	if !ok || timers == nil {
		return false
	}
	co := platform.FileChooserOptions{Title: opts.Title, Save: opts.Mode == FileSave}
	if p, ok := from.Host().(interface{ PortalParent() string }); ok {
		// The dialog opens as the window's child (modal to it).
		co.ParentWindow = p.PortalParent()
	}
	if opts.Path != "" {
		if st, err := os.Stat(opts.Path); err == nil && st.IsDir() {
			co.Folder, _ = filepath.Abs(opts.Path)
		} else if abs, err := filepath.Abs(opts.Path); err == nil {
			co.Folder, co.Name = filepath.Dir(abs), filepath.Base(abs)
		}
	}
	if pats := strings.FieldsFunc(opts.Filter, func(r rune) bool { return r == ' ' || r == ';' || r == ',' }); len(pats) > 0 {
		co.Filters = []platform.FileFilter{{Name: strings.Join(pats, " "), Patterns: pats}}
	}
	return openNative(co, func(paths []string) {
		// Back on the UI goroutine through the window's timers.
		timers.AfterFunc(0, func() {
			if len(paths) == 0 {
				if opts.OnCancel != nil {
					opts.OnCancel()
				}
				return
			}
			if opts.OnPick != nil {
				opts.OnPick(paths[0])
			}
		})
	})
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

// stubEntries is a sample listing for demos and tests. It is never used as a
// silent fallback for a failed directory read.
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
//
// In device pixels, like every other Measure: the path field, the table
// and the button row inside are laid out at the look's scale, so a card
// asked for in design pixels is too small for its own contents on a
// scaled desktop — the title clips, the table runs past the right edge
// and the buttons crowd the bottom.
func (fd *FileDialog) Measure(c layout.Constraints) paintengine2d.Point {
	lk := fd.Look()
	return c.Constrain(paintengine2d.Pt(style.Dip(lk, 520), style.Dip(lk, 420)))
}
