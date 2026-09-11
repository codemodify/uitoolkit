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
	CardView   bool
	Density    style.Density
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

	chromePrefs ChromePrefs
	cardView    bool
	density     style.Density
	tags        []Tag
	threaded    bool
	hideMuted   bool
	muted       map[string]bool

	table                                      *widgets.TableView
	cards                                      *widgets.CardList
	listStack                                  *widgets.Stack
	tree                                       *widgets.TreeView
	preview                                    *widgets.TextArea
	source                                     *widgets.TextArea
	attachList                                 *widgets.ListView
	askedEmpty                                 bool
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
	p := loadChromePrefs()
	s.chromePrefs = p
	s.cardView = opts.CardView || p.CardView
	s.threaded = p.Threaded
	s.hideMuted = p.HideMute
	s.density = opts.Density
	if s.density == style.DensityDefault && p.Density != "" {
		s.density = p.density()
	}
	if st, err := cli.Status(); err == nil {
		s.backend = st.Backend
		s.online = st.Online
	}
	if tags, err := cli.Tags(); err == nil {
		s.tags = tags
	}
	s.account = firstAccountID(cli)
	if inbox, ok := specialFolderClient(cli, s.account, FolderInbox); ok {
		s.folder = inbox.ID
	}
	return s
}

func (s *session) persistChrome() {
	s.chromePrefs.CardView = s.cardView
	s.chromePrefs.Threaded = s.threaded
	s.chromePrefs.HideMute = s.hideMuted
	s.chromePrefs.Density = s.density.String()
	if s.opts.Layout == LayoutClassic {
		s.chromePrefs.Layout = "classic"
	} else {
		s.chromePrefs.Layout = "vertical"
	}
	s.chromePrefs.Light = s.opts.Light
	saveChromePrefs(s.chromePrefs)
}

func (s *session) applyLook() {
	var look style.LookAndFeel
	if s.opts.Light {
		look = style.LightLook()
	} else {
		look = style.DarkLook()
	}
	look = style.WithDensity(look, s.density)
	s.app.SetLook(look)
}

func (s *session) rebuild() {
	s.persistChrome()
	s.applyLook()
	s.win.SetContent(s.build())
}

func (s *session) build() widget.Component {
	s.applyLook()
	s.status = widgets.NewStatusBar("Ready.", "", "Offline demo", "v"+uitoolkit.Version)
	s.hdrFrom = widgets.NewLabel("")
	s.hdrSubj = widgets.NewTitle("")
	s.hdrDate = widgets.NewLabel("")
	s.hdrTo = widgets.NewLabel("")
	s.hdrExtra = widgets.NewLabel("")
	s.folderL = widgets.NewLabel("Folders")
	s.preview = widgets.NewTextArea("", "Select a message (plain text)", nil)
	s.preview.MinRows = 8
	s.preview.Wrap = true
	s.source = widgets.NewMonoTextArea("", "Raw source", nil)
	s.source.MinRows = 8
	s.source.Wrap = false

	s.table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "★", Width: 28, MinWidth: 24, Sortable: true},
		{Title: "📎", Width: 28, MinWidth: 24, Sortable: true},
		{Title: "Subject", MinWidth: 180, Sortable: true},
		{Title: "Correspondents", Width: 148, MinWidth: 110, Sortable: true},
		{Title: "Date", Width: 108, MinWidth: 88, Sortable: true},
		{Title: "Size", Width: 72, MinWidth: 60, Sortable: true, Align: style.AlignEnd},
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
	s.cards = widgets.NewCardList(0, s.cardAt, func(i int) {
		s.clickRow(i, false)
	})
	s.cards.OnContext = func(i int, p paintengine2d.Point) {
		if i >= 0 && i < len(s.rows) {
			s.clickRow(i, false)
		}
		s.messageMenu(s.cards, p)
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
				if acct == AccountUnified || acct == AccountTags || acct == AccountSmart || acct == AccountCategories {
					return
				}
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
			widgets.Item("Remove Account…", s.removeCurrentAccount),
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
	tagNames := s.tagNames()
	tagCombo := widgets.NewComboBox(append([]string{"Tags"}, tagNames...), 0, func(i int) {
		if i <= 0 {
			s.filter.Tag = ""
		} else {
			s.filter.Tag = tagNames[i-1]
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
	s.attachList.OnSelect = func(i int) {
		s.attachSel = i
		s.openAttachment(i)
	}
	s.applyRowMetrics()
	s.attachList.SetVisible(false)
	headCol := widgets.NewColumn(s.hdrSubj, s.hdrFrom, s.hdrTo, s.hdrDate, s.hdrExtra, s.attachList).WithGap(3).WithPad(10)
	previewCol := widgets.NewColumn(headCol, widgets.NewSeparator(), tabs).WithGap(0)
	previewCol.AddFlex(tabs, 1)

	s.table.SetVisible(!s.cardView)
	s.cards.SetVisible(s.cardView)
	s.listStack = widgets.NewStack(s.table, s.cards)
	thread := widgets.NewColumn(s.qfBar, s.listStack).WithGap(6).WithPad(8)
	thread.AddFlex(s.listStack, 1)
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

	s.applyRowMetrics()
	sidebar := widgets.NewColumn(
		widgets.NewTitle("Account"),
		s.identity,
		widgets.NewTitle("Folders"),
		s.tree,
		s.folderL,
	).WithGap(4).WithPad(6)
	sidebar.AddFlex(s.tree, 1)

	var split *widgets.Splitter
	if s.opts.Layout == LayoutClassic {
		right := widgets.NewSplitter(false, s.center, previewCol)
		right.Ratio = 0.46
		split = widgets.NewSplitter(true, sidebar, right)
		split.Ratio = 0.18
	} else {
		mid := widgets.NewSplitter(true, s.center, previewCol)
		mid.Ratio = 0.58
		split = widgets.NewSplitter(true, sidebar, mid)
		split.Ratio = 0.17
	}

	s.chrome = widgets.NewTitleBar("Mail", s.subtitle())
	root := widgets.NewColumn(s.menuBar(), s.toolBar(), s.chrome, split, s.status).WithGap(0)
	root.AddFlex(split, 1)
	s.refreshAll()
	return wrapShortcutsReady(root, s.handleKey, s.maybeAskAddAccount)
}

func (s *session) menuBar() *widgets.MenuBar {
	layoutClassic := s.opts.Layout == LayoutClassic
	return widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&New Message", "Ctrl+N", s.write),
			widgets.Item("New &Folder…", s.newFolder),
			widgets.Item("Add &Account…", s.openAddAccount),
			widgets.Item("Remove &Account…", s.removeCurrentAccount),
			widgets.Sep(),
			widgets.ItemAccel("&Get New Messages", "F5", s.getMessages),
			widgets.Item("Get Messages for Current Account", s.getMessages),
			widgets.Sep(),
			widgets.Item("Work Offline", s.toggleOnline),
			widgets.Item("New Smart Folder…", s.newSmartFolder),
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
			widgets.CheckItem("&Table view", !s.cardView, func() { s.setCardView(false) }),
			widgets.CheckItem("C&ard view", s.cardView, func() { s.setCardView(true) }),
			widgets.Sep(),
			widgets.CheckItem("&Compact", s.density == style.DensityCompact, func() { s.setDensity(style.DensityCompact) }),
			widgets.CheckItem("&Default density", s.density == style.DensityDefault, func() { s.setDensity(style.DensityDefault) }),
			widgets.CheckItem("&Relaxed", s.density == style.DensityRelaxed, func() { s.setDensity(style.DensityRelaxed) }),
			widgets.Sep(),
			widgets.Item("Sort by Date", func() { s.sortCol, s.sortAsc = 4, false; s.refreshList() }),
			widgets.Item("Sort by Subject", func() { s.sortCol, s.sortAsc = 2, true; s.refreshList() }),
			widgets.Item("Sort by Correspondent", func() { s.sortCol, s.sortAsc = 3, true; s.refreshList() }),
			widgets.CheckItem("&Threaded", s.threaded, func() {
				s.threaded = !s.threaded
				s.persistChrome()
				s.refreshList()
			}),
			widgets.CheckItem("Hide muted threads", s.hideMuted, func() {
				s.hideMuted = !s.hideMuted
				s.persistChrome()
				s.refreshList()
			}),
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
			widgets.Item("Unified Inbox", func() {
				s.central = false
				s.folder = FolderUnifiedInbox
				s.selected = nil
				s.refreshAll()
			}),
			widgets.Item("VIP", func() {
				s.central = false
				s.folder = FolderVIP
				s.selected = nil
				s.refreshAll()
			}),
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
			widgets.Item("Mute Thread", func() { s.muteThread(true) }),
			widgets.Item("Unmute Thread", func() { s.muteThread(false) }),
			widgets.Item("Add sender to VIP", s.addVIP),
			widgets.Item("Move sender to Primary", func() { s.recategorize(CatPrimary) }),
			widgets.Item("Move sender to Other", func() { s.recategorize(CatOther) }),
			widgets.Sep(),
			widgets.ItemAccel("&Delete", "#", s.deleteSel),
		),
		widgets.NewMenu("&Tools",
			widgets.ItemAccel("Account Settings", "Ctrl+,", s.openPrefs),
			widgets.Item("Preferences", s.openPrefs),
			widgets.Item("Message Filters", s.openFilters),
			widgets.Item("Smart Folders", s.openSmartFolders),
			widgets.Item("Apply Filters Now", func() {
				n, err := s.cli.ApplyRules(s.folder)
				if err != nil {
					s.mark(err.Error())
					return
				}
				s.refreshAll()
				s.mark(fmt.Sprintf("Filters applied (%d)", n))
			}),
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
	cards := widgets.ToolToggle("Cards", s.cardView, func() { s.setCardView(!s.cardView) })
	cards.Tip = "Toggle card vs table thread list"
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
		qf, cards, lay,
	)
}

func (s *session) tagPopup(from widget.Component, p paintengine2d.Point) {
	names := s.tagNames()
	items := make([]*widgets.MenuItem, 0, len(names))
	for _, t := range names {
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
		widgets.Item("Mute Thread", func() { s.muteThread(true) }),
		widgets.Item("Add sender to VIP", s.addVIP),
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
		if m.ThreadID != "" && s.muted[m.ThreadID] {
			sub = "🔇 " + sub
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
	if ids, err := s.cli.MutedThreads(); err == nil {
		s.muted = map[string]bool{}
		for _, id := range ids {
			s.muted[id] = true
		}
	}
	if s.hideMuted && len(s.muted) > 0 {
		var keep []Message
		for _, m := range all {
			if m.ThreadID == "" || !s.muted[m.ThreadID] {
				keep = append(keep, m)
			}
		}
		all = keep
	}
	if s.threaded {
		return groupThreaded(all, kind, s.sortCol, s.sortAsc)
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
		s.table.SetVisible(!s.cardView)
		s.table.Invalidate()
	}
	if s.cards != nil {
		s.cards.Count = len(s.rows)
		s.cards.Selected = s.primaryIndex()
		s.cards.SetVisible(s.cardView)
		s.cards.Invalidate()
	}
	s.loadPreview()
	s.refreshStatus()
}

func (s *session) rebuildTree() {
	if s.tree == nil {
		return
	}
	if tags, err := s.cli.Tags(); err == nil {
		s.tags = tags
	}
	var roots []*widgets.TreeNode
	var selected *widgets.TreeNode

	unified := widgets.NewTreeNode("Unified Folders")
	unified.Data = AccountUnified
	unified.Expanded = true
	if vfs, err := s.cli.VirtualFolders(); err == nil {
		for _, f := range vfs {
			if f.AccountID != AccountUnified {
				continue
			}
			label := f.Name
			nUnread, _ := s.cli.Unread(f.ID)
			if nUnread > 0 {
				label = fmt.Sprintf("%s (%d)", f.Name, nUnread)
			}
			n := widgets.NewTreeNode(label)
			n.Data = f.ID
			n.Bold = nUnread > 0
			unified.Children = append(unified.Children, n)
			if f.ID == s.folder && !s.central {
				selected = n
			}
		}
	}
	roots = append(roots, unified)

	smart := widgets.NewTreeNode("Smart Folders")
	smart.Data = AccountSmart
	smart.Expanded = true
	if sfs, err := s.cli.SmartFolders(); err == nil {
		for _, sf := range sfs {
			fid := sf.FolderIDFor()
			label := sf.Name
			nUnread, _ := s.cli.Unread(fid)
			if nUnread > 0 {
				label = fmt.Sprintf("%s (%d)", sf.Name, nUnread)
			}
			n := widgets.NewTreeNode(label)
			n.Data = fid
			n.Bold = nUnread > 0
			smart.Children = append(smart.Children, n)
			if fid == s.folder && !s.central {
				selected = n
			}
		}
	}
	if len(smart.Children) == 0 {
		n := widgets.NewTreeNode("New…")
		n.Data = FolderID("")
		smart.Children = append(smart.Children, n)
	}
	roots = append(roots, smart)

	cats := widgets.NewTreeNode("Categories")
	cats.Data = AccountCategories
	cats.Expanded = true
	if vfs, err := s.cli.VirtualFolders(); err == nil {
		for _, f := range vfs {
			if f.AccountID != AccountCategories {
				continue
			}
			label := f.Name
			nUnread, _ := s.cli.Unread(f.ID)
			if nUnread > 0 {
				label = fmt.Sprintf("%s (%d)", f.Name, nUnread)
			}
			n := widgets.NewTreeNode(label)
			n.Data = f.ID
			n.Bold = nUnread > 0
			cats.Children = append(cats.Children, n)
			if f.ID == s.folder && !s.central {
				selected = n
			}
		}
	}
	roots = append(roots, cats)

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

	tagsNode := widgets.NewTreeNode("Tags")
	tagsNode.Data = AccountTags
	tagsNode.Expanded = true
	for _, t := range s.tags {
		label := t.Name
		fid := TagFolderID(t.Name)
		nUnread, _ := s.cli.Unread(fid)
		if nUnread > 0 {
			label = fmt.Sprintf("%s (%d)", t.Name, nUnread)
		}
		n := widgets.NewTreeNode(label)
		n.Data = fid
		n.Bold = nUnread > 0
		n.Color = ParseHexColor(t.Color)
		tagsNode.Children = append(tagsNode.Children, n)
		if fid == s.folder && !s.central {
			selected = n
		}
	}
	roots = append(roots, tagsNode)

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
	if s.cards != nil {
		s.cards.Selected = i
		s.cards.Invalidate()
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
		s.preview.SetText(DisplayBody(m))
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
		if f.Virtual {
			name = "Virtual · " + f.Name
		}
	}
	sel := ""
	if n := len(s.selected); n > 1 {
		sel = fmt.Sprintf("  ·  %d selected", n)
	}
	s.status.Set(0, fmt.Sprintf("%d unread%s", unread, sel))
	s.status.Set(1, fmt.Sprintf("%s  ·  %d shown", name, len(s.rows)))
	line := s.backendLabel()
	if ops, err := s.cli.Outbox(); err == nil && len(ops) > 0 {
		line += fmt.Sprintf(" · %d queued", len(ops))
	}
	if s.online {
		s.status.Set(2, "Online · "+line)
	} else {
		s.status.Set(2, "Offline · "+line)
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
	res, err := s.cli.Sync(acct)
	if err != nil {
		n, ferr := s.cli.Fetch(acct)
		if ferr != nil {
			widgets.Warn(s.win.Content(), "Get Messages", err.Error(), nil)
			return
		}
		res.New = n
	}
	s.refreshAll()
	if res.New == 0 {
		s.mark("No new messages on " + acct)
		return
	}
	s.mark(fmt.Sprintf("Downloaded %d message(s)", res.New))
}

func (s *session) toggleOnline() {
	s.online = !s.online
	n, err := s.cli.SetOnline(s.online)
	if err != nil {
		s.mark(err.Error())
	}
	s.refreshStatus()
	if s.online {
		s.mark(fmt.Sprintf("Online · flushed %d queued op(s)", n))
		s.refreshAll()
		return
	}
	s.mark("Working offline · send/move/delete/flag queue in the Outbox")
}

func (s *session) muteThread(muted bool) {
	m, ok := s.primary()
	if !ok {
		s.mark("No message")
		return
	}
	tid := m.ThreadID
	if tid == "" {
		tid = ThreadIDOf(m)
	}
	if err := s.cli.MuteThread(tid, muted); err != nil {
		s.mark(err.Error())
		return
	}
	if muted {
		s.mark("Muted thread")
	} else {
		s.mark("Unmuted thread")
	}
	s.refreshList()
	s.rebuildTree()
}

func (s *session) addVIP() {
	m, ok := s.primary()
	if !ok {
		s.mark("No message")
		return
	}
	v, err := s.cli.PutVIP(VIP{Address: extractAddr(m.From), Name: DisplayName(m.From)})
	if err != nil {
		s.mark(err.Error())
		return
	}
	s.mark("VIP: " + v.Address)
	s.rebuildTree()
}

func (s *session) recategorize(cat string) {
	m, ok := s.primary()
	if !ok {
		s.mark("No message")
		return
	}
	if err := s.cli.SetSenderCategory(extractAddr(m.From), cat); err != nil {
		s.mark(err.Error())
		return
	}
	s.mark("Sender → " + cat)
	s.refreshAll()
}

func (s *session) newSmartFolder() {
	win, err := s.app.NewWindow(platform.WindowOptions{
		Title: "Smart Folder", Width: 420, Height: 280, MinWidth: 320, MinHeight: 220,
	})
	if err != nil {
		s.mark(err.Error())
		return
	}
	name := widgets.NewTextField("", "Folder name", nil)
	query := widgets.NewTextField(s.filter.Query, "Search query", nil)
	save := widgets.NewButton("Save", func() {
		sf, err := s.cli.PutSmartFolder(SmartFolder{
			Name:   strings.TrimSpace(name.Text),
			Filter: Filter{Query: strings.TrimSpace(query.Text), Unread: s.filter.Unread, Starred: s.filter.Starred, Attachment: s.filter.Attachment},
		})
		if err != nil {
			widgets.Warn(win.Content(), "Smart Folder", err.Error(), nil)
			return
		}
		s.folder = sf.FolderIDFor()
		s.central = false
		s.refreshAll()
		s.mark("Smart folder: " + sf.Name)
		win.Close()
	})
	save.Primary = true
	cancel := widgets.NewButton("Cancel", func() { win.Close() })
	form := widgets.NewColumn(
		widgets.NewTitle("Saved search"),
		widgets.NewLabel("Name"), name,
		widgets.NewLabel("Query"), query,
		widgets.NewLabel("Unread / starred / attachment pins on the Quick Filter are included."),
		widgets.NewRow(widgets.NewSpacer(), cancel, save).WithGap(8),
	).WithGap(8)
	win.SetContent(widgets.NewPad(12, form))
}

func (s *session) openSmartFolders() {
	s.newSmartFolder()
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
			"No IMAP/SMTP in this process. Empty until you add an account.\n"+
			"MemoryStore dogfood: UITK_MAIL=memory.\n\n"+
			"Message view is plain text only (HTML is stripped).\n"+
			"UI: Titillium Web. Source tab: JetBrains Mono.\n"+
			"See docs/mail.md",
		nil)
}

func (s *session) maybeAskAddAccount(from widget.Component) {
	if s.askedEmpty {
		return
	}
	accts, err := s.cli.Accounts()
	if err != nil || len(accts) > 0 {
		return
	}
	s.askedEmpty = true
	widgets.Confirm(from, "Mail", FirstRunPrompt, func(yes bool) {
		if yes {
			s.openAddAccount()
		}
	})
}

func (s *session) openAddAccount() {
	if _, err := OpenAddAccount(s.app, s.cli, func() {
		s.account = firstAccountID(s.cli)
		if inbox, ok := specialFolderClient(s.cli, s.account, FolderInbox); ok {
			s.folder = inbox.ID
		}
		s.refreshAll()
		s.mark("Account saved — Get Messages to connect")
	}); err != nil {
		widgets.Warn(s.win.Content(), "Add account", err.Error(), nil)
		return
	}
	s.mark("Add account")
}

func (s *session) cardAt(i int) widgets.CardContent {
	if i < 0 || i >= len(s.rows) {
		return widgets.CardContent{}
	}
	m := s.rows[i]
	snip := m.Snippet
	if snip == "" {
		snip = snippetOf(m.Body)
	}
	var badges []widgets.CardBadge
	for _, name := range m.Tags {
		col := paintengine2d.RGB(0.5, 0.5, 0.55)
		if t, ok := tagByName(s.tags, name); ok {
			col = ParseHexColor(t.Color)
		}
		badges = append(badges, widgets.CardBadge{Label: name, Color: col})
	}
	return widgets.CardContent{
		Title:    m.Correspondent(s.kind),
		Subtitle: m.Subject,
		Meta:     formatDate(m.Date, DemoNow),
		Snippet:  snip,
		Badges:   badges,
		Bold:     !m.Read,
		Starred:  m.Starred,
	}
}

func (s *session) setCardView(on bool) {
	s.cardView = on
	s.persistChrome()
	if s.table != nil {
		s.table.SetVisible(!on)
	}
	if s.cards != nil {
		s.cards.SetVisible(on)
	}
	if s.win != nil {
		s.win.RequestLayout()
	}
	s.refreshList()
	if on {
		s.mark("Card view")
	} else {
		s.mark("Table view")
	}
}

func (s *session) setDensity(d style.Density) {
	s.density = d
	s.persistChrome()
	s.rebuild()
}

func (s *session) applyRowMetrics() {
	rh, cardH, treeH := densityRows(s.density)
	if s.table != nil {
		s.table.RowHeight = rh
	}
	if s.cards != nil {
		s.cards.CardHeight = cardH
	}
	if s.tree != nil {
		s.tree.RowHeight = treeH
	}
	if s.attachList != nil {
		s.attachList.RowHeight = treeH
	}
}

func densityRows(d style.Density) (table, card, tree float32) {
	switch d {
	case style.DensityCompact:
		return 22, 56, 20
	case style.DensityRelaxed:
		return 36, 88, 32
	default:
		return 28, 68, 24
	}
}

func (s *session) tagNames() []string {
	if len(s.tags) == 0 {
		if tags, err := s.cli.Tags(); err == nil {
			s.tags = tags
		}
	}
	out := make([]string, 0, len(s.tags))
	for _, t := range s.tags {
		out = append(out, t.Name)
	}
	if len(out) == 0 {
		return demoTags()
	}
	return out
}

func (s *session) openAttachment(i int) {
	m, ok := s.primary()
	if !ok || i < 0 || i >= len(s.attNames) {
		return
	}
	partID := fmt.Sprintf("att-%d", i+1)
	if i < len(m.Parts) {
		partID = m.Parts[i].ID
	}
	p, err := s.cli.OpenPart(m.ID, partID)
	if err != nil {
		s.mark("Attachment: " + s.attNames[i] + " (demo)")
		return
	}
	if p.Path != "" {
		s.mark("Opened " + p.Path + " (xdg-open)")
		return
	}
	s.mark("Attachment: " + s.attNames[i])
}

func (s *session) openFilters() {
	if _, err := OpenFilters(s.app, s.cli); err != nil {
		widgets.Warn(s.win.Content(), "Filters", err.Error(), nil)
		return
	}
	s.mark("Message Filters")
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
	remove := widgets.NewButton("Remove account…", s.removeCurrentAccount)
	return widgets.NewColumn(s.acctTitle, s.acctBody, widgets.NewSeparator(),
		widgets.NewRow(get, write, inbox, prefs, remove).WithGap(8),
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
	name, addr, proto := "Account", "", "IMAP"
	for _, a := range s.accounts() {
		if a.ID == s.account {
			name, addr, proto = a.Name, a.Address, ProtocolLabel(a)
			break
		}
	}
	unread, _ := s.cli.UnreadTotal()
	st, _ := s.cli.Status()
	s.acctTitle.SetText(name)
	s.acctBody.SetText(fmt.Sprintf("%s\n\nProtocol: %s\nIdentity for this window.\nUnread (all folders): %d\nDaemon: %s  ·  %s\nSocket: %s\n\nGet Messages, Write, or open Inbox — retrieve stays in mailclientd.",
		addr, proto, unread, st.Backend, s.backendLabel(), s.cli.Socket))
}

func (s *session) openPrefs() {
	if _, err := OpenPrefs(s.app, s.cli, s.afterAccountsChanged); err != nil {
		widgets.Warn(s.win.Content(), "Preferences", err.Error(), nil)
		return
	}
	s.mark("Preferences")
}

func (s *session) removeCurrentAccount() {
	id := s.account
	var acct Account
	for _, a := range s.accounts() {
		if a.ID == id {
			acct = a
			break
		}
	}
	if acct.ID == "" {
		list := s.accounts()
		if len(list) > 0 {
			acct = list[0]
		}
	}
	if acct.ID == "" {
		widgets.Warn(s.win.Content(), "Remove account", "There is no account to remove.", nil)
		return
	}
	confirmRemoveAccount(s.win.Content(), acct, func() {
		if err := s.cli.DeleteAccount(acct.ID); err != nil {
			widgets.Warn(s.win.Content(), "Remove account", err.Error(), nil)
			return
		}
		s.afterAccountsChanged()
		s.mark("Account removed")
	})
}

func (s *session) afterAccountsChanged() {
	accts := s.accounts()
	still := false
	for _, a := range accts {
		if a.ID == s.account {
			still = true
			break
		}
	}
	if !still {
		s.account = firstAccountID(s.cli)
		s.selected = nil
		if inbox, ok := specialFolderClient(s.cli, s.account, FolderInbox); ok {
			s.folder = inbox.ID
			s.central = false
		} else {
			s.folder = ""
		}
	}
	s.refreshAll()
	if len(accts) == 0 {
		s.askedEmpty = false
		s.maybeAskAddAccount(s.win.Content())
	}
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
		if cl, ok := c.(*widgets.CardList); ok && cl.Count > 0 {
			cl.Selected = 0
			if cl.OnSelect != nil {
				cl.OnSelect(0)
			}
			cl.Invalidate()
		}
		if mb, ok := c.(*widgets.MenuBar); ok && openMenu >= 0 {
			mb.Open(openMenu)
		}
	})
}

// PrepareShotCards forces card view for screenshots.
func PrepareShotCards(w *app.Window) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if cl, ok := c.(*widgets.CardList); ok {
			cl.SetVisible(true)
			if cl.Count > 0 {
				cl.Selected = 0
				if cl.OnSelect != nil {
					cl.OnSelect(0)
				}
			}
			cl.Invalidate()
		}
		if tv, ok := c.(*widgets.TableView); ok && len(tv.Columns) >= 5 {
			tv.SetVisible(false)
		}
	})
}
