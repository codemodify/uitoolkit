// Package shellapp is an application shell of the shape code editors and
// web dashboards have settled on: a title bar the application fills, a
// rail of icons down one side, a sidebar, and a status bar along the
// bottom.
//
// It is here because two applications asked whether the toolkit could
// make one, and the honest way to answer was to build it. Everything in
// here is ordinary widgets — the shape is not a feature, it is a layout
// — and that is the point the sample makes.
package shellapp

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Shell is the whole window's content.
type Shell struct {
	win     *app.Window
	rail    *widgets.FlexBox
	side    *widgets.Panel
	split   *widgets.Splitter
	status  *widgets.FlexBox
	branch  *widgets.Label
	active  int
	railBtn []*widgets.IconButton
}

// railItem is one icon down the left edge.
type railItem struct {
	icon  style.ToolIcon
	name  string
	badge string
}

var railItems = []railItem{
	{style.IconNew, "Explorer", ""},
	{style.IconSearch, "Search", ""},
	{style.IconExternalLink, "Source Control", "86"},
	{style.IconForward, "Run and Debug", ""},
	{style.IconArchive, "Extensions", "2"},
	{style.IconFlag, "Testing", ""},
	{style.IconStar, "Chat", ""},
}

// New builds the shell on w.
func New(w *app.Window) widget.Component {
	s := &Shell{win: w}

	// The title bar: the application's menu at the start, the window's
	// name in the middle, its own buttons at the end. The window's
	// controls are added by the toolkit on whichever side this era puts
	// them, so this row does not have to know.
	appMark := widgets.NewIconButton(style.IconLayout, "comms-mail", nil)
	menu := widgets.NewRow(
		appMark,
		s.menuButton("File", "New File", "Open…", "Save"),
		s.menuButton("Edit", "Undo", "Redo", "Find…"),
		s.menuButton("Selection", "Select All", "Expand"),
		s.menuButton("View", "Command Palette…", "Appearance"),
		widgets.NewMenuButton(style.IconMore, "More",
			&widgets.MenuItem{Text: "Preferences"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: func() { w.Close() }}),
	).WithGap(2)

	title := widgets.NewLabel("comms-mail — uitoolkit shell")
	title.Align = style.AlignCenter

	head := widgets.NewHeaderBar(
		[]widget.Component{menu},
		title,
		[]widget.Component{
			widgets.NewIconButton(style.IconTable, "Toggle Panel", nil),
			widgets.NewIconButton(style.IconColumns, "Toggle Side Bar", nil),
			widgets.NewIconButton(style.IconLayout, "Customize Layout", nil),
		})
	head.ShowTitle = false
	w.SetTitleBar(head)

	// The application draws to its own edges, so the era's border round
	// the outside would read as a frame inside a frame.
	w.SetBorderless(true)

	s.rail = s.buildRail()
	s.side = s.buildSidebar()
	body := widgets.NewLabel("Open a file to begin.")
	body.Align = style.AlignCenter
	body.Wrap = true

	s.split = widgets.NewSplitter(widgets.SplitColumns, s.side, s.pane(body))
	s.split.Ratio = 0.26

	s.status = s.buildStatus()

	middle := widgets.NewRow(s.rail, s.split).WithGap(0)
	middle.AddFlex(s.split, 1)

	col := widgets.NewColumn(middle, s.status).WithGap(0)
	col.AddFlex(middle, 1)
	return col
}

// menuButton is one of the title bar's menus.
func (s *Shell) menuButton(name string, items ...string) *widgets.MenuButton {
	rows := make([]*widgets.MenuItem, 0, len(items))
	for _, it := range items {
		rows = append(rows, &widgets.MenuItem{Text: it})
	}
	// A named menu rather than a mark: the button shows the word.
	return widgets.NewTextMenuButton(name, rows...)
}

// buildRail is the column of icons down the left edge — the activity bar
// of every editor written since 2015.
func (s *Shell) buildRail() *widgets.FlexBox {
	col := widgets.NewColumn().WithGap(2).WithPad(4)
	for i, it := range railItems {
		i, it := i, it
		b := widgets.NewIconButton(it.icon, it.name, func() { s.selectRail(i) })
		b.Toggle = true
		b.Checked = i == s.active
		if it.badge != "" {
			b.Content = badgedIcon(b, it.badge)
		}
		s.railBtn = append(s.railBtn, b)
		col.Add(b)
	}
	col.Add(widgets.NewSpacer())
	col.AddFlex(col.Children()[len(col.Children())-1], 1)
	col.Add(widgets.NewIconButton(style.IconUser, "Accounts", nil))
	col.Add(widgets.NewIconButton(style.IconSettings, "Manage", nil))
	return col
}

// selectRail latches one rail button and unlatches the rest.
func (s *Shell) selectRail(i int) {
	s.active = i
	for k, b := range s.railBtn {
		b.Checked = k == i
	}
	s.side.Title = railItems[i].name
	s.side.Invalidate()
}

// buildSidebar is the pane the rail switches.
func (s *Shell) buildSidebar() *widgets.Panel {
	list := widgets.NewColumn().WithGap(2).WithPad(6)
	for _, name := range []string{"main.go", "shell.go", "README.md", "go.mod"} {
		list.Add(widgets.NewLabel(name))
	}
	p := widgets.NewPanel(railItems[0].name, widgets.NewScrollView(list))
	return p
}

// pane is the editor area.
func (s *Shell) pane(c widget.Component) widget.Component {
	return widgets.NewPad(0, c)
}

// buildStatus is the bar along the bottom.
func (s *Shell) buildStatus() *widgets.FlexBox {
	s.branch = widgets.NewLabel("dev*")
	left := widgets.NewRow(
		widgets.NewIconButton(style.IconExternalLink, "Remote Window", nil),
		s.branch,
		widgets.NewIconButton(style.IconRedo, "Synchronize Changes", nil),
		widgets.NewLabel("0 errors, 0 warnings"),
	).WithGap(8)
	right := widgets.NewRow(
		widgets.NewIconButton(style.IconStar, "Chat", nil),
		widgets.NewIconButton(style.IconBell, "Notifications", nil),
	).WithGap(4)
	row := widgets.NewRow(left, widgets.NewSpacer(), right).WithGap(8).WithPad(4)
	row.AddFlex(row.Children()[1], 1)
	return row
}

// badgedIcon draws the button's icon with a count on it, the way an
// activity bar marks pending work.
func badgedIcon(b *widgets.IconButton, text string) widgets.ButtonContentPainter {
	inner := b.Content
	return func(ctx *paintengine2d.Context, r paintengine2d.Rect, st style.ControlState) {
		if inner != nil {
			inner(ctx, r, st)
		}
		lk := b.Look()
		f := lk.MutedFont()
		w := f.Advance(text) + style.Dip(lk, 6)
		h := f.Height()
		box := paintengine2d.XYWH(r.Max.X-w-2, r.Max.Y-h-2, w, h)
		ctx.DrawRoundRect(box, h*0.5, h*0.5, paintengine2d.Fill(lk.Palette().Accent))
		f.Draw(ctx, text,
			paintengine2d.Pt(box.Min.X+style.Dip(lk, 3), box.Min.Y),
			lk.Palette().Background)
	}
}

var _ = fmt.Sprintf
var _ = layout.AlignStart

// NewBrowser is the other title-bar shape: the window's controls, then
// back and forward, then a field that takes the rest of the width — what
// a browser's chrome looks like, and what a web dashboard is seen
// through.
//
// It is the same HeaderBar as the editor shape, filled differently. The
// window's own buttons are placed by the era on whichever side that era
// puts them, so neither shape has to know where they are: the
// application says what goes at the start, in the middle and at the end,
// and the controls take their own side.
func NewBrowser(w *app.Window) widget.Component {
	back := widgets.NewIconButton(style.IconArrowLeft, "Back", nil)
	fwd := widgets.NewIconButton(style.IconArrowRight, "Forward", nil)
	reload := widgets.NewIconButton(style.IconRedo, "Reload", nil)

	// The address field takes the free width, which is what the centre
	// slot is for.
	addr := widgets.NewTextField("dash.example.com/c4d61c0f435a", "Search or enter address", nil)

	head := widgets.NewHeaderBar(
		[]widget.Component{back, fwd, reload},
		addr,
		[]widget.Component{
			widgets.NewIconButton(style.IconUser, "Profile", nil),
			widgets.NewMenuButton(style.IconMore, "Menu",
				&widgets.MenuItem{Text: "New Tab", Shortcut: "Ctrl+T"},
				&widgets.MenuItem{Text: "History"},
				&widgets.MenuItem{Separator: true},
				&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: func() { w.Close() }}),
		})
	head.ShowTitle = false
	w.SetTitleBar(head)
	w.SetBorderless(true)

	// A rail, a sidebar and a page — the dashboard under the chrome.
	rail := widgets.NewColumn().WithGap(6).WithPad(6)
	for _, it := range []railItem{
		{style.IconMenu, "Collapse", ""},
		{style.IconCards, "Overview", ""},
		{style.IconSearch, "Explore", ""},
		{style.IconStar, "Favourites", ""},
	} {
		rail.Add(widgets.NewIconButton(it.icon, it.name, nil))
	}

	side := widgets.NewColumn().WithGap(6).WithPad(8)
	side.Add(widgets.NewTextField("", "Quick search…", nil))
	for _, name := range []string{"Account home", "Recents", "Domains", "Workers", "R2"} {
		side.Add(widgets.NewLabel(name))
	}

	page := widgets.NewLabel("The page")
	page.Align = style.AlignCenter

	split := widgets.NewSplitter(widgets.SplitColumns, widgets.NewScrollView(side), page)
	split.Ratio = 0.24

	row := widgets.NewRow(rail, split).WithGap(0)
	row.AddFlex(split, 1)
	return row
}
