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
//	go run ./examples/uitoolkit-sample-look-like-thunderbird
//	go run ./examples/uitoolkit-sample-look-like-thunderbird -headless
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

var inbox = []message{
	{"Give 10% off a Pixel phone, get $50 back.", "Google Pixel <noreply@google.com>", "2:08 PM"},
	{"You have $25 in Uber Cash waiting in your account", "Uber <uber@uber.com>", "1:52 PM"},
	{"Final Hours: Go Truly Wireless", "First Backer <news@first-backer.com>", "1:20 PM"},
	{"Endless \"DADA\" Infinite Slider", "Kickstarter <contact@backthebest.com>", "1:04 PM"},
	{"Unlock savings with GoDaddy Payments.", "GoDaddy <donotreply@godaddy.com>", "12:41 PM"},
	{"Your Screen, Finally Moves With You", "First Backer <news@first-backer.com>", "9/29"},
	{"Educational: Assemble A Smartphone Yourself", "Kickstarter <contact@backthebest.com>", "9/29"},
}

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write thunderbird.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Inbox — uitoolkit Mail", Width: 1180, Height: 720,
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
	actions := widgets.NewRow(
		widgets.NewIconButton(style.IconColumns, "Toggle Folder Pane", nil),
		widgets.NewToolButton("Get Messages", style.IconDownload, nil),
		widgets.NewToolButton("Write", style.IconPen, nil),
		widgets.NewToolButton("Delete", style.IconTrash, nil),
	).WithGap(4)

	search := widgets.NewTextField("", "Search…   Ctrl+K", nil)

	head := widgets.NewHeaderBar(
		[]widget.Component{actions},
		search,
		[]widget.Component{
			widgets.NewMenuButton(style.IconMenu, "Application Menu",
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
	for _, account := range []string{"j9@nchip.com", "john@nchip.com", "ping@sccllc.com"} {
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
	head.AddRow("From", widgets.NewLabel("Google Pixel <googlepixel-noreply@google.com>"))
	head.AddRow("To", widgets.NewLabel("you@example.com"))
	head.AddRow("Subject", widgets.NewLabel("Give 10% off a Pixel phone, get $50 back."))

	body := widgets.NewLabel(
		"They save on a new phone, you get Google Store credit. " +
			"Send your exclusive code to a friend and you get $50 in Google Store credit.")
	body.Wrap = true

	col := widgets.NewColumn(actions, widgets.NewPad(8, head), widgets.NewPad(8, body)).WithGap(4)
	return widgets.NewScrollView(col)
}

func statusBar() widget.Component {
	gap := widgets.NewSpacer()
	row := widgets.NewRow(
		widgets.NewIconButton(style.IconArrowRight, "Activity Manager", nil),
		widgets.NewLabel("Connected"),
		gap,
		widgets.NewIconButton(style.IconBell, "Notifications", nil),
	).WithGap(6).WithPad(4)
	row.AddFlex(gap, 1)
	return row
}
