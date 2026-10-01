// Command look-like-thunderbird builds a mail client's shell: the
// actions in the title bar beside a search field, a folder tree, a
// message list, and a reading pane under its own row of actions.
//
// Its title bar is the interesting part. Get Messages, Write and Delete
// are icon *then* label, tight, and flat until the pointer is over them
// — which is ToolButton, not Button.Icon. A push button's label is
// centred by the engine, so an icon has to reserve a strip at each end;
// three of those in a title bar would cost nearly two hundred pixels of
// width for nothing. Knowing which of the shapes a control is *is* the
// job (docs/widgets.md, "Three shapes of button with a mark on it").
//
//	go run -tags theme_engine_adwaita ./examples/uitoolkit-sample-look-like-thunderbird
//	go run -tags theme_engine_adwaita ./examples/uitoolkit-sample-look-like-thunderbird -headless
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

type message struct{ subject, from, date string }

// Invented senders at example.com and example.org, which exist for this
// (RFC 2606) and can belong to nobody. A sample ships in screenshots, so
// a real address in it is somebody's inbox and a real brand in it is a
// claim the sample has no business making.
var inbox = []message{
	{"Your order has shipped", "Parcel Desk <noreply@example.com>", "2:08 PM"},
	{"£25 of ride credit is waiting", "City Rides <receipts@example.com>", "1:52 PM"},
	{"Final hours: go truly wireless", "First Backer <news@example.org>", "1:20 PM"},
	{"Endless \"DADA\" infinite slider", "Back The Best <contact@example.org>", "1:04 PM"},
	{"Your domain renews next month", "Domain Desk <billing@example.com>", "12:41 PM"},
	{"Your screen, finally moves with you", "First Backer <news@example.org>", "9/29"},
	{"Educational: assemble a smartphone yourself", "Back The Best <contact@example.org>", "9/29"},
}

// themePack is the look this sample wears. Theme engines are chosen at
// build time (docs/engines.md), so a plain build has only the default
// one and the look falls back to the default — which the toolkit
// says at start-up, and app.Application.Diagnostics collects for a test.
const themePack = "adwaita"

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write thunderbird.png")
	flag.Parse()

	// The shape is only half of looking like thunderbird; the other half is the
	// paint. The sample states the pack it wants and nothing else, so a
	// user's icon set, corner policy and typefaces still come from their
	// own settings — GNOME's flat chrome, which is what Thunderbird wears on this desktop.
	a := uitoolkit.New(uitoolkit.Options{
		Headless: *headless,
		// Without this the id would be the binary's name, and a sample
		// built as "thunderbird" would claim the real thunderbird's identity on the
		// desktop — its task-bar slot, its icon, its window rules.
		AppID: "uitoolkit-sample-look-like-thunderbird",
		Theme: uitoolkit.ThemeOverride{Pack: themePack},
	})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit-sample-look-like-thunderbird", Width: 1180, Height: 720,
		MinWidth: 700, MinHeight: 460, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetTitleBar(titleBar(win))
	win.SetBorderless(true)
	win.SetContent(body())

	if *headless {
		if err := win.WritePNG("thunderbird.png"); err != nil {
			log.Fatal(err)
		}
		return
	}
	a.Run()
}

// titleBar: a pane toggle, the three actions, a search field taking the
// free width, and the application menu.
func titleBar(win *app.Window) *widgets.HeaderBar {
	// Flat marks, not framed buttons. A push button's face beside a row of
	// tool buttons reads as a control stuck half pressed — the tool
	// buttons have no frame until the pointer is over them and this one
	// had one always. WithPadding keeps it off the window's edge.
	actions := widgets.NewRow(
		widgets.NewFlatIconButton(style.IconColumns, "Toggle Folder Pane", nil),
		widgets.NewToolButton("Get Messages", style.IconDownload, nil),
		widgets.NewToolButton("Write", style.IconPen, nil),
		widgets.NewToolButton("Delete", style.IconTrash, nil),
	).WithGap(4).WithPadding(6, 0, 0, 0)

	search := widgets.NewTextField("", "Search…   Ctrl+K", nil)

	head := widgets.NewHeaderBar(
		[]widget.Component{actions},
		search,
		[]widget.Component{
			widgets.NewFlatMenuButton(style.IconMenu, "Application Menu",
				&widgets.MenuItem{Text: "New Message", Shortcut: "Ctrl+N"},
				&widgets.MenuItem{Text: "Address Book"},
				&widgets.MenuItem{Separator: true},
				&widgets.MenuItem{Text: "Settings"},
				&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: win.Close}),
		})
	head.ShowTitle = false
	return head
}

// body is the three panes and the status bar.
func body() widget.Component {
	list := messageList()
	right := widgets.NewSplitter(widgets.SplitRows, list, readingPane())
	right.Ratio = 0.42

	split := widgets.NewSplitter(widgets.SplitColumns, widgets.NewScrollView(folders()), right)
	split.Ratio = 0.22

	col := widgets.NewColumn(split, statusBar()).WithGap(0)
	col.AddFlex(split, 1)
	return col
}

// folders is the account tree down the left.
func folders() widget.Component {
	t := widgets.NewTreeView()
	for _, account := range []string{"ada@example.com", "work@example.com", "lists@example.org"} {
		node := &widgets.TreeNode{Label: account, Icon: style.IconMail, Expanded: true}
		for _, f := range []struct {
			name string
			icon style.ToolIcon
		}{
			{"Inbox", style.IconInbox},
			{"Drafts", style.IconPen},
			{"Sent", style.IconSend},
			{"Archives", style.IconArchive},
			{"Spam", style.IconJunk},
			{"Trash", style.IconTrash},
		} {
			node.Children = append(node.Children, &widgets.TreeNode{Label: f.name, Icon: f.icon})
		}
		t.Roots = append(t.Roots, node)
	}
	return t
}

// messageList is the table over the reading pane. It scrolls sideways,
// because a mail list has more columns than a narrow pane can hold and
// the last ones are otherwise unreachable.
func messageList() *widgets.TableView {
	t := widgets.NewTableView(
		[]widgets.TableColumn{
			{Title: "Subject", Width: 320, MinWidth: 140},
			{Title: "Correspondents", Width: 260, MinWidth: 120},
			{Title: "Date", Width: 90, MinWidth: 60},
		},
		len(inbox),
		func(row, col int) string {
			m := inbox[row]
			switch col {
			case 0:
				return m.subject
			case 1:
				return m.from
			}
			return m.date
		}, nil)
	t.Horizontal = true
	t.Selected = 0
	t.CellIcon = func(row, col int) (style.ToolIcon, paintengine2d.Color) {
		if col == 0 && row < 5 {
			return style.IconStar, paintengine2d.Color{}
		}
		return style.IconNone, paintengine2d.Color{}
	}
	return t
}

// readingPane is the message under its own actions.
func readingPane() widget.Component {
	actions := widgets.NewRow(
		widgets.NewToolButton("Reply", style.IconReply, nil),
		widgets.NewToolButton("Reply All", style.IconReplyAll, nil),
		widgets.NewToolButton("Forward", style.IconForward, nil),
		widgets.NewToolButton("Archive", style.IconArchive, nil),
		widgets.NewToolButton("Spam", style.IconJunk, nil),
		widgets.NewToolButton("Delete", style.IconTrash, nil),
	).WithGap(4).WithPad(6)

	head := widgets.NewForm()
	head.AddRow("From", widgets.NewLabel("Parcel Desk <noreply@example.com>"))
	head.AddRow("To", widgets.NewLabel("you@example.com"))
	head.AddRow("Subject", widgets.NewLabel("Your order has shipped"))

	body := widgets.NewLabel(
		"Your parcel left the depot this morning and should reach you " +
			"tomorrow. Track it from the link in your account, or reply to " +
			"this message and we will look it up for you.")
	body.Wrap = true

	col := widgets.NewColumn(actions, widgets.NewPad(8, head), widgets.NewPad(8, body)).WithGap(4)
	return widgets.NewScrollView(col)
}

func statusBar() widget.Component {
	gap := widgets.NewSpacer()
	// Flat here too: a status bar's marks are not buttons you are meant to
	// see, and a framed one at each end reads as pressed.
	row := widgets.NewRow(
		widgets.NewFlatIconButton(style.IconArrowRight, "Activity Manager", nil),
		widgets.NewLabel("Connected"),
		gap,
		widgets.NewFlatIconButton(style.IconBell, "Notifications", nil),
	).WithGap(6).WithPad(4)
	row.AddFlex(gap, 1)
	return row
}
