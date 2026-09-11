package mail

import (
	"fmt"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
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

// MailApp is the Thunderbird-chrome 3-pane on a demo Store.
func MailApp(a *app.Application, win *app.Window) widget.Component {
	return Open(a, win, NewDemoStore(), AppOptions{ShowFilter: true})
}

// Open builds the Mail chrome against any Store.
func Open(a *app.Application, win *app.Window, store Store, opts AppOptions) widget.Component {
	s := newSession(a, win, store, opts)
	return s.build()
}

type session struct {
	app   *app.Application
	win   *app.Window
	store Store
	opts  AppOptions

	folder   FolderID
	selected []MessageID
	filter   Filter
	sortCol  int
	sortAsc  bool
	online   bool
	rows     []Message
	kind     FolderKind

	table                                      *widgets.TableView
	tree                                       *widgets.TreeView
	preview                                    *widgets.TextArea
	source                                     *widgets.TextArea
	hdrFrom, hdrSubj, hdrDate, hdrTo, hdrExtra *widgets.Label
	status                                     *widgets.StatusBar
	qf                                         *widgets.TextField
	qfBar                                      widget.Component
	unreadTg                                   *widgets.ToolItem
	starTg                                     *widgets.ToolItem
	attachTg                                   *widgets.ToolItem
	chrome                                     *widgets.TitleBar
	folderL                                    *widgets.Label
}

func newSession(a *app.Application, win *app.Window, store Store, opts AppOptions) *session {
	s := &session{
		app: a, win: win, store: store, opts: opts,
		sortCol: 4, sortAsc: false, online: true,
	}
	if inbox, ok := specialFolder(store, firstAccountID(store), FolderInbox); ok {
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
				if inbox, ok := specialFolder(s.store, acct, FolderInbox); ok {
					s.folder = inbox.ID
					s.selected = nil
					s.refreshAll()
				}
			}
			return
		}
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
	tagCombo := widgets.NewComboBox(append([]string{"Tags"}, demoTags()...), 0, func(i int) {
		if i <= 0 {
			s.filter.Tag = ""
		} else {
			s.filter.Tag = demoTags()[i-1]
		}
		s.refreshList()
	})
	pins := widgets.NewToolBar(s.unreadTg, s.starTg, s.attachTg)
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
	headCol := widgets.NewColumn(s.hdrSubj, s.hdrFrom, s.hdrTo, s.hdrDate, s.hdrExtra).WithGap(2).WithPad(8)
	previewCol := widgets.NewColumn(headCol, widgets.NewSeparator(), tabs).WithGap(0)
	previewCol.AddFlex(tabs, 1)

	thread := widgets.NewColumn(s.qfBar, s.table).WithGap(6).WithPad(6)
	thread.AddFlex(s.table, 1)

	sidebar := widgets.NewColumn(
		widgets.NewTitle("Folders"),
		s.tree,
		s.folderL,
	).WithGap(8).WithPad(10)
	sidebar.AddFlex(s.tree, 1)

	var split *widgets.Splitter
	if s.opts.Layout == LayoutClassic {
		right := widgets.NewSplitter(false, thread, previewCol)
		right.Ratio = 0.42
		split = widgets.NewSplitter(true, sidebar, right)
		split.Ratio = 0.22
	} else {
		mid := widgets.NewSplitter(true, thread, previewCol)
		mid.Ratio = 0.48
		split = widgets.NewSplitter(true, sidebar, mid)
		split.Ratio = 0.20
	}

	s.chrome = widgets.NewTitleBar("Mail", s.subtitle())
	root := widgets.NewColumn(s.menuBar(), s.toolBar(), s.chrome, split, s.status).WithGap(0)
	root.AddFlex(split, 1)
	s.refreshAll()
	return root
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
				f, ok := s.store.Folder(s.folder)
				if !ok {
					s.mark("No folder")
					return
				}
				widgets.Info(s.win.Content(), "Folder Properties",
					fmt.Sprintf("%s\n%s\n%d messages  ·  %d unread", f.Name, f.ID, len(s.store.List(f.ID)), s.store.Unread(f.ID)),
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
			widgets.ItemAccel("Next Message", "F8", func() { s.moveSel(1) }),
			widgets.ItemAccel("Previous Message", "F7", func() { s.moveSel(-1) }),
			widgets.Item("Next Unread Message", s.nextUnread),
			widgets.Sep(),
			widgets.Item("Inbox", func() { s.goKind(FolderInbox) }),
			widgets.Item("Drafts", func() { s.goKind(FolderDrafts) }),
			widgets.Item("Sent", func() { s.goKind(FolderSent) }),
			widgets.Item("Junk", func() { s.goKind(FolderJunk) }),
			widgets.Item("Trash", func() { s.goKind(FolderTrash) }),
		),
		widgets.NewMenu("&Message",
			widgets.ItemAccel("&New Message", "Ctrl+N", s.write),
			widgets.ItemAccel("&Reply", "Ctrl+R", s.reply),
			widgets.Item("Reply All", s.reply),
			widgets.ItemAccel("&Forward", "Ctrl+L", s.forward),
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
			widgets.ItemAccel("&Delete", "Del", s.deleteSel),
		),
		widgets.NewMenu("&Tools",
			widgets.Item("Account Settings", func() {
				var b strings.Builder
				b.WriteString("Demo accounts (not IMAP):\n\n")
				for _, a := range s.store.Accounts() {
					fmt.Fprintf(&b, "  %s  <%s>\n", a.Name, a.Address)
				}
				b.WriteString("\nImplement Store on an IMAP client to wire this dialog.")
				widgets.Info(s.win.Content(), "Account Settings", b.String(), nil)
			}),
			widgets.Item("Add-ons and Themes", func() {
				widgets.Info(s.win.Content(), "Add-ons", "LookAndFeel is Dark / Light Classic. No XPI store.", nil)
			}),
			widgets.Item("Error Console", func() { s.mark("Error Console (stub)") }),
			widgets.Item("Activity Manager", func() { s.mark("Activity Manager (stub)") }),
		),
		widgets.NewMenu("&Help",
			widgets.Item("Keyboard", func() {
				widgets.Info(s.win.Content(), "Keyboard",
					"F5 Get Messages · Ctrl+N Write · Ctrl+R Reply · Del Delete\nCtrl+F Quick Filter · F7 / F8 previous / next\nEsc closes tooltip → popup → overlay.", nil)
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
	f, ok := s.store.Folder(s.folder)
	kind := FolderInbox
	if ok {
		kind = f.Kind
	}
	s.kind = kind
	all := s.store.List(s.folder)
	out := make([]Message, 0, len(all))
	for _, m := range all {
		if s.filter.Match(m) {
			out = append(out, m)
		}
	}
	sortMessages(out, s.sortCol, s.sortAsc, kind)
	return out
}

func (s *session) refreshAll() {
	s.rebuildTree()
	s.refreshList()
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
	for _, acct := range s.store.Accounts() {
		node := widgets.NewTreeNode(acct.Address)
		node.Data = acct.ID
		node.Expanded = true
		byParent := map[FolderID][]Folder{}
		for _, f := range s.store.Folders(acct.ID) {
			byParent[f.Parent] = append(byParent[f.Parent], f)
		}
		var addKids func(parent *widgets.TreeNode, pid FolderID)
		addKids = func(parent *widgets.TreeNode, pid FolderID) {
			for _, f := range byParent[pid] {
				label := f.Name
				if n := s.store.Unread(f.ID); n > 0 {
					label = fmt.Sprintf("%s (%d)", f.Name, n)
				}
				n := widgets.NewTreeNode(label)
				n.Data = f.ID
				n.Expanded = f.Kind == FolderArchive || len(byParent[f.ID]) > 0
				addKids(n, f.ID)
				parent.Children = append(parent.Children, n)
				if f.ID == s.folder {
					selected = n
				}
			}
		}
		addKids(node, "")
		roots = append(roots, node)
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
	if m, ok := s.store.Get(id); ok && !m.Read {
		_ = s.store.SetFlags(id, FlagPatch{Read: boolPtr(true)})
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
	return s.store.Get(s.selected[len(s.selected)-1])
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
		if m.HasAttach {
			att := strings.Join(m.Attachments, ", ")
			if att == "" {
				att = "yes"
			}
			if extra != "" {
				extra += "  ·  "
			}
			extra += "Attachments: " + att
		}
		s.hdrExtra.SetText(extra)
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
	unread := s.store.UnreadTotal()
	name := string(s.folder)
	if f, ok := s.store.Folder(s.folder); ok {
		name = f.Name
	}
	sel := ""
	if n := len(s.selected); n > 1 {
		sel = fmt.Sprintf("  ·  %d selected", n)
	}
	s.status.Set(0, fmt.Sprintf("%d unread%s", unread, sel))
	s.status.Set(1, fmt.Sprintf("%s  ·  %d shown", name, len(s.rows)))
	if s.online {
		s.status.Set(2, "Online")
	} else {
		s.status.Set(2, "Offline")
	}
	s.status.Set(3, "v"+uitoolkit.Version)
	if s.folderL != nil {
		s.folderL.SetText(fmt.Sprintf("%d unread in all folders", unread))
	}
}

func (s *session) subtitle() string {
	f, ok := s.store.Folder(s.folder)
	folder := "Mail"
	acct := ""
	if ok {
		folder = f.Name
		for _, a := range s.store.Accounts() {
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
	_, err := OpenCompose(s.app, s.store, ComposeOptions{OnChange: s.refreshAll})
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
	_, err := OpenCompose(s.app, s.store, ComposeOptions{ReplyTo: &cp, OnChange: s.refreshAll})
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
	_, err := OpenCompose(s.app, s.store, ComposeOptions{Forward: &cp, OnChange: s.refreshAll})
	if err != nil {
		widgets.Warn(s.win.Content(), "Forward", err.Error(), nil)
	}
}

func (s *session) getMessages() {
	acct := s.accountID()
	n, err := s.store.Fetch(acct)
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
		_ = s.store.SetFlags(id, FlagPatch{Read: boolPtr(read)})
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
		_ = s.store.SetFlags(id, FlagPatch{Starred: boolPtr(star)})
	}
	s.refreshAll()
}

func (s *session) toggleTag(tag string) {
	if len(s.ids()) == 0 {
		s.mark("No selection")
		return
	}
	for _, id := range s.ids() {
		m, ok := s.store.Get(id)
		if !ok {
			continue
		}
		next := toggleTag(m.Tags, tag)
		_ = s.store.SetFlags(id, FlagPatch{Tags: &next})
	}
	s.refreshAll()
	s.mark("Tag " + tag)
}

func (s *session) deleteSel() {
	ids := s.ids()
	if len(ids) == 0 {
		return
	}
	if err := s.store.Delete(ids); err != nil {
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
	junk, ok := specialFolder(s.store, s.accountID(), FolderJunk)
	if !ok {
		s.mark("No Junk folder")
		return
	}
	if err := s.store.Move(ids, junk.ID); err != nil {
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
	arch, ok := specialFolder(s.store, s.accountID(), FolderArchive)
	if !ok {
		s.mark("No Archives folder")
		return
	}
	if err := s.store.Move(ids, arch.ID); err != nil {
		widgets.Warn(s.win.Content(), "Archive", err.Error(), nil)
		return
	}
	s.selected = nil
	s.refreshAll()
	s.mark("Archived")
}

func (s *session) emptyTrash() {
	trash, ok := specialFolder(s.store, s.accountID(), FolderTrash)
	if !ok {
		s.mark("No Trash")
		return
	}
	list := s.store.List(trash.ID)
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
			_ = s.store.Delete(ids)
			s.selected = nil
			s.refreshAll()
			s.mark("Trash emptied")
		})
}

func (s *session) newFolder() {
	acct := s.accountID()
	ms, ok := s.store.(*MemoryStore)
	if !ok {
		widgets.Info(s.win.Content(), "New Folder",
			"MemoryStore can add folders in-process. An IMAP Store would send CREATE.", nil)
		return
	}
	name := fmt.Sprintf("New Folder %d", len(ms.Folders(acct))+1)
	ms.mu.Lock()
	id := FolderID(fmt.Sprintf("%s/%s", acct, strings.ToLower(strings.ReplaceAll(name, " ", "-"))))
	ms.folders = append(ms.folders, Folder{ID: id, AccountID: acct, Name: name, Kind: FolderCustom})
	ms.mu.Unlock()
	s.folder = id
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
	if f, ok := specialFolder(s.store, s.accountID(), k); ok {
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
			"Dogfood: MenuBar, ToolBar, TreeView, TableView, Splitter,\n"+
			"TabView, TextArea, MessageBox, tooltips, retained scene.\n\n"+
			"Demo Store is in-memory (maildir-ish flags).\n"+
			"IMAP/SMTP: implement mail.Store — see internal/mail/store.go.\n\n"+
			"UI: Titillium Web. Source tab: JetBrains Mono.",
		nil)
}

func (s *session) accountID() string {
	if f, ok := s.store.Folder(s.folder); ok {
		return f.AccountID
	}
	return firstAccountID(s.store)
}

func firstAccountID(store Store) string {
	accts := store.Accounts()
	if len(accts) == 0 {
		return ""
	}
	return accts[0].ID
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
