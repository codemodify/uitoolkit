package demo

import (
	"fmt"
	"path"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

type fileRow struct {
	Name, Kind, Size, Modified string
	Dir                        bool
	Body                       string
}

type filePlace struct {
	Label string
	Path  string
	Rows  []fileRow
}

// FilesApp is the Files / Projects dogfood: sidebar tree, table, toolbar,
// menus, preview TextArea (JetBrains Mono for code), tabs, dialogs, HiDPI metrics.
// Its folders open in tabs in the window's title bar (Dolphin's folder tabs
// in Chromium's place): the tree and the table show the selected tab's
// folder, "+" or Ctrl+T opens Home in a new tab, Ctrl+W closes one,
// Ctrl+Tab switches, a tab dragged along the strip moves, and a tab
// dragged out of the strip becomes a Files window of its own — dropped
// back on another window's strip it joins that one.
//
// Each tab keeps the folders it went through: Back and Forward on the tool
// bar, Alt+Left and Alt+Right, the mouse's back and forward buttons and a
// three-finger swipe across the touchpad go back and forward, as in
// Dolphin, Nautilus and every browser; a double-click on a folder in the
// listing opens it.
func FilesApp(win *app.Window) widget.Component { return filesApp(win, nil) }

// filesApp is FilesApp with the folders its window opens with: nil for
// the two a fresh window shows, one place for a window a torn-off tab
// made.
func filesApp(win *app.Window, open []int) widget.Component {
	places := samplePlaces()
	place := 1
	if len(open) > 0 && open[0] >= 0 && open[0] < len(places) {
		place = open[0]
	}
	if place >= len(places) {
		place = 0
	}
	sel := 0
	tabs := widgets.NewBrowserTabs()

	status := widgets.NewStatusBar("Ready.", places[0].Path, "v"+uitoolkit.Version)
	mark := func(msg string) { status.Set(0, msg) }

	preview := widgets.NewTextArea("", "Select a file", nil)
	preview.MinRows = 8
	preview.Wrap = false

	details := widgets.NewLabel("")
	pathLbl := widgets.NewLabel("")

	var table *widgets.TableView
	var tree *widgets.TreeView
	var backBtn, fwdBtn *widgets.ToolItem
	syncHistory := func() {}

	loadPreview := func() {
		if place < 0 || place >= len(places) {
			return
		}
		rows := places[place].Rows
		if sel < 0 || sel >= len(rows) {
			preview.SetText("")
			preview.Mono = false
			details.SetText("No selection")
			pathLbl.SetText(places[place].Path)
			return
		}
		r := rows[sel]
		full := path.Join(places[place].Path, r.Name)
		pathLbl.SetText(full)
		preview.SetText(r.Body)
		preview.Mono = isCodeName(r.Name)
		kind := r.Kind
		if r.Dir {
			kind = "Folder"
		}
		ui, mono := lookFaces(win)
		face := ui
		if isCodeName(r.Name) {
			face = mono
		}
		details.SetText(fmt.Sprintf("%s\n%s  ·  %s  ·  %s\n%s UI  ·  %s preview",
			r.Name, kind, r.Size, r.Modified, ui, face))
		status.Set(1, full)
	}

	refreshTable := func() {
		rows := places[place].Rows
		table.RowCount = len(rows)
		if sel >= len(rows) {
			sel = len(rows) - 1
		}
		if sel < 0 && len(rows) > 0 {
			sel = 0
		}
		if sel >= 0 {
			table.SetSelectedRows([]int{sel})
		} else {
			table.SetSelectedRows(nil)
		}
		loadPreview()
		status.Set(0, fmt.Sprintf("%d items", len(rows)))
	}

	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Sortable: true},
		{Title: "Kind", Width: 88, Sortable: true},
		{Title: "Size", Width: 72, Sortable: true, Align: style.AlignEnd},
		{Title: "Modified", Width: 110, Sortable: true},
	}, len(places[place].Rows), func(row, col int) string {
		rows := places[place].Rows
		if row < 0 || row >= len(rows) {
			return ""
		}
		r := rows[row]
		switch col {
		case 1:
			return r.Kind
		case 2:
			return r.Size
		case 3:
			return r.Modified
		default:
			return r.Name
		}
	}, func(i int) {
		sel = i
		loadPreview()
		mark("Selected " + places[place].Rows[i].Name)
	})
	// A file manager selects like Explorer and Finder: Ctrl, Shift, Ctrl+A;
	// typing a name jumps to it.
	table.Mode = widgets.SelectExtended
	table.OnSelectionChange = func(rows []int) {
		if len(rows) > 1 {
			mark(fmt.Sprintf("%d items selected", len(rows)))
		}
	}
	table.SetSelectedRows([]int{0})
	// A double-click (or Return) on a folder that is a place opens it.
	var openRow func(row int)
	table.OnActivate = func(row int) {
		if openRow != nil {
			openRow(row)
		}
	}
	table.OnSort = func(col int, asc bool) {
		rows := places[place].Rows
		sortFileRows(rows, col, asc)
		refreshTable()
	}
	// Dragging rows out hands another application real files (files_drag.go);
	// dropping them on a folder in the tree, or on a folder tab, moves them
	// there without anything leaving the process.
	table.OnDrag = func(rows []int) *widget.Drag {
		d := filesDragRows(win, places, place, rows)
		if d == nil {
			return nil
		}
		from := place
		d.Done = func(action platform.DragAction) {
			switch action {
			case platform.DragMove:
				if from == place {
					refreshTable()
				}
				mark(fmt.Sprintf("Moved %d item(s)", len(rows)))
			case platform.DragNone:
				mark("Drag cancelled")
			default:
				mark(fmt.Sprintf("Copied %d item(s)", len(rows)))
			}
		}
		return d
	}

	roots := make([]*widgets.TreeNode, 0, len(places))
	placeIndex := func(name string) int {
		for i, p := range places {
			if p.Label == name {
				return i
			}
		}
		return -1
	}
	// A place that is a folder of another shows under it, not again at the
	// top (projects/uitoolkit sits under Projects).
	nested := map[int]bool{}
	for _, p := range places {
		for _, r := range p.Rows {
			if j := placeIndex(r.Name); r.Dir && j >= 0 {
				nested[j] = true
			}
		}
	}
	nodeOf := map[int]*widgets.TreeNode{}
	for i, p := range places {
		if nested[i] {
			continue
		}
		n := widgets.NewTreeNode(p.Label)
		n.Data = i
		nodeOf[i] = n
		for _, r := range p.Rows {
			if r.Dir {
				child := widgets.NewTreeNode(r.Name)
				if j := placeIndex(r.Name); j >= 0 {
					child.Data = j
					nodeOf[j] = child
				} else {
					child.Data = i
				}
				n.Children = append(n.Children, child)
			}
		}
		n.Expanded = len(n.Children) > 0
		roots = append(roots, n)
	}
	tree = widgets.NewTreeView(roots...)
	tree.Sidebar = true // the places pane
	tree.SetAccessibleName("Places")
	if n := nodeOf[place]; n != nil {
		tree.Selected = n
	} else {
		tree.Selected = roots[0]
	}
	// showPlace shows place i in the table and the window title.
	showPlace := func(i int) {
		place = i
		sel = 0
		refreshTable()
		if win != nil {
			win.SetTitle(places[i].Label + " — Files")
		}
		syncHistory()
	}
	// visit shows place i in the selected tab, as a step in its history
	// (back returns to where it was); travel moves through the history.
	visit := func(i int) {
		t := tabs.Selected()
		if t < 0 {
			showPlace(i)
			return
		}
		ft := tabFolder(tabs, t)
		if ft.place != i {
			ft.back = append(ft.back, ft.place)
			ft.fwd = nil
			ft.place = i
		}
		tabs.SetTab(t, widgets.BrowserTab{Title: places[i].Label, Data: ft})
		if n := nodeOf[i]; n != nil {
			tree.Selected = n
			tree.Invalidate()
		}
		showPlace(i)
	}
	travel := func(forward bool) bool {
		t := tabs.Selected()
		if t < 0 {
			return false
		}
		ft := tabFolder(tabs, t)
		i, ok := ft.travel(forward)
		if !ok {
			return false
		}
		tabs.SetTab(t, widgets.BrowserTab{Title: places[i].Label, Data: ft})
		if n := nodeOf[i]; n != nil {
			tree.Selected = n
			tree.Invalidate()
		}
		showPlace(i)
		if forward {
			mark("Forward to " + places[i].Label)
		} else {
			mark("Back to " + places[i].Label)
		}
		return true
	}
	openRow = func(row int) {
		rows := places[place].Rows
		if row < 0 || row >= len(rows) {
			return
		}
		sel = row
		loadPreview()
		if j := placeIndex(rows[row].Name); rows[row].Dir && j >= 0 {
			visit(j)
			mark("Opened " + places[j].Label)
		}
	}
	// A folder in the places tree takes files: the drag's own rows move
	// into it, and files from another application are listed in it.
	// Files move between folders as well as copy into them: with the
	// desktop's modifier for a move (Shift on Plasma and GNOME) the drop
	// takes them out of the folder they came from.
	tree.DropActions = platform.DragCopy | platform.DragMove
	tree.OnDropNode = func(n *widgets.TreeNode, e widget.DropEvent) bool {
		i, ok := n.Data.(int)
		if !ok || !filesDropInto(places, i, e) {
			return false
		}
		refreshTable()
		mark(filesDropMessage(e, places[i].Label))
		return true
	}
	tree.OnSelect = func(n *widgets.TreeNode) {
		if n == nil {
			return
		}
		i, ok := n.Data.(int)
		if !ok || i < 0 || i >= len(places) {
			return
		}
		// The tree navigates the selected tab, as a browser's address bar does.
		visit(i)
		mark("Place  " + places[i].Label)
	}

	openStub := func() {
		widgets.ShowFileDialog(win.Content(), widgets.FileDialogOptions{
			Title:  "Open project",
			Path:   places[place].Path,
			Filter: "*.go;*.md",
			Entries: []widgets.FileInfo{
				{Name: "cmd", Dir: true},
				{Name: "internal", Dir: true},
				{Name: "main.go"},
				{Name: "README.md"},
			},
			OnPick: func(p string) { mark("Open " + p) },
		})
	}

	about := func() {
		ui, mono := lookFaces(win)
		widgets.Info(win.Content(), "About Files",
			"Files / Projects dogfood on uitoolkit.\nUI: "+ui+". Code preview: "+mono+".",
			func() { mark("About") })
	}

	remove := func() {
		if sel < 0 || sel >= len(places[place].Rows) {
			return
		}
		name := places[place].Rows[sel].Name
		widgets.Confirm(win.Content(), "Move to trash?",
			"Remove "+name+" from the listing (sample data only).",
			func(yes bool) {
				if !yes {
					mark("Kept " + name)
					return
				}
				places[place].Rows = append(places[place].Rows[:sel], places[place].Rows[sel+1:]...)
				refreshTable()
				mark("Trashed " + name)
			})
	}

	menubar := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("Open…", "Ctrl+O", openStub),
			widgets.ItemAccel("New folder", "Ctrl+N", func() { mark("New folder") }),
			widgets.Sep(),
			widgets.Item("About Files", about),
		),
		widgets.NewMenu("&Edit",
			widgets.ItemAccel("Find", "Ctrl+F", func() { mark("Find") }),
			widgets.Sep(),
			widgets.Item("Move to trash…", remove),
		),
		widgets.NewMenu("&View",
			widgets.Item("Refresh", func() { refreshTable(); mark("Refreshed") }),
		),
		widgets.NewMenu("&Help",
			widgets.Item("Keyboard", func() {
				widgets.Info(win.Content(), "Keyboard",
					"Arrows move the table. Return opens the stub picker from File.", nil)
			}),
		),
	)

	newBtn := widgets.ToolIconBtn(style.IconNew, "New", func() { mark("New folder") })
	newBtn.Tip = "New folder"
	openBtn := widgets.ToolIconBtn(style.IconOpen, "", openStub)
	openBtn.Tip = "Open (stub picker)"
	saveBtn := widgets.ToolIconBtn(style.IconSave, "", func() { mark("Saved listing") })
	saveBtn.Tip = "Save"
	searchBtn := widgets.ToolIconBtn(style.IconSearch, "", func() { mark("Search") })
	searchBtn.Tip = "Search"
	backBtn = widgets.ToolText("‹ Back", func() { travel(false) })
	backBtn.Tip = "Back (Alt+Left)"
	fwdBtn = widgets.ToolText("Forward ›", func() { travel(true) })
	fwdBtn.Tip = "Forward (Alt+Right)"
	tools := widgets.NewToolBar(backBtn, fwdBtn, widgets.ToolDivider(), newBtn, openBtn, saveBtn, widgets.ToolDivider(), searchBtn)
	syncHistory = func() {
		var ft *folderTab
		if t := tabs.Selected(); t >= 0 {
			ft, _ = tabs.Tab(t).Data.(*folderTab)
		}
		back, fwd := ft != nil && len(ft.back) > 0, ft != nil && len(ft.fwd) > 0
		if backBtn.Disabled == !back && fwdBtn.Disabled == !fwd {
			return
		}
		backBtn.Disabled, fwdBtn.Disabled = !back, !fwd
		tools.Invalidate()
	}

	filter := widgets.NewTextField("", "Filter this folder", func(s string) {
		mark("Filter  " + s)
	})

	sidebar := widgets.NewColumn(
		widgets.NewTitle("Places"),
		tree,
		widgets.NewLabel("Projects + folders"),
	).WithGap(8).WithPad(10)
	sidebar.AddFlex(tree, 1)

	previewTab := widgets.NewPad(8, preview)
	detailCol := widgets.NewColumn(pathLbl, widgets.NewSeparator(), details).WithGap(8).WithPad(12)
	pages := widgets.NewTabView(
		widgets.Tab{Title: "Preview", Content: previewTab},
		widgets.Tab{Title: "Details", Content: detailCol},
	)
	pages.OnChange = func(i int) {
		if i == 0 {
			mark("Preview")
		} else {
			mark("Details")
		}
	}

	listing := widgets.NewColumn(filter, table).WithGap(8).WithPad(8)
	listing.AddFlex(table, 1)

	right := widgets.NewColumn(listing, pages).WithGap(0)
	right.AddFlex(listing, 1)
	right.AddFlex(pages, 1)

	split := widgets.NewSplitter(true, sidebar, right)
	split.Ratio = 0.28

	// The folder tabs.
	keepOne := func() {
		// A window keeps its last tab (Dolphin's, Konsole's way).
		for i := 0; i < tabs.Len(); i++ {
			tab := tabs.Tab(i)
			tab.NoClose = tabs.Len() == 1
			tabs.SetTab(i, tab)
		}
	}
	openTab := func(at, p int) int {
		tabs.InsertTab(at, widgets.BrowserTab{Title: places[p].Label, Data: &folderTab{place: p}})
		keepOne()
		return at
	}
	selectTab := func(i int) {
		p, ok := tabPlace(tabs.Tab(i).Data)
		if !ok || p < 0 || p >= len(places) {
			return
		}
		if n := nodeOf[p]; n != nil {
			tree.Selected = n
			tree.Invalidate()
		}
		showPlace(p)
	}
	newTab := func() {
		// Home, next to the selected tab.
		tabs.Select(openTab(tabs.Selected()+1, 2))
		mark("New tab")
	}
	tabs.OnSelect = func(i int) {
		selectTab(i)
		mark("Tab  " + tabs.Tab(i).Title)
	}
	tabs.OnNew = newTab
	tabs.OnClose = func(i int) {
		title := tabs.Tab(i).Title
		tabs.RemoveTab(i)
		keepOne()
		mark("Closed " + title)
	}
	tabs.OnReorder = func(from, to int) { mark(fmt.Sprintf("Moved %s to %d", tabs.Tab(to).Title, to+1)) }
	// A folder tab takes files too — dropping a file on another tab is how
	// a file manager moves it there without opening the folder first.
	tabs.DropActions = platform.DragCopy | platform.DragMove
	tabs.OnDropTab = func(i int, e widget.DropEvent) bool {
		dst, ok := tabPlace(tabs.Tab(i).Data)
		if !ok || !filesDropInto(places, dst, e) {
			return false
		}
		refreshTable()
		mark(filesDropMessage(e, places[dst].Label))
		return true
	}
	tabs.OnContextMenu = func(i int, at paintengine2d.Point) bool {
		items := []*widgets.MenuItem{widgets.ItemAccel("New Tab", "Ctrl+T", newTab)}
		if i >= 0 {
			items = append(items,
				widgets.Item("Duplicate Tab", func() {
					if p, ok := tabPlace(tabs.Tab(i).Data); ok {
						tabs.Select(openTab(i+1, p))
					}
				}),
				widgets.Item("Move Tab to New Window", func() {
					tabs.TearOffTab(i, at)
				}),
				widgets.Sep(),
				widgets.ItemAccel("Close Tab", "Ctrl+W", func() { tabs.CloseTab(i) }),
				widgets.Item("Close Other Tabs", func() {
					keep := tabs.Tab(i)
					for j := tabs.Len() - 1; j >= 0; j-- {
						if tabs.Tab(j).Data != keep.Data || tabs.Tab(j).Title != keep.Title {
							tabs.RemoveTab(j)
						}
					}
					keepOne()
				}))
		}
		widgets.ShowContextMenu(tabs, at, items...)
		return true
	}
	// A tab dragged out of the strip opens a Files window of its own
	// showing that folder, and the desktop carries it under the pointer
	// (docs/decorations.md). Dropped on another Files window's strip it
	// joins that one instead.
	tabs.OnTearOff = func(_ int, tab widgets.BrowserTab) widget.TearOffWindow {
		p, ok := tabPlace(tab.Data)
		a := win.App()
		if !ok || a == nil {
			return nil
		}
		w, hh := 1040, 680
		if win != nil {
			if cw, ch := win.Size(); cw > 0 && ch > 0 {
				w, hh = cw, ch
			}
		}
		w2, err := a.NewWindow(platform.WindowOptions{
			Title: places[p].Label + " — Files", Width: w, Height: hh,
			MinWidth: 720, MinHeight: 480,
		})
		if err != nil {
			mark(err.Error())
			return nil
		}
		w2.SetContent(filesApp(w2, []int{p}))
		return w2
	}
	tabs.OnMergeTab = func(at int, tab widgets.BrowserTab, _ widget.DropEvent) bool {
		p, ok := tabPlace(tab.Data)
		if !ok {
			// From another Files process there is no index, only the
			// title the private type carries.
			if p, ok = placeNamed(places, tab.Title); !ok {
				return false
			}
		}
		tabs.Select(openTab(at, p))
		mark("Merged " + places[p].Label)
		return true
	}
	// Besides the tab itself, a folder tab carries its path as text, so
	// dropping one in a terminal or an editor says where it was.
	tabs.OnTabDrag = func(_ int, tab widgets.BrowserTab) *widget.Drag {
		if p, ok := tabPlace(tab.Data); ok {
			return widget.DragText(places[p].Path)
		}
		return nil
	}
	openTab(0, place)
	if len(open) == 0 {
		openTab(1, 0)
	}
	tabs.Select(0)
	syncHistory()

	// The tabs are the window's title bar: the caption of the frame the
	// toolkit draws (the caption buttons beside them, the empty strip moves
	// the window), or the first row under the desktop's frame.
	head := widgets.NewHeaderBar([]widget.Component{menubar}, tabs, nil)
	var rows []widget.Component
	if win != nil {
		win.SetTitleBar(head)
		win.SetTitle(places[place].Label + " — Files")
	} else {
		rows = append(rows, head)
	}
	rows = append(rows, tools, split, status)
	col := widgets.NewColumn(rows...).WithGap(0)
	col.AddFlex(split, 1)

	loadPreview()
	root := newShortcutRoot(col, func(e widget.KeyEvent) bool {
		// Alt+Left and Alt+Right go back and forward, on both desktops.
		if e.Mods.Alt() && !e.Mods.Ctrl() && (e.Key == platform.KeyLeft || e.Key == platform.KeyRight) {
			travel(e.Key == platform.KeyRight)
			return true
		}
		return tabs.Shortcut(e)
	})
	// The mouse's back and forward buttons and a sideways swipe
	// (widget.HistoryNavigator).
	root.navigate = travel
	return root
}

// folderTab is what a Files tab holds: the folder it shows and the ones it
// can go back and forward to, so each tab has a history of its own — and
// keeps it when it is torn off into a window of its own.
type folderTab struct {
	place     int
	back, fwd []int
}

// travel moves one step back or forward; ok is false at either end.
func (f *folderTab) travel(forward bool) (int, bool) {
	from, to := &f.back, &f.fwd
	if forward {
		from, to = &f.fwd, &f.back
	}
	if len(*from) == 0 {
		return 0, false
	}
	next := (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	*to = append(*to, f.place)
	f.place = next
	return next, true
}

// tabFolder is tab i's folderTab, made for a tab that has none.
func tabFolder(tabs *widgets.BrowserTabs, i int) *folderTab {
	tab := tabs.Tab(i)
	if ft, ok := tab.Data.(*folderTab); ok {
		return ft
	}
	p, _ := tabPlace(tab.Data)
	ft := &folderTab{place: p}
	tab.Data = ft
	tabs.SetTab(i, tab)
	return ft
}

// tabPlace is the folder a tab shows.
func tabPlace(d any) (int, bool) {
	switch v := d.(type) {
	case *folderTab:
		return v.place, true
	case int:
		return v, true
	}
	return 0, false
}

// shortcutRoot is a window's content root with the app's own keys: those
// no focused widget took reach it (a browser's Ctrl+T, Ctrl+W, Ctrl+Tab).
type shortcutRoot struct {
	widget.Base
	onKey func(widget.KeyEvent) bool
	// navigate, if set, goes back or forward (widget.HistoryNavigator).
	navigate func(forward bool) bool
}

func newShortcutRoot(child widget.Component, onKey func(widget.KeyEvent) bool) *shortcutRoot {
	r := &shortcutRoot{onKey: onKey}
	r.Init(r)
	r.Add(child)
	return r
}

func (r *shortcutRoot) Measure(c layout.Constraints) paintengine2d.Point {
	return r.Children()[0].Measure(c)
}

func (r *shortcutRoot) Arrange(b paintengine2d.Rect) {
	r.SetBounds(b)
	r.Children()[0].Arrange(paintengine2d.XYWH(0, 0, b.Dx(), b.Dy()))
}

func (r *shortcutRoot) KeyPress(e widget.KeyEvent) bool { return r.onKey != nil && r.onKey(e) }

// NavigateHistory implements widget.HistoryNavigator.
func (r *shortcutRoot) NavigateHistory(forward bool) bool {
	return r.navigate != nil && r.navigate(forward)
}

// lookFaces are the typefaces the window's look reads in (its era's when
// installed, else the bundled Titillium Web and JetBrains Mono).
func lookFaces(win *app.Window) (ui, mono string) {
	if c, ok := win.Look().(*style.Classic); ok {
		return c.UIFamily(), c.MonoFamily()
	}
	return style.FamilyUI, style.FamilyMono
}

func isCodeName(name string) bool {
	n := strings.ToLower(name)
	for _, ext := range []string{".go", ".c", ".h", ".rs", ".js", ".ts", ".mod", ".json"} {
		if strings.HasSuffix(n, ext) {
			return true
		}
	}
	return false
}

func sortFileRows(rows []fileRow, col int, asc bool) {
	less := func(i, j int) bool {
		var a, b string
		switch col {
		case 1:
			a, b = rows[i].Kind, rows[j].Kind
		case 2:
			a, b = rows[i].Size, rows[j].Size
		case 3:
			a, b = rows[i].Modified, rows[j].Modified
		default:
			a, b = rows[i].Name, rows[j].Name
		}
		if !asc {
			return a > b
		}
		return a < b
	}
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
}

func samplePlaces() []filePlace {
	goBody := `package main

import "fmt"

func main() {
    fmt.Println("uitoolkit files")
}
`
	readme := "Files / Projects\n\nTitillium Web labels.\nJetBrains Mono for .go preview.\n"
	return []filePlace{
		{
			Label: "Projects",
			Path:  "/work/projects",
			Rows: []fileRow{
				{Name: "uitoolkit", Kind: "Folder", Size: "—", Modified: "Today", Dir: true, Body: "Go desktop UI toolkit."},
				{Name: "paintengine2d", Kind: "Folder", Size: "—", Modified: "Today", Dir: true, Body: "CPU scanline painter."},
				{Name: "notes", Kind: "Folder", Size: "—", Modified: "Mon", Dir: true, Body: "Sample notes app."},
			},
		},
		{
			Label: "uitoolkit",
			Path:  "/work/projects/uitoolkit",
			Rows: []fileRow{
				{Name: "main.go", Kind: "Go", Size: "1 KB", Modified: "Today", Body: goBody},
				{Name: "export.go", Kind: "Go", Size: "9 KB", Modified: "Today", Body: "package uitoolkit\n\n// public widgets\n"},
				{Name: "README.md", Kind: "Markdown", Size: "8 KB", Modified: "Today", Body: readme},
				{Name: "go.mod", Kind: "Go module", Size: "120 B", Modified: "Tue", Body: "module github.com/codemodify/uitoolkit\n\ngo 1.22.2\n"},
				{Name: "version.go", Kind: "Go", Size: "80 B", Modified: "Today", Body: "package uitoolkit\n\nconst Version = \"0.8.2\"\n"},
			},
		},
		{
			Label: "Home",
			Path:  "/home/ada",
			Rows: []fileRow{
				{Name: "Documents", Kind: "Folder", Size: "—", Modified: "Sun", Dir: true, Body: "Personal documents."},
				{Name: "todo.txt", Kind: "Text", Size: "220 B", Modified: "Yesterday", Body: "Event-driven Run.\nTag v0.5.1.\n"},
			},
		},
	}
}

// placeNamed is the folder a label names, for a tab that arrived from
// another Files process: the private type carries the title and the
// document behind it is this process's own.
func placeNamed(places []filePlace, label string) (int, bool) {
	for i, p := range places {
		if p.Label == label {
			return i, true
		}
	}
	return 0, false
}
