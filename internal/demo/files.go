package demo

import (
	"fmt"
	"path"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
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
func FilesApp(win *app.Window) widget.Component {
	places := samplePlaces()
	place := 1
	if place >= len(places) {
		place = 0
	}
	sel := 0

	status := widgets.NewStatusBar("Ready.", places[0].Path, "v"+uitoolkit.Version)
	mark := func(msg string) { status.Set(0, msg) }

	preview := widgets.NewTextArea("", "Select a file", nil)
	preview.MinRows = 8
	preview.Wrap = false

	details := widgets.NewLabel("")
	pathLbl := widgets.NewLabel("")

	var table *widgets.TableView
	var tree *widgets.TreeView

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
		details.SetText(fmt.Sprintf("%s\n%s  ·  %s  ·  %s\nTitillium UI  ·  %s preview",
			r.Name, kind, r.Size, r.Modified, previewFace(r.Name)))
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
		table.Selected = sel
		table.Invalidate()
		loadPreview()
		status.Set(0, fmt.Sprintf("%d items", len(rows)))
	}

	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Sortable: true},
		{Title: "Kind", Width: 88, Sortable: true},
		{Title: "Size", Width: 72, Sortable: true, Align: style.AlignEnd},
		{Title: "Modified", Width: 110, Sortable: true},
	}, len(places[0].Rows), func(row, col int) string {
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
	table.Selected = 0
	table.OnSort = func(col int, asc bool) {
		rows := places[place].Rows
		sortFileRows(rows, col, asc)
		refreshTable()
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
	for i, p := range places {
		n := widgets.NewTreeNode(p.Label)
		n.Data = i
		for _, r := range p.Rows {
			if r.Dir {
				child := widgets.NewTreeNode(r.Name)
				if j := placeIndex(r.Name); j >= 0 {
					child.Data = j
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
	if place >= 0 && place < len(roots) {
		tree.Selected = roots[place]
	} else {
		tree.Selected = roots[0]
	}
	tree.OnSelect = func(n *widgets.TreeNode) {
		if n == nil {
			return
		}
		i, ok := n.Data.(int)
		if !ok || i < 0 || i >= len(places) {
			return
		}
		place = i
		sel = 0
		refreshTable()
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
		widgets.Info(win.Content(), "About Files",
			"Files / Projects dogfood on uitoolkit.\nUI: Titillium Web. Code preview: JetBrains Mono.",
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
	tools := widgets.NewToolBar(newBtn, openBtn, saveBtn, widgets.ToolDivider(), searchBtn)

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
	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Preview", Content: previewTab},
		widgets.Tab{Title: "Details", Content: detailCol},
	)
	tabs.OnChange = func(i int) {
		if i == 0 {
			mark("Preview")
		} else {
			mark("Details")
		}
	}

	listing := widgets.NewColumn(filter, table).WithGap(8).WithPad(8)
	listing.AddFlex(table, 1)

	right := widgets.NewColumn(listing, tabs).WithGap(0)
	right.AddFlex(listing, 1)
	right.AddFlex(tabs, 1)

	split := widgets.NewSplitter(true, sidebar, right)
	split.Ratio = 0.28

	chrome := widgets.NewTitleBar("Files", "projects  ·  Titillium Web  ·  v"+uitoolkit.Version)
	root := widgets.NewColumn(menubar, tools, chrome, split, status).WithGap(0)
	root.AddFlex(split, 1)

	loadPreview()
	return root
}

func previewFace(name string) string {
	if isCodeName(name) {
		return "JetBrains Mono"
	}
	return "Titillium Web"
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
				{Name: "version.go", Kind: "Go", Size: "80 B", Modified: "Today", Body: "package uitoolkit\n\nconst Version = \"0.8.0\"\n"},
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
