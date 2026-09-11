package mail

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// LayoutMode is Thunderbird’s View → Layout.
type LayoutMode int

const (
	// LayoutVertical is folder | thread list | preview (3 columns).
	LayoutVertical LayoutMode = iota
	// LayoutClassic is folder | (thread list above preview).
	LayoutClassic
)

// AppOptions tweak the first build (theme, layout, Quick Filter visibility).
type AppOptions struct {
	Light      bool
	Layout     LayoutMode
	ShowFilter bool
}

// MailApp starts an in-process mailclientd (MemoryStore) and the UI client.
func MailApp(a *app.Application, win *app.Window) widget.Component {
	sock, _, err := StartDemo(context.Background())
	if err != nil {
		return widgets.NewLabel("mailclientd: " + err.Error())
	}
	cli, err := DialWait(sock, 2*time.Second)
	if err != nil {
		return widgets.NewLabel(err.Error())
	}
	return Open(a, win, cli, AppOptions{ShowFilter: true})
}

// Open builds the Mail chrome against a mailclientd Client (no Store / IMAP).
func Open(a *app.Application, win *app.Window, cli *Client, opts AppOptions) widget.Component {
	s := newSession(a, win, cli, opts)
	return s.build()
}

type session struct {
	app  *app.Application
	win  *app.Window
	cli  *Client
	opts AppOptions

	folder    FolderID
	account   string
	central   bool // Account Central instead of the thread list
	selected  []MessageID
	filter    Filter
	sortCol   int
	sortAsc   bool
	online    bool
	rows      []Message
	kind      FolderKind
	backend   string
	attachSel int
	attNames  []string

	table                                      *widgets.TableView
	tree                                       *widgets.TreeView
	preview                                    *widgets.TextArea
	source                                     *widgets.TextArea
	attachList                                 *widgets.ListView
	hdrFrom, hdrSubj, hdrDate, hdrTo, hdrExtra *widgets.Label
	status                                     *widgets.StatusBar
	qf                                         *widgets.TextField
	qfBar                                      widget.Component
	identity                                   *widgets.ComboBox
	unreadTg                                   *widgets.ToolItem
	starTg                                     *widgets.ToolItem
	attachTg                                   *widgets.ToolItem
	senderTg, recipTg, subjTg, bodyTg          *widgets.ToolItem
	chrome                                     *widgets.TitleBar
	folderL                                    *widgets.Label
	acctPanel                                  widget.Component
	acctTitle                                  *widgets.Label
	acctBody                                   *widgets.Label
	thread                                     widget.Component
	center                                     *widgets.Stack
}

func newSession(a *app.Application, win *app.Window, cli *Client, opts AppOptions) *session {
	s := &session{
		app: a, win: win, cli: cli, opts: opts,
		sortCol: 4, sortAsc: false, online: true,
	}
	if st, err := cli.Status(); err == nil {
		s.backend = st.Backend
		s.online = st.Online
	}
	s.account = firstAccountID(cli)
	if inbox, ok := specialFolderClient(cli, s.account, FolderInbox); ok {
		s.folder = inbox.ID
	}
	return s
}

func (s *session) rebuild() {
	if s.opts.Light {
		s.app.SetLook(style.LightLook())
	} else {
		s.app.SetLook(style.DarkLook())
	}
	s.win.SetContent(s.build())
}

func (s *session) build() widget.Component {
	s.status = widgets.NewStatusBar("Ready.", "", "Offline demo", "v"+uitoolkit.Version)
	s.hdrFrom = widgets.NewLabel("")
	s.hdrSubj = widgets.NewTitle("")
	s.hdrDate = widgets.NewLabel("")
	s.hdrTo = widgets.NewLabel("")
	s.hdrExtra = widgets.NewLabel("")
	s.folderL = widgets.NewLabel("Folders")
	s.preview = widgets.NewTextArea("", "Select a message", nil)
	s.preview.MinRows = 8
	s.preview.Wrap = true
	s.source = widgets.NewMonoTextArea("", "Raw source", nil)
	s.source.MinRows = 8
	s.source.Wrap = false

	s.table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "★", Width: 36, Sortable: true},
		{Title: "📎", Width: 36, Sortable: true},
		{Title: "Subject", Sortable: true},
		{Title: "Correspondents", Width: 168, Sortable: true},
		{Title: "Date", Width: 96, Sortable: true},
		{Title: "Size", Width: 64, Sortable: true, Align: style.AlignEnd},
	}, 0, s.cellText, func(i int) {
		s.clickRow(i, false)
	})
	s.table.CellBold = func(row, col int) bool {
		if row < 0 || row >= len(s.rows) {
			return false
		}
		return !s.rows[row].Read
	}
	s.table.OnSort = func(col int, asc bool) {
		s.sortCol, s.sortAsc = col, asc
		s.refreshList()
	}
	s.table.OnContext = func(i int, p paintengine2d.Point) {
		if i >= 0 && i < len(s.rows) {
			s.clickRow(i, false)
		}
		s.messageMenu(s.table, p)
	}

	s.tree = widgets.NewTreeView()
	s.rebuildTree()
	s.tree.OnSelect = func(n *widgets.TreeNode) {
		if n == nil {
			return
		}
		id, ok := n.Data.(FolderID)
		if !ok || id == "" {
			if acct, ok := n.Data.(string); ok {
				s.showAccountCentral(acct)
			}
			return
		}
		s.central = false
		s.folder = id
		s.selected = nil
		s.refreshAll()
	}
	s.tree.OnContext = func(n *widgets.TreeNode, p paintengine2d.Point) {
		if n != nil {
			if id, ok := n.Data.(FolderID); ok && id != "" {
				s.folder = id
				s.selected = nil
				s.refreshAll()
			}
		}
		widgets.ShowContextMenu(s.tree, p,
			widgets.Item("Get Messages", s.getMessages),
			widgets.Item("New Folder…", s.newFolder),
			widgets.Sep(),
			widgets.Item("Empty Trash", s.emptyTrash),
			widgets.Item("Compact Folders", func() { s.mark("Compact Folders (stub)") }),
		)
	}

	s.qf = widgets.NewTextField("", "Quick Filter (subject, people, body)", func(q string) {
		s.filter.Query = q
		s.refreshList()
	})
	s.unreadTg = widgets.ToolToggle("Unread", s.filter.Unread, func() {
		s.filter.Unread = !s.filter.Unread
		s.unreadTg.Down = s.filter.Unread
		s.refreshList()
	})
	s.unreadTg.Tip = "Show only unread"
	s.starTg = widgets.ToolToggle("Starred", s.filter.Starred, func() {
		s.filter.Starred = !s.filter.Starred
		s.starTg.Down = s.filter.Starred
		s.refreshList()
	})
	s.starTg.Tip = "Show only starred"
	s.attachTg = widgets.ToolToggle("Attachment", s.filter.Attachment, func() {
		s.filter.Attachment = !s.filter.Attachment
		s.attachTg.Down = s.filter.Attachment
		s.refreshList()
	})
	s.attachTg.Tip = "Show only with attachment"
	s.senderTg = widgets.ToolToggle("From", s.filter.Sender, func() {
		s.filter.Sender = !s.filter.Sender
		s.senderTg.Down = s.filter.Sender
		s.refreshList()
	})
	s.senderTg.Tip = "Search From"
	s.recipTg = widgets.ToolToggle("To", s.filter.Recipients, func() {
		s.filter.Recipients = !s.filter.Recipients
		s.recipTg.Down = s.filter.Recipients
		s.refreshList()
	})
	s.recipTg.Tip = "Search To / Cc"
	s.subjTg = widgets.ToolToggle("Subject", s.filter.SubjectOnly, func() {
		s.filter.SubjectOnly = !s.filter.SubjectOnly
		s.subjTg.Down = s.filter.SubjectOnly
		s.refreshList()
	})
	s.subjTg.Tip = "Search Subject"
	s.bodyTg = widgets.ToolToggle("Body", s.filter.Body, func() {
		s.filter.Body = !s.filter.Body
		s.bodyTg.Down = s.filter.Body
		s.refreshList()
	})
	s.bodyTg.Tip = "Search body"
	tagCombo := widgets.NewComboBox(append([]string{"Tags"}, demoTags()...), 0, func(i int) {
		if i <= 0 {
			s.filter.Tag = ""
		} else {
			s.filter.Tag = demoTags()[i-1]
		}
		s.refreshList()
	})
	pins := widgets.NewToolBar(s.unreadTg, s.starTg, s.attachTg, widgets.ToolDivider(), s.senderTg, s.recipTg, s.subjTg, s.bodyTg)
	s.qfBar = widgets.NewRow(s.qf, pins, tagCombo).WithGap(8)
	row, _ := s.qfBar.(*widgets.FlexBox)
	if row != nil {
		row.AddFlex(s.qf, 1)
	}
	s.qfBar.SetVisible(s.opts.ShowFilter)

	previewTab := widgets.NewPad(8, s.preview)
	sourceTab := widgets.NewPad(8, s.source)
	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Message", Content: previewTab},
		widgets.Tab{Title: "Source", Content: sourceTab},
	)
	tabs.OnChange = func(i int) {
		if i == 0 {
			s.mark("Message")
		} else {
			s.mark("Source  ·  JetBrains Mono")
		}
	}
	s.attachList = widgets.NewListView(0, func(i int) string {
		if i < 0 || i >= len(s.attNames) {
			return ""
		}
		return "📎  " + s.attNames[i]
	}, func(i int) {
		s.attachSel = i
		if i >= 0 && i < len(s.attNames) {
			s.mark("Attachment: " + s.attNames[i] + " (demo)")
		}
	})
	s.attachList.RowHeight = 24
	s.attachList.SetVisible(false)
	headCol := widgets.NewColumn(s.hdrSubj, s.hdrFrom, s.hdrTo, s.hdrDate, s.hdrExtra, s.attachList).WithGap(2).WithPad(8)
	previewCol := widgets.NewColumn(headCol, widgets.NewSeparator(), tabs).WithGap(0)
	previewCol.AddFlex(tabs, 1)

	thread := widgets.NewColumn(s.qfBar, s.table).WithGap(6).WithPad(6)
	thread.AddFlex(s.table, 1)
	s.thread = thread
	s.acctPanel = s.buildAccountCentral()
	s.acctPanel.SetVisible(false)
	s.center = widgets.NewStack(thread, s.acctPanel)

	idents := s.identityLabels()
	s.identity = widgets.NewComboBox(idents, s.identityIndex(), func(i int) {
		accts := s.accounts()
		if i >= 0 && i < len(accts) {
			s.showAccountCentral(accts[i].ID)
		}
	})

	sidebar := widgets.NewColumn(
		widgets.NewTitle("Account"),
		s.identity,
		widgets.NewTitle("Folders"),
		s.tree,
		s.folderL,
	).WithGap(8).WithPad(10)
	sidebar.AddFlex(s.tree, 1)

	var split *widgets.Splitter
	if s.opts.Layout == LayoutClassic {
		right := widgets.NewSplitter(false, s.center, previewCol)
		right.Ratio = 0.42
		split = widgets.NewSplitter(true, sidebar, right)
		split.Ratio = 0.22
	} else {
		mid := widgets.NewSplitter(true, s.center, previewCol)
		mid.Ratio = 0.48
		split = widgets.NewSplitter(true, sidebar, mid)
		split.Ratio = 0.20
	}

	s.chrome = widgets.NewTitleBar("Mail", s.subtitle())
	root := widgets.NewColumn(s.menuBar(), s.toolBar(), s.chrome, split, s.status).WithGap(0)
	root.AddFlex(split, 1)
	s.refreshAll()
	return wrapShortcuts(root, s.handleKey)
}

func (s *session) menuBar() *widgets.MenuBar {
	layoutClassic := s.opts.Layout == LayoutClassic
	return widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&New Message", "Ctrl+N", s.write),
			widgets.Item("New &Folder…", s.newFolder),
			widgets.Sep(),
			widgets.ItemAccel("&Get New Messages", "F5", s.getMessages),
			widgets.Item("Get Messages for Current Account", s.getMessages),
			widgets.Sep(),
			widgets.Item("Work Offline", func() {
				s.online = !s.online
				s.refreshStatus()
				s.mark("Online state toggled")
			}),
			widgets.Item("Compact Folders", func() { s.mark("Compact Folders (stub)") }),
			widgets.Item("Empty Trash", s.emptyTrash),
			widgets.Sep(),
			widgets.ItemAccel("&Close Window", "Ctrl+W", func() { s.win.Close() }),
			widgets.ItemAccel("&Quit", "Ctrl+Q", func() { s.app.Quit() }),
		),
		widgets.NewMenu("&Edit",
			&widgets.MenuItem{Text: "Undo", Shortcut: "Ctrl+Z", Disabled: true},
			widgets.Sep(),
			widgets.ItemAccel("&Find", "Ctrl+F", func() {
				s.opts.ShowFilter = true
				if s.qfBar != nil {
					s.qfBar.SetVisible(true)
				}
				if s.qf != nil {
					s.win.RequestFocus(s.qf)
				}
				s.win.RequestLayout()
				s.mark("Quick Filter")
			}),
			widgets.ItemAccel("Select &All", "Ctrl+A", s.selectAll),
			widgets.Sep(),
			widgets.Item("Folder Properties", func() {
				f, ok, err := s.cli.GetFolder(s.folder)
				if err != nil || !ok {
					s.mark("No folder")
					return
				}
				unread, _ := s.cli.Unread(f.ID)
				list, _ := s.cli.ListMessages(f.ID, Filter{})
				widgets.Info(s.win.Content(), "Folder Properties",
					fmt.Sprintf("%s\n%s\n%d messages  ·  %d unread", f.Name, f.ID, len(list), unread),
					nil)
			}),
		),
		widgets.NewMenu("&View",
			widgets.CheckItem("Mail Toolbar", true, func() { s.mark("Mail Toolbar") }),
			widgets.CheckItem("Quick Filter Bar", s.opts.ShowFilter, s.toggleFilter),
			widgets.Sep(),
			widgets.CheckItem("&Vertical (3-pane)", !layoutClassic, func() {
				s.opts.Layout = LayoutVertical
				s.rebuild()
			}),
			widgets.CheckItem("&Classic (preview below)", layoutClassic, func() {
				s.opts.Layout = LayoutClassic
				s.rebuild()
			}),
			widgets.Sep(),
			widgets.Item("Sort by Date", func() { s.sortCol, s.sortAsc = 4, false; s.refreshList() }),
			widgets.Item("Sort by Subject", func() { s.sortCol, s.sortAsc = 2, true; s.refreshList() }),
			widgets.Item("Sort by Correspondent", func() { s.sortCol, s.sortAsc = 3, true; s.refreshList() }),
			widgets.Sep(),
			widgets.CheckItem("&Dark", !s.opts.Light, func() {
				s.opts.Light = false
				s.rebuild()
			}),
			widgets.CheckItem("&Light", s.opts.Light, func() {
				s.opts.Light = true
				s.rebuild()
			}),
		),
		widgets.NewMenu("&Go",
			widgets.ItemAccel("Next Message", "n", func() { s.moveSel(1) }),
			widgets.ItemAccel("Previous Message", "p", func() { s.moveSel(-1) }),
			widgets.Item("Next Unread Message", s.nextUnread),
			widgets.Sep(),
			widgets.Item("Inbox", func() { s.goKind(FolderInbox) }),
			widgets.Item("Drafts", func() { s.goKind(FolderDrafts) }),
			widgets.Item("Sent", func() { s.goKind(FolderSent) }),
			widgets.Item("Junk", func() { s.goKind(FolderJunk) }),
			widgets.Item("Trash", func() { s.goKind(FolderTrash) }),
		),
		widgets.NewMenu("&Message",
			widgets.ItemAccel("&New Message", "c", s.write),
			widgets.ItemAccel("&Reply", "r", s.reply),
			widgets.Item("Reply All", s.reply),
			widgets.ItemAccel("&Forward", "f", s.forward),
			widgets.Sep(),
			widgets.Item("Archive", s.archive),
			widgets.ItemAccel("Mark as Read", "M", func() { s.setRead(true) }),
			widgets.Item("Mark as Unread", func() { s.setRead(false) }),
			widgets.Item("Star", s.toggleStar),
			widgets.Sep(),
			widgets.Item("Tag · Important", func() { s.toggleTag("Important") }),
			widgets.Item("Tag · Work", func() { s.toggleTag("Work") }),
			widgets.Item("Tag · Personal", func() { s.toggleTag("Personal") }),
			widgets.Item("Tag · To Do", func() { s.toggleTag("To Do") }),
			widgets.Sep(),
			widgets.ItemAccel("&Delete", "#", s.deleteSel),
		),
		widgets.NewMenu("&Tools",
			widgets.ItemAccel("Account Settings", "Ctrl+,", s.openPrefs),
			widgets.Item("Preferences", s.openPrefs),
			widgets.Item("Add-ons and Themes", func() {
				widgets.Info(s.win.Content(), "Add-ons", "LookAndFeel is Dark / Light Classic. No XPI store.", nil)
			}),
			widgets.Item("Error Console", func() { s.mark("Error Console (stub)") }),
			widgets.Item("Activity Manager", func() { s.mark("Activity Manager (stub)") }),
		),
		widgets.NewMenu("&Help",
			widgets.Item("Keyboard", func() {
				widgets.Info(s.win.Content(), "Keyboard", ShortcutHelp, nil)
			}),
			widgets.Item("About Mail", s.about),
		),
	)
}

func (s *session) toolBar() *widgets.ToolBar {
	get := widgets.ToolIconBtn(style.IconOpen, "Get Messages", s.getMessages)
	get.Tip = "Get new messages for this account (demo Fetch)"
	write := widgets.ToolIconBtn(style.IconNew, "Write", s.write)
	write.Tip = "Write a new message"
	reply := widgets.ToolIconBtn(style.IconUndo, "Reply", s.reply)
	reply.Tip = "Reply"
	fwd := widgets.ToolIconBtn(style.IconRedo, "Forward", s.forward)
	fwd.Tip = "Forward"
	tag := widgets.ToolText("Tag", func() {
		o := widget.DeviceOrigin(s.win.Content())
		s.tagPopup(s.win.Content(), paintengine2d.Pt(o.X+220, o.Y+72))
	})
	tag.Tip = "Tag the selection"
	junk := widgets.ToolText("Junk", s.junk)
	junk.Tip = "Mark as junk"
	arch := widgets.ToolText("Archive", s.archive)
	arch.Tip = "Archive"
	del := widgets.ToolIconBtn(style.IconCut, "Delete", s.deleteSel)
	del.Tip = "Delete (move to Trash)"
	qf := widgets.ToolToggle("Quick Filter", s.opts.ShowFilter, s.toggleFilter)
	qf.Tip = "Show the Quick Filter bar"
	lay := widgets.ToolToggle("Classic", s.opts.Layout == LayoutClassic, func() {
		if s.opts.Layout == LayoutClassic {
			s.opts.Layout = LayoutVertical
		} else {
			s.opts.Layout = LayoutClassic
		}
		s.rebuild()
	})
	lay.Tip = "Toggle classic vs vertical 3-pane"
	return widgets.NewToolBar(
		get, write, widgets.ToolDivider(),
		reply, fwd, tag, widgets.ToolDivider(),
		arch, junk, del, widgets.ToolDivider(),
		qf, lay,
	)
}

func (s *session) tagPopup(from widget.Component, p paintengine2d.Point) {
	items := make([]*widgets.MenuItem, 0, len(demoTags()))
	for _, t := range demoTags() {
		tag := t
		items = append(items, widgets.Item(tag, func() { s.toggleTag(tag) }))
	}
	widgets.ShowContextMenu(from, p, items...)
}

func (s *session) messageMenu(from widget.Component, p paintengine2d.Point) {
	widgets.ShowContextMenu(from, p,
		widgets.Item("Reply", s.reply),
		widgets.Item("Forward", s.forward),
		widgets.Sep(),
		widgets.Item("Mark as Read", func() { s.setRead(true) }),
		widgets.Item("Mark as Unread", func() { s.setRead(false) }),
		widgets.Item("Star", s.toggleStar),
		widgets.Sep(),
		widgets.Item("Tag · Important", func() { s.toggleTag("Important") }),
		widgets.Item("Archive", s.archive),
		widgets.Item("Junk", s.junk),
		widgets.Item("Delete", s.deleteSel),
	)
}

func (s *session) cellText(row, col int) string {
	if row < 0 || row >= len(s.rows) {
		return ""
	}
	m := s.rows[row]
	switch col {
	case 0:
		if m.Starred {
			return "★"
		}
		return ""
	case 1:
		if m.HasAttach {
			return "📎"
		}
		return ""
	case 2:
		sub := m.Subject
		if strings.TrimSpace(sub) == "" {
			sub = "(no subject)"
		}
		if !m.Read {
			return "● " + sub
		}
		return sub
	case 3:
		return m.Correspondent(s.kind)
	case 4:
		return formatDate(m.Date, DemoNow)
	case 5:
		return formatSize(m.Size)
	default:
		return ""
	}
}

func (s *session) visible() []Message {
	f, ok, _ := s.cli.GetFolder(s.folder)
	kind := FolderInbox
	if ok {
		kind = f.Kind
	}
	s.kind = kind
	all, err := s.cli.ListMessages(s.folder, s.filter)
	if err != nil {
		s.mark(err.Error())
		return nil
	}
	sortMessages(all, s.sortCol, s.sortAsc, kind)
	return all
}

func (s *session) refreshAll() {
	s.rebuildTree()
	s.refreshList()
	s.refreshAccount()
	s.showCenter()
	s.refreshStatus()
}

func (s *session) refreshList() {
	s.rows = s.visible()
	if len(s.selected) == 0 && len(s.rows) > 0 {
		s.selected = []MessageID{s.rows[0].ID}
	}
	if s.table != nil {
		s.table.RowCount = len(s.rows)
		s.table.SortCol = s.sortCol
		s.table.SortAsc = s.sortAsc
		s.table.Selected = s.primaryIndex()
		s.table.Invalidate()
	}
	s.loadPreview()
	s.refreshStatus()
}

func (s *session) rebuildTree() {
	if s.tree == nil {
		return
	}
	var roots []*widgets.TreeNode
	var selected *widgets.TreeNode
	acctUnread := 0
	for _, acct := range s.accounts() {
		node := widgets.NewTreeNode(acct.Address)
		node.Data = acct.ID
		node.Expanded = true
		byParent := map[FolderID][]Folder{}
		folders, _ := s.cli.ListFolders(acct.ID)
		for _, f := range folders {
			byParent[f.Parent] = append(byParent[f.Parent], f)
		}
		var addKids func(parent *widgets.TreeNode, pid FolderID)
		addKids = func(parent *widgets.TreeNode, pid FolderID) {
			for _, f := range byParent[pid] {
				label := f.Name
				nUnread, _ := s.cli.Unread(f.ID)
				if nUnread > 0 {
					label = fmt.Sprintf("%s (%d)", f.Name, nUnread)
					acctUnread += nUnread
				}
				n := widgets.NewTreeNode(label)
				n.Data = f.ID
				n.Bold = nUnread > 0
				n.Expanded = f.Kind == FolderArchive || len(byParent[f.ID]) > 0
				addKids(n, f.ID)
				parent.Children = append(parent.Children, n)
				if f.ID == s.folder && !s.central {
					selected = n
				}
			}
		}
		addKids(node, "")
		node.Bold = acctUnread > 0
		if s.central && s.account == acct.ID {
			selected = node
		}
		roots = append(roots, node)
		acctUnread = 0
	}
	s.tree.Roots = roots
	s.tree.Selected = selected
	s.tree.Invalidate()
}

func (s *session) clickRow(i int, add bool) {
	if i < 0 || i >= len(s.rows) {
		return
	}
	id := s.rows[i].ID
	if add {
		if !s.hasSel(id) {
			s.selected = append(s.selected, id)
		}
	} else {
		s.selected = []MessageID{id}
	}
	if s.table != nil {
		s.table.Selected = i
		s.table.Invalidate()
	}
	if m, ok, _ := s.cli.GetMessage(id); ok && !m.Read {
		_ = s.cli.SetFlags(id, FlagPatch{Read: boolPtr(true)})
		s.rows = s.visible()
		s.rebuildTree()
	}
	s.loadPreview()
	s.refreshStatus()
}

func (s *session) primaryIndex() int {
	if len(s.selected) == 0 {
		return -1
	}
	want := s.selected[len(s.selected)-1]
	for i, m := range s.rows {
		if m.ID == want {
			return i
		}
	}
	return -1
}

func (s *session) hasSel(id MessageID) bool {
	for _, x := range s.selected {
		if x == id {
			return true
		}
	}
	return false
}

func (s *session) primary() (Message, bool) {
	if len(s.selected) == 0 {
		return Message{}, false
	}
	m, ok, err := s.cli.GetMessage(s.selected[len(s.selected)-1])
	if err != nil {
		return Message{}, false
	}
	return m, ok
}

func (s *session) loadPreview() {
	m, ok := s.primary()
	if !ok {
		if s.preview != nil {
			s.preview.SetText("")
		}
		if s.source != nil {
			s.source.SetText("")
		}
		if s.hdrSubj != nil {
			s.hdrSubj.SetText("No message selected")
			s.hdrFrom.SetText("")
			s.hdrTo.SetText("")
			s.hdrDate.SetText("")
			s.hdrExtra.SetText("")
		}
		s.attNames = nil
		if s.attachList != nil {
			s.attachList.Count = 0
			s.attachList.SetVisible(false)
		}
		if s.chrome != nil {
			s.chrome.SetSubtitle(s.subtitle())
		}
		return
	}
	if s.hdrSubj != nil {
		s.hdrSubj.SetText(m.Subject)
		s.hdrFrom.SetText("From: " + m.From)
		s.hdrTo.SetText("To: " + m.To)
		s.hdrDate.SetText("Date: " + m.Date.Format("Mon, 02 Jan 2006 15:04 MST"))
		extra := ""
		if len(m.Tags) > 0 {
			extra = "Tags: " + strings.Join(m.Tags, ", ")
		}
		if m.HasAttach && extra != "" {
			extra += "  ·  "
		}
		if m.HasAttach {
			extra += fmt.Sprintf("%d attachment(s)", len(m.Attachments))
		}
		s.hdrExtra.SetText(extra)
	}
	s.attNames = append([]string(nil), m.Attachments...)
	if s.attachList != nil {
		s.attachList.Count = len(s.attNames)
		s.attachList.SetVisible(len(s.attNames) > 0)
		s.attachList.Invalidate()
	}
	if s.preview != nil {
		s.preview.SetText(m.Body)
	}
	if s.source != nil {
		s.source.SetText(rawSource(m))
	}
	if s.chrome != nil {
		s.chrome.SetSubtitle(s.subtitle())
	}
}

func (s *session) refreshStatus() {
	if s.status == nil {
		return
	}
	unread, _ := s.cli.UnreadTotal()
	name := string(s.folder)
	if f, ok, _ := s.cli.GetFolder(s.folder); ok {
		name = f.Name
	}
	sel := ""
	if n := len(s.selected); n > 1 {
		sel = fmt.Sprintf("  ·  %d selected", n)
	}
	s.status.Set(0, fmt.Sprintf("%d unread%s", unread, sel))
	s.status.Set(1, fmt.Sprintf("%s  ·  %d shown", name, len(s.rows)))
	if s.online {
		s.status.Set(2, "Online · "+s.backendLabel())
	} else {
		s.status.Set(2, "Offline · "+s.backendLabel())
	}
	s.status.Set(3, "v"+uitoolkit.Version)
	if s.folderL != nil {
		s.folderL.SetText(fmt.Sprintf("%d unread in all folders", unread))
	}
}

func (s *session) subtitle() string {
	f, ok, _ := s.cli.GetFolder(s.folder)
	folder := "Mail"
	acct := ""
	if ok {
		folder = f.Name
		for _, a := range s.accounts() {
			if a.ID == f.AccountID {
				acct = a.Address
			}
		}
	}
	layout := "vertical"
	if s.opts.Layout == LayoutClassic {
		layout = "classic"
	}
	if acct == "" {
		return fmt.Sprintf("%s  ·  %s  ·  Titillium Web  ·  v%s", folder, layout, uitoolkit.Version)
	}
	return fmt.Sprintf("%s  ·  %s  ·  %s  ·  v%s", acct, folder, layout, uitoolkit.Version)
}

func (s *session) mark(msg string) {
	if s.status != nil {
		s.status.Set(0, msg)
	}
}

func (s *session) ids() []MessageID {
	if len(s.selected) > 0 {
		return append([]MessageID(nil), s.selected...)
	}
	return nil
}

func (s *session) write() {
	_, err := OpenCompose(s.app, s.cli, ComposeOptions{OnChange: s.refreshAll})
	if err != nil {
		widgets.Warn(s.win.Content(), "Write", err.Error(), nil)
		return
	}
	s.mark("Write")
}

func (s *session) reply() {
	m, ok := s.primary()
	if !ok {
		s.mark("No message")
		return
	}
	cp := m.Clone()
	_, err := OpenCompose(s.app, s.cli, ComposeOptions{ReplyTo: &cp, OnChange: s.refreshAll})
	if err != nil {
		widgets.Warn(s.win.Content(), "Reply", err.Error(), nil)
	}
}

func (s *session) forward() {
	m, ok := s.primary()
	if !ok {
		s.mark("No message")
		return
	}
	cp := m.Clone()
	_, err := OpenCompose(s.app, s.cli, ComposeOptions{Forward: &cp, OnChange: s.refreshAll})
	if err != nil {
		widgets.Warn(s.win.Content(), "Forward", err.Error(), nil)
	}
}

func (s *session) getMessages() {
	acct := s.accountID()
	n, err := s.cli.Fetch(acct)
	if err != nil {
		widgets.Warn(s.win.Content(), "Get Messages", err.Error(), nil)
		return
	}
	s.refreshAll()
	if n == 0 {
		s.mark("No new messages on " + acct)
		return
	}
	s.mark(fmt.Sprintf("Downloaded %d message(s)", n))
}

func (s *session) setRead(read bool) {
	for _, id := range s.ids() {
		_ = s.cli.SetFlags(id, FlagPatch{Read: boolPtr(read)})
	}
	s.refreshAll()
}

func (s *session) toggleStar() {
	m, ok := s.primary()
	if !ok {
		return
	}
	star := !m.Starred
	for _, id := range s.ids() {
		_ = s.cli.SetFlags(id, FlagPatch{Starred: boolPtr(star)})
	}
	s.refreshAll()
}

func (s *session) toggleTag(tag string) {
	if len(s.ids()) == 0 {
		s.mark("No selection")
		return
	}
	for _, id := range s.ids() {
		m, ok, err := s.cli.GetMessage(id)
		if err != nil || !ok {
			continue
		}
		next := toggleTag(m.Tags, tag)
		_ = s.cli.SetFlags(id, FlagPatch{Tags: &next})
	}
	s.refreshAll()
	s.mark("Tag " + tag)
}

func (s *session) deleteSel() {
	ids := s.ids()
	if len(ids) == 0 {
		return
	}
	if err := s.cli.Delete(ids); err != nil {
		widgets.Warn(s.win.Content(), "Delete", err.Error(), nil)
		return
	}
	s.selected = nil
	s.refreshAll()
	s.mark("Deleted")
}

func (s *session) junk() {
	ids := s.ids()
	if len(ids) == 0 {
		return
	}
	junk, ok := specialFolderClient(s.cli, s.accountID(), FolderJunk)
	if !ok {
		s.mark("No Junk folder")
		return
	}
	if err := s.cli.Move(ids, junk.ID); err != nil {
		widgets.Warn(s.win.Content(), "Junk", err.Error(), nil)
		return
	}
	s.selected = nil
	s.refreshAll()
	s.mark("Moved to Junk")
}

func (s *session) archive() {
	ids := s.ids()
	if len(ids) == 0 {
		return
	}
	arch, ok := specialFolderClient(s.cli, s.accountID(), FolderArchive)
	if !ok {
		s.mark("No Archives folder")
		return
	}
	if err := s.cli.Move(ids, arch.ID); err != nil {
		widgets.Warn(s.win.Content(), "Archive", err.Error(), nil)
		return
	}
	s.selected = nil
	s.refreshAll()
	s.mark("Archived")
}

func (s *session) emptyTrash() {
	trash, ok := specialFolderClient(s.cli, s.accountID(), FolderTrash)
	if !ok {
		s.mark("No Trash")
		return
	}
	list, _ := s.cli.ListMessages(trash.ID, Filter{})
	if len(list) == 0 {
		s.mark("Trash is empty")
		return
	}
	widgets.Confirm(s.win.Content(), "Empty Trash?",
		fmt.Sprintf("Permanently delete %d messages? (demo store only)", len(list)),
		func(yes bool) {
			if !yes {
				s.mark("Kept trash")
				return
			}
			ids := make([]MessageID, len(list))
			for i, m := range list {
				ids[i] = m.ID
			}
			_ = s.cli.Delete(ids)
			s.selected = nil
			s.refreshAll()
			s.mark("Trash emptied")
		})
}

func (s *session) newFolder() {
	acct := s.accountID()
	folders, _ := s.cli.ListFolders(acct)
	name := fmt.Sprintf("New Folder %d", len(folders)+1)
	f, err := s.cli.CreateFolder(acct, name, "")
	if err != nil {
		widgets.Warn(s.win.Content(), "New Folder", err.Error(), nil)
		return
	}
	s.central = false
	s.folder = f.ID
	s.selected = nil
	s.refreshAll()
	s.mark("Created " + name)
}

func (s *session) selectAll() {
	s.selected = s.selected[:0]
	for _, m := range s.rows {
		s.selected = append(s.selected, m.ID)
	}
	s.refreshStatus()
	s.mark(fmt.Sprintf("%d selected", len(s.selected)))
}

func (s *session) moveSel(delta int) {
	if len(s.rows) == 0 {
		return
	}
	i := s.primaryIndex()
	i += delta
	if i < 0 {
		i = 0
	}
	if i >= len(s.rows) {
		i = len(s.rows) - 1
	}
	s.clickRow(i, false)
}

func (s *session) nextUnread() {
	start := s.primaryIndex() + 1
	for i := 0; i < len(s.rows); i++ {
		j := (start + i) % len(s.rows)
		if !s.rows[j].Read {
			s.clickRow(j, false)
			return
		}
	}
	s.mark("No unread in this folder")
}

func (s *session) goKind(k FolderKind) {
	if f, ok := specialFolderClient(s.cli, s.accountID(), k); ok {
		s.central = false
		s.folder = f.ID
		s.selected = nil
		s.refreshAll()
	}
}

func (s *session) toggleFilter() {
	s.opts.ShowFilter = !s.opts.ShowFilter
	if s.qfBar != nil {
		s.qfBar.SetVisible(s.opts.ShowFilter)
		s.win.RequestLayout()
	}
	s.mark("Quick Filter")
}

func (s *session) about() {
	widgets.Info(s.win.Content(), "About Mail",
		"Mail — Thunderbird chrome on uitoolkit "+uitoolkit.Version+".\n"+
			"mailclientui talks JSON-RPC to mailclientd (Unix socket).\n"+
			"No IMAP in this process. Demo backend is MemoryStore.\n\n"+
			"UI: Titillium Web. Source tab: JetBrains Mono.\n"+
			"See docs/mail.md",
		nil)
}

func (s *session) accountID() string {
	if s.account != "" {
		return s.account
	}
	if f, ok, _ := s.cli.GetFolder(s.folder); ok {
		return f.AccountID
	}
	return firstAccountID(s.cli)
}

func firstAccountID(cli *Client) string {
	accts, err := cli.Accounts()
	if err != nil || len(accts) == 0 {
		return ""
	}
	return accts[0].ID
}

func (s *session) accounts() []Account {
	a, err := s.cli.Accounts()
	if err != nil {
		s.mark(err.Error())
		return nil
	}
	return a
}

func (s *session) identityLabels() []string {
	accts := s.accounts()
	out := make([]string, 0, len(accts))
	for _, a := range accts {
		out = append(out, fmt.Sprintf("%s <%s>", a.Name, a.Address))
	}
	if len(out) == 0 {
		out = []string{"(no account)"}
	}
	return out
}

func (s *session) identityIndex() int {
	accts := s.accounts()
	for i, a := range accts {
		if a.ID == s.account {
			return i
		}
	}
	return 0
}

func (s *session) backendLabel() string {
	if s.backend == "" {
		return "mailclientd"
	}
	return s.backend
}

func (s *session) showAccountCentral(acct string) {
	s.account = acct
	s.central = true
	if inbox, ok := specialFolderClient(s.cli, acct, FolderInbox); ok {
		s.folder = inbox.ID
	}
	s.selected = nil
	s.refreshAll()
	s.showCenter()
}

func (s *session) showCenter() {
	if s.thread != nil {
		s.thread.SetVisible(!s.central)
	}
	if s.acctPanel != nil {
		s.acctPanel.SetVisible(s.central)
	}
	if s.win != nil {
		s.win.RequestLayout()
	}
}

func (s *session) buildAccountCentral() widget.Component {
	s.acctTitle = widgets.NewTitle("Account Central")
	s.acctBody = widgets.NewLabel("Select an identity in the picker or folder tree.")
	get := widgets.NewButton("Get Messages", s.getMessages)
	write := widgets.NewButton("Write", s.write)
	prefs := widgets.NewButton("Account Settings", s.openPrefs)
	inbox := widgets.NewButton("Open Inbox", func() {
		s.goKind(FolderInbox)
	})
	return widgets.NewColumn(s.acctTitle, s.acctBody, widgets.NewSeparator(),
		widgets.NewRow(get, write, inbox, prefs).WithGap(8),
	).WithGap(10).WithPad(16)
}

func (s *session) refreshAccount() {
	if s.identity != nil {
		s.identity.Items = s.identityLabels()
		s.identity.Selected = s.identityIndex()
		s.identity.Invalidate()
	}
	if s.acctTitle == nil {
		return
	}
	name, addr := "Account", ""
	for _, a := range s.accounts() {
		if a.ID == s.account {
			name, addr = a.Name, a.Address
			break
		}
	}
	unread, _ := s.cli.UnreadTotal()
	st, _ := s.cli.Status()
	s.acctTitle.SetText(name)
	s.acctBody.SetText(fmt.Sprintf("%s\n\nIdentity for this window.\nUnread (all folders): %d\nDaemon: %s  ·  %s\nSocket: %s\n\nGet Messages, Write, or open Inbox — IMAP stays in mailclientd.",
		addr, unread, st.Backend, s.backendLabel(), s.cli.Socket))
}

func (s *session) openPrefs() {
	if _, err := OpenPrefs(s.app, s.cli); err != nil {
		widgets.Warn(s.win.Content(), "Preferences", err.Error(), nil)
		return
	}
	s.mark("Preferences")
}

func (s *session) handleKey(e widget.KeyEvent) bool {
	if isTextFocus(s.win.Focus()) {
		return false
	}
	if e.Mods.Ctrl() || e.Mods.Alt() {
		if e.Mods.Ctrl() && e.Key == platform.KeyComma {
			s.openPrefs()
			return true
		}
		return false
	}
	if isHashDelete(e) || e.Key == platform.KeyDelete {
		s.deleteSel()
		return true
	}
	switch e.Key {
	case platform.KeyN:
		s.moveSel(1)
		return true
	case platform.KeyP:
		s.moveSel(-1)
		return true
	case platform.KeyR:
		s.reply()
		return true
	case platform.KeyF:
		s.forward()
		return true
	case platform.KeyC:
		s.write()
		return true
	case platform.KeyM:
		s.setRead(true)
		return true
	}
	return false
}

func specialFolderClient(cli *Client, accountID string, kind FolderKind) (Folder, bool) {
	folders, err := cli.ListFolders(accountID)
	if err != nil {
		return Folder{}, false
	}
	for _, f := range folders {
		if f.Kind == kind && f.Parent == "" {
			return f, true
		}
	}
	return Folder{}, false
}

func demoTags() []string {
	return []string{"Important", "Work", "Personal", "To Do", "Later"}
}

func rawSource(m Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\n", m.From)
	fmt.Fprintf(&b, "To: %s\n", m.To)
	if m.Cc != "" {
		fmt.Fprintf(&b, "Cc: %s\n", m.Cc)
	}
	fmt.Fprintf(&b, "Date: %s\n", m.Date.Format(timeRFC))
	fmt.Fprintf(&b, "Subject: %s\n", m.Subject)
	if m.Starred {
		b.WriteString("X-Flag: flagged\n")
	}
	if !m.Read {
		b.WriteString("X-Flag: unseen\n")
	}
	if len(m.Tags) > 0 {
		fmt.Fprintf(&b, "X-Tags: %s\n", strings.Join(m.Tags, ", "))
	}
	b.WriteByte('\n')
	b.WriteString(m.Body)
	return b.String()
}

const timeRFC = "Mon, 02 Jan 2006 15:04:05 -0700"

// PrepareShot selects Ada’s Inbox welcome message and optionally opens File.
func PrepareShot(w *app.Window, openMenu int) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && len(tv.Columns) >= 5 {
			tv.Selected = 0
			if tv.OnSelect != nil {
				tv.OnSelect(0)
			}
			tv.Invalidate()
		}
		if mb, ok := c.(*widgets.MenuBar); ok && openMenu >= 0 {
			mb.Open(openMenu)
		}
	})
}
