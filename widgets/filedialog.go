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
	// FileOpenFolder picks a directory. The listing shows folders only,
	// activating one enters it, and the button returns the one that is
	// selected — or, with nothing selected, the folder being shown, which
	// is what every folder chooser does.
	FileOpenFolder
)

// picksFolder reports whether this mode returns a directory.
func (m FileDialogMode) picksFolder() bool { return m == FileOpenFolder }

// FileDialogOptions configures ShowFileDialog. Entries and OnNavigate feed
// the toolkit's own dialog.
type FileDialogOptions struct {
	Title string
	// Path is the folder to open in. For [FileSave] it may be a whole
	// file path instead, in which case its directory is listed and its
	// base name is suggested — see Name.
	Path string
	// Name is the file name a [FileSave] dialog starts with ("Invoice.eml").
	// Path stays the folder. Without it, a Path that names a file rather
	// than a directory is split into the two, so a caller that has only
	// one string to give still gets a listing rather than an empty one.
	Name       string
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
	body    *FlexBox
	overlay *Overlay
	// closeWindow closes the window a chooser too big for its host was put
	// in ([DialogWindowHost]); nil for the ordinary overlay.
	closeWindow func()
}

// minCardW and minCardH are the smallest a chooser's card may be, in 1x
// design pixels: a path field, a listing worth reading, and the buttons.
const minCardW, minCardH = 520, 380

// NewFileDialog builds the overlay + card. Call Show, or use ShowFileDialog.
func NewFileDialog(opts FileDialogOptions) *FileDialog {
	if opts.Title == "" {
		switch opts.Mode {
		case FileSave:
			opts.Title = "Save file"
		case FileOpenFolder:
			opts.Title = "Choose folder"
		default:
			opts.Title = "Open file"
		}
	}
	if opts.Path == "" {
		opts.Path = "."
	}
	opts.Path, opts.Name = splitSavePath(opts)
	fd := &FileDialog{opts: opts}
	fd.Init(fd)

	fd.dir = opts.Path
	start := opts.Path
	if opts.Name != "" {
		start = filepath.Join(opts.Path, opts.Name)
	}
	fd.path = NewTextField(start, "Path", nil)
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
	switch opts.Mode {
	case FileSave:
		openLbl = "Save"
	case FileOpenFolder:
		openLbl = "Choose"
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

	fd.body = browse
	card := NewPanel(opts.Title, browse)
	card.Raised = true
	fd.overlay = NewOverlay(card)
	fd.overlay.MinCardW = minCardW
	fd.overlay.MinCardH = minCardH
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
	if fd.opts.Mode.picksFolder() {
		// Not the glob: a folder chooser ignores it, and printing it
		// would claim a filter that is not being applied.
		return "Folders only"
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
	keep := func(ents []FileInfo) []FileInfo {
		if fd.opts.Mode.picksFolder() {
			return foldersOnly(ents)
		}
		return filterEntries(ents, fd.opts.Filter)
	}
	if fd.opts.OnNavigate != nil {
		if ents := fd.opts.OnNavigate(p); ents != nil {
			return keep(ents), nil
		}
	}
	ents, err := ReadDirEntries(p)
	if err == nil {
		return keep(ents), nil
	}
	if len(fd.opts.Entries) > 0 {
		// Explicit caller-supplied listing (the documented stub seam).
		return keep(append([]FileInfo(nil), fd.opts.Entries...)), nil
	}
	return nil, err
}

// foldersOnly is the filter a folder chooser uses instead of the glob:
// a file is not an answer to "which folder", and showing files that
// cannot be picked is how a chooser gets accused of ignoring clicks.
func foldersOnly(ents []FileInfo) []FileInfo {
	out := make([]FileInfo, 0, len(ents))
	for _, e := range ents {
		if e.Dir {
			out = append(out, e)
		}
	}
	return out
}

// splitSavePath turns a Path that names a file into a folder and a
// suggested name, for the callers that have one string to give.
//
// It is deliberately timid: only for Save, only when no Name was passed,
// and only when the filesystem agrees — the parent lists and the path
// itself does not. A stubbed dialog (Entries, OnNavigate) and a path
// that is simply wrong are both left exactly as they were, because
// guessing there would move a listing the caller chose.
func splitSavePath(opts FileDialogOptions) (dir, name string) {
	dir, name = opts.Path, opts.Name
	if opts.Mode != FileSave || name != "" || opts.Path == "" {
		return
	}
	if opts.Entries != nil || opts.OnNavigate != nil {
		return
	}
	if st, err := os.Stat(opts.Path); err == nil && st.IsDir() {
		return
	}
	parent := filepath.Dir(opts.Path)
	base := filepath.Base(opts.Path)
	if parent == opts.Path || base == "." || base == string(filepath.Separator) {
		return
	}
	if st, err := os.Stat(parent); err != nil || !st.IsDir() {
		return
	}
	return parent, base
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
	if fd.closeWindow != nil {
		// A chooser in a window of its own: the window goes with it, and
		// once, however the dialog ended — a button, the window's own close
		// control, or the application calling it off.
		cl := fd.closeWindow
		fd.closeWindow = nil
		cl()
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

// Cancel ends the chooser as its Cancel button does: OnCancel runs, the
// overlay or the window it is in goes, and a second call does nothing.
//
// It is what an application calls to take the dialog back — a vault that has
// locked, a window that is closing — and what the window's own close control
// runs when the chooser is in a window of its own.
func (fd *FileDialog) Cancel() { fd.finish(false) }

// chosen is the path the dialog would return: **what the path field
// says**, then the selected row, then the directory.
//
// The field comes first because it is the only one of the three the user
// can disagree with. Selecting a row writes that row's full path into
// the field (see onSelect), so for a plain selection the two agree and
// the order does not matter. It matters when they differ, and they
// differ exactly when the user typed: select old.txt in a Save dialog,
// type new.txt over it, press Save. Reading the row there returns
// old.txt — the file the user was careful *not* to name — and the caller
// overwrites it.
func (fd *FileDialog) chosen() string {
	if fd.path != nil && fd.path.Text != "" {
		return fd.path.Text
	}
	if fd.table != nil && fd.table.Selected >= 0 && fd.table.Selected < len(fd.entries) {
		e := fd.entries[fd.table.Selected]
		return filepath.Join(fd.dir, e.Name)
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
	if fd.showInAWindowOfItsOwn(from) {
		return fd
	}
	fd.Show(from)
	return fd
}

// DialogWindowHost is a host that can open a window of its own to put a
// dialog in. [app.Window] implements it.
//
// It is for the dialog that does not fit in the window asking for it. An
// overlay is held to its host — 92% of its width and 88% of its height, so a
// modal cannot cover the window it belongs to — and a skinned player's window
// is 275 by 116 pixels. A file chooser needs 520 by 380 before its listing has
// a row in it, so the one shown there had a table of no height at all and its
// buttons below the window's own foot: not a smaller chooser, an unusable one.
// The same fallback is what a window gets when the desktop's own chooser is
// not there to ask.
//
// A host that has no windows to open — an offscreen test host, a single-window
// shell — leaves this out, and the dialog is an overlay as it always was.
type DialogWindowHost interface {
	// OpenDialogWindow puts c in a window of its own, at least w by h
	// logical pixels, titled title and belonging to this one, and returns
	// the function that closes it. cancel is what the window's own close
	// control runs — the dialog's Cancel, so a window closed from its frame
	// answers the caller rather than leaving it waiting. False where the
	// host could not open a window.
	OpenDialogWindow(title string, w, h int, c widget.Component, cancel func()) (close func(), ok bool)
}

// dialogWindowSize is what a chooser asks for when it gets a window: enough
// for the card's own minimum with room for a listing worth reading, which is
// the whole reason it is not an overlay.
const dialogWindowW, dialogWindowH = 680, 500

// showInAWindowOfItsOwn opens the chooser in a window when the one asking is
// too small to hold it, and reports whether it did.
func (fd *FileDialog) showInAWindowOfItsOwn(from widget.Component) bool {
	if from == nil || fd.overlay == nil || fd.overlay.Card == nil {
		return false
	}
	h, ok := from.Host().(DialogWindowHost)
	if !ok || overlayFits(from, fd.overlay) {
		return false
	}
	// The body, not the card: the card is a panel with a caption of its
	// own, which is what an overlay floating inside a window needs and
	// exactly what a window with a title bar of its own does not. Out of
	// the card first, because a component has one parent.
	card, body := fd.overlay.Card, fd.body
	card.Remove(body)
	fd.overlay = nil
	close, ok := h.OpenDialogWindow(fd.opts.Title, dialogWindowW, dialogWindowH, body, func() { fd.finish(false) })
	if !ok {
		// Put it back and let the caller show the overlay after all.
		card.Add(body)
		fd.overlay = NewOverlay(card)
		fd.overlay.MinCardW, fd.overlay.MinCardH = minCardW, minCardH
		fd.overlay.OnClose = func() { fd.finish(false) }
		return false
	}
	fd.closeWindow = close
	return true
}

// overlayFits reports whether the window hosting from has room for o's card
// at the size it asks for. The caps are Overlay.Arrange's own.
func overlayFits(from widget.Component, o *Overlay) bool {
	wr, ok := from.Host().(widget.WindowRecter)
	if !ok {
		return true
	}
	box := wr.WindowRect()
	if box.Empty() {
		return true
	}
	lk := from.Look()
	return style.Dip(lk, o.MinCardW) <= box.Dx()*0.92 && style.Dip(lk, o.MinCardH) <= box.Dy()*0.88
}

// openNative is the portal call (a seam for tests).
var openNative = platform.OpenFileChooser

func showNativeFileDialog(from widget.Component, opts FileDialogOptions) bool {
	timers, ok := from.Host().(widget.Timers)
	if !ok || timers == nil {
		return false
	}
	co := platform.FileChooserOptions{
		Title:     opts.Title,
		Save:      opts.Mode == FileSave,
		Directory: opts.Mode.picksFolder(),
	}
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
	// An explicit Name wins the split above: the caller said which part
	// of Path is the folder by not putting the name in it.
	if opts.Name != "" {
		co.Name = opts.Name
		if st, err := os.Stat(opts.Path); err == nil && st.IsDir() {
			co.Folder, _ = filepath.Abs(opts.Path)
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
