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

	"github.com/codemodify/paintengine2d"
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
	{style.IconExternalLink, "Source Control", "86"},
	{style.IconArrowRight, "Run and Debug", ""},
	{style.IconArchive, "Extensions", "2"},
	{style.IconFlag, "Testing", ""},
}

type editor struct {
	side    *widgets.Panel
	buttons []*widgets.IconButton
}

// themePack is the look this sample wears. Theme engines are chosen at
// build time (docs/engines.md), so a plain build has only the default
// one and the look falls back to the default — which the toolkit
// says at start-up, and app.Application.Diagnostics collects for a test.
const themePack = "vscode-night"

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write vscode.png")
	flag.Parse()

	// The shape is only half of looking like vscode; the other half is the
	// paint. The sample states the pack it wants and nothing else, so a
	// user's icon set, corner policy and typefaces still come from their
	// own settings — VS Code's own dark pack.
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
		widgets.NewIconButton(style.IconLayout, "uitoolkit", nil),
		menu("File", "New File", "Open…", "Save"),
		menu("Edit", "Undo", "Redo", "Find…"),
		menu("Selection", "Select All", "Expand Selection"),
		menu("View", "Command Palette…", "Appearance"),
		widgets.NewMenuButton(style.IconMore, "More",
			&widgets.MenuItem{Text: "Preferences"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: win.Close}),
	).WithGap(2)

	title := widgets.NewLabel("main.go — uitoolkit")
	title.Align = style.AlignCenter

	head := widgets.NewHeaderBar(
		[]widget.Component{menus},
		title,
		[]widget.Component{
			widgets.NewIconButton(style.IconColumns, "Toggle Primary Side Bar", nil),
			widgets.NewIconButton(style.IconRows, "Toggle Panel", nil),
			widgets.NewIconButton(style.IconLayout, "Customize Layout", nil),
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
	return widgets.NewTextMenuButton(name, rows...)
}

// body is the rail, the side bar, the editor and the status bar.
func (e *editor) body() widget.Component {
	e.side = widgets.NewPanel(rails[0].name, widgets.NewScrollView(fileList()))

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
		b := widgets.NewIconButton(r.icon, r.name, func() { e.choose(i) })
		b.Toggle = true
		b.Checked = i == 0
		if r.badge != "" {
			b.Content = withBadge(b, r.badge)
		}
		e.buttons = append(e.buttons, b)
		col.Add(b)
	}
	gap := widgets.NewSpacer()
	col.Add(gap)
	col.AddFlex(gap, 1)
	col.Add(widgets.NewIconButton(style.IconUser, "Accounts", nil))
	col.Add(widgets.NewIconButton(style.IconSettings, "Manage", nil))
	return col
}

// choose latches one rail button and retitles the side bar.
func (e *editor) choose(i int) {
	for k, b := range e.buttons {
		b.Checked = k == i
	}
	e.side.Title = rails[i].name
	e.side.Invalidate()
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
		widgets.NewIconButton(style.IconExternalLink, "Open a Remote Window", nil),
		widgets.NewLabel("dev*"),
		widgets.NewIconButton(style.IconSync, "Synchronize Changes", nil),
		widgets.NewLabel("0 errors, 0 warnings"),
	).WithGap(8)
	right := widgets.NewRow(
		widgets.NewIconButton(style.IconStar, "Chat", nil),
		widgets.NewIconButton(style.IconBell, "Notifications", nil),
	).WithGap(4)
	gap := widgets.NewSpacer()
	row := widgets.NewRow(left, gap, right).WithGap(8).WithPad(4)
	row.AddFlex(gap, 1)
	return row
}

// withBadge draws the button's icon with a count on it, the way an
// activity bar marks pending work.
func withBadge(b *widgets.IconButton, text string) widgets.ButtonContentPainter {
	inner := b.Content
	return func(ctx *paintengine2d.Context, r paintengine2d.Rect, st style.ControlState) {
		if inner != nil {
			inner(ctx, r, st)
		}
		lk := b.Look()
		f := lk.MutedFont()
		w := f.Advance(text) + style.Dip(lk, 6)
		h := f.Height()
		box := paintengine2d.XYWH(r.Max.X-w-1, r.Max.Y-h-1, w, h)
		ctx.DrawRoundRect(box, h*0.5, h*0.5, paintengine2d.Fill(lk.Palette().Accent))
		f.Draw(ctx, text, paintengine2d.Pt(box.Min.X+style.Dip(lk, 3), box.Min.Y),
			lk.Palette().Background)
	}
}
