// Command look-like-vscode builds a code editor's shell out of uitoolkit
// widgets: a title bar the application fills, an activity rail down the
// left with badges, a side bar the rail switches, an editor area, and a
// status bar along the bottom.
//
// The point of the sample is that none of that shape is a feature. It is
// a HeaderBar, a Row, a Column and a Splitter; what makes it look like
// an editor is how they are filled.
//
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-vscode
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-vscode -headless
package main

import (
	"flag"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// rail is one icon down the activity bar.
type rail struct {
	icon  style.ToolIcon
	name  string
	badge string
}

var rails = []rail{
	{style.IconNew, "Explorer", ""},
	{style.IconSearch, "Search", ""},
	{style.IconExternalLink, "Source Control", ""},
	{style.IconArrowRight, "Run and Debug", ""},
	{style.IconArchive, "Extensions", ""},
	{style.IconFlag, "Testing", ""},
}

type editor struct {
	side      *widgets.FlexBox
	sideTitle *widgets.Label
	buttons   []*widgets.IconButton
}

// themePack is the look this sample wears. Theme engines are chosen at
// build time (docs/engines.md), so a plain build has only the default
// one and the look falls back to the default — which the toolkit
// says at start-up, and app.Application.Diagnostics collects for a test.
const themePack = "vscode"

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write vscode.png")
	flag.Parse()

	// The shape is only half of looking like vscode; the other half is the
	// paint. The sample states the pack it wants and nothing else, so a
	// user's icon set, corner policy and typefaces still come from their
	// own settings — VS Code's own light pack.
	a := uitoolkit.New(uitoolkit.Options{
		Headless: *headless,
		// Without this the id would be the binary's name, and a sample
		// built as "vscode" would claim the real vscode's identity on the
		// desktop — its task-bar slot, its icon, its window rules.
		AppID: "uitoolkit-sample-look-like-vscode",
		Theme: uitoolkit.ThemeOverride{Pack: themePack},
	})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit-sample-look-like-vscode", Width: 1100, Height: 700,
		MinWidth: 640, MinHeight: 420, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}

	e := &editor{}
	// VS Code's menu, title and its own buttons share one row that is the
	// window's caption, under every theme it ships.
	win.SetTitleBar(e.titleBar(win))
	win.SetCaptionStyle(uitoolkit.CaptionMerged)
	// The application draws to the window's edges, so an era's bevel
	// round the outside would read as a frame inside a frame.
	win.SetBorderless(true)
	win.SetContent(e.body())

	if *headless {
		if err := win.WritePNG("vscode.png"); err != nil {
			log.Fatal(err)
		}
		return
	}
	a.Run()
}

// titleBar is the row the application fills: its mark, its menus, the
// document's name in the middle, and its own buttons at the end. The
// window's close, minimize and maximize are added by the toolkit on
// whichever side this era puts them, so this does not have to know.
func (e *editor) titleBar(win *app.Window) *widgets.HeaderBar {
	menus := widgets.NewRow(
		widgets.NewFlatIconButton(style.IconLayout, "uitoolkit", nil),
		menu("File", "New File", "Open…", "Save"),
		menu("Edit", "Undo", "Redo", "Find…"),
		menu("Selection", "Select All", "Expand Selection"),
		menu("View", "Command Palette…", "Appearance"),
		widgets.NewFlatMenuButton(style.IconMore, "More",
			&widgets.MenuItem{Text: "Preferences"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: win.Close}),
	).WithGap(2).WithPadding(4, 0, 0, 0)

	title := widgets.NewLabel("main.go — uitoolkit")
	title.Align = style.AlignCenter

	head := widgets.NewHeaderBar(
		[]widget.Component{menus},
		title,
		[]widget.Component{
			widgets.NewFlatIconButton(style.IconColumns, "Toggle Primary Side Bar", nil),
			widgets.NewFlatIconButton(style.IconRows, "Toggle Panel", nil),
			widgets.NewFlatIconButton(style.IconLayout, "Customize Layout", nil),
		})
	head.ShowTitle = false
	return head
}

// menu is one of the title bar's named menus.
func menu(name string, items ...string) *widgets.MenuButton {
	rows := make([]*widgets.MenuItem, 0, len(items))
	for _, it := range items {
		rows = append(rows, &widgets.MenuItem{Text: it})
	}
	b := widgets.NewTextMenuButton(name, rows...)
	// A menu bar is a row of words, not of buttons. VS Code's File and
	// Edit have no frame until the pointer is over them.
	b.Flat = true
	return b
}

// body is the rail, the side bar, the editor and the status bar.
func (e *editor) body() widget.Component {
	// A header and a list on the side bar's own surface — not a Panel,
	// whose frame and title band draw a box inside the pane. VS Code's
	// Explorer has no box: a small bold word, then the files.
	e.sideTitle = widgets.NewLabel(rails[0].name)
	e.sideTitle.Title = true
	files := widgets.NewScrollView(fileList())
	e.side = widgets.NewColumn(widgets.NewPad(8, e.sideTitle), files).WithGap(0)
	e.side.AddFlex(files, 1)

	split := widgets.NewSplitter(widgets.SplitColumns, e.side, welcome())
	split.Ratio = 0.24

	middle := widgets.NewRow(e.activityBar(), split).WithGap(0)
	middle.AddFlex(split, 1)

	col := widgets.NewColumn(middle, statusBar()).WithGap(0)
	col.AddFlex(middle, 1)
	return col
}

// activityBar is the column of icons down the left edge. Each latches;
// the splitter will not let the pane holding them be dragged narrower
// than they need.
func (e *editor) activityBar() *widgets.FlexBox {
	col := widgets.NewColumn().WithGap(2).WithPad(4)
	for i, r := range rails {
		i, r := i, r
		b := widgets.NewFlatIconButton(r.icon, r.name, func() { e.choose(i) })
		b.Toggle = true
		b.Checked = i == 0
		b.Badge = r.badge
		e.buttons = append(e.buttons, b)
		col.Add(b)
	}
	gap := widgets.NewSpacer()
	col.Add(gap)
	col.AddFlex(gap, 1)
	col.Add(widgets.NewFlatIconButton(style.IconUser, "Accounts", nil))
	col.Add(widgets.NewFlatIconButton(style.IconSettings, "Manage", nil))
	return col
}

// choose latches one rail button and retitles the side bar.
func (e *editor) choose(i int) {
	for k, b := range e.buttons {
		b.Checked = k == i
	}
	e.sideTitle.Text = rails[i].name
	e.sideTitle.Invalidate()
}

func fileList() widget.Component {
	col := widgets.NewColumn().WithGap(2).WithPad(6)
	for _, name := range []string{"main.go", "shell.go", "README.md", "go.mod", "go.sum"} {
		col.Add(widgets.NewLabel(name))
	}
	return col
}

func welcome() widget.Component {
	rows := widgets.NewColumn().WithGap(8)
	for _, r := range [][2]string{
		{"Open Chat", "Ctrl+Alt+I"},
		{"Show All Commands", "Ctrl+Shift+P"},
		{"Start Debugging", "F5"},
	} {
		name := widgets.NewLabel(r[0])
		key := widgets.NewLabel(r[1])
		key.Mono = true
		key.Align = style.AlignEnd
		row := widgets.NewRow(name, key).WithGap(24)
		row.AddFlex(name, 1)
		rows.Add(row)
	}
	return widgets.NewPad(48, rows)
}

func statusBar() widget.Component {
	left := widgets.NewRow(
		widgets.NewFlatIconButton(style.IconExternalLink, "Open a Remote Window", nil),
		widgets.NewLabel("dev*"),
		widgets.NewFlatIconButton(style.IconSync, "Synchronize Changes", nil),
		widgets.NewLabel("0 errors, 0 warnings"),
	).WithGap(8)
	right := widgets.NewRow(
		widgets.NewFlatIconButton(style.IconStar, "Chat", nil),
		widgets.NewFlatIconButton(style.IconBell, "Notifications", nil),
	).WithGap(4)
	gap := widgets.NewSpacer()
	row := widgets.NewRow(left, gap, right).WithGap(8).WithPad(4)
	row.AddFlex(gap, 1)
	return row
}
