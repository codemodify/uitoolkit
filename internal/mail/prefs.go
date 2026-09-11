package mail

import (
	"fmt"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// OpenPrefs opens Account Settings / Filters / Appearance.
func OpenPrefs(a *app.Application, cli *Client) (*app.Window, error) {
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Preferences", Width: 720, Height: 560, MinWidth: 480, MinHeight: 360,
	})
	if err != nil {
		return nil, err
	}
	win.SetContent(PrefsApp(a, win, cli))
	return win, nil
}

// OpenFilters is Tools → Message Filters (same window, Filters tab).
func OpenFilters(a *app.Application, cli *Client) (*app.Window, error) {
	return OpenPrefs(a, cli)
}

// PrefsApp is tabbed: Accounts, Identities, Filters, Tags, Appearance.
func PrefsApp(a *app.Application, win *app.Window, cli *Client) widget.Component {
	st, _ := cli.Status()
	status := widgets.NewStatusBar("mailclientd settings.", st.Backend, "v"+uitoolkit.Version)

	accountsTab := prefsAccounts(a, cli, st)
	identsTab := prefsIdentities(cli)
	filtersTab := prefsFilters(cli, win)
	tagsTab := prefsTags(cli)
	appearTab := prefsAppearance(a, win)

	vipTab := prefsVIP(cli)
	smartTab := prefsSmart(cli, win)
	notifyTab := prefsNotify(cli)

	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Accounts", Content: widgets.NewPad(10, accountsTab)},
		widgets.Tab{Title: "Identities", Content: widgets.NewPad(10, identsTab)},
		widgets.Tab{Title: "Filters", Content: widgets.NewPad(10, filtersTab)},
		widgets.Tab{Title: "Smart", Content: widgets.NewPad(10, smartTab)},
		widgets.Tab{Title: "VIP", Content: widgets.NewPad(10, vipTab)},
		widgets.Tab{Title: "Notify", Content: widgets.NewPad(10, notifyTab)},
		widgets.Tab{Title: "Tags", Content: widgets.NewPad(10, tagsTab)},
		widgets.Tab{Title: "Appearance", Content: widgets.NewPad(10, appearTab)},
	)
	closeBtn := widgets.NewButton("Close", func() { win.Close() })
	tools := widgets.NewRow(widgets.NewSpacer(), closeBtn).WithGap(8)
	chrome := widgets.NewTitleBar("Preferences", "accounts · smart · VIP · notify · v"+uitoolkit.Version)
	root := widgets.NewColumn(chrome, tabs, tools, status).WithGap(0)
	root.AddFlex(tabs, 1)
	return root
}

func prefsAccounts(a *app.Application, cli *Client, st DaemonStatus) widget.Component {
	accounts, _ := cli.Accounts()
	table := widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Width: 160, Sortable: true},
		{Title: "Address", Sortable: true},
		{Title: "Transport", Width: 90, Sortable: true},
		{Title: "ID", Width: 80, Sortable: true},
	}, len(accounts), func(row, col int) string {
		if row < 0 || row >= len(accounts) {
			return ""
		}
		a := accounts[row]
		switch col {
		case 1:
			return a.Address
		case 2:
			if a.Transport == "" {
				return st.Backend
			}
			return a.Transport
		case 3:
			return a.ID
		default:
			return a.Name
		}
	}, nil)
	table.Selected = 0
	health := st.Health
	if health == "" {
		health = "ok"
	}
	info := widgets.NewLabel(fmt.Sprintf(
		"Backend: %s   Socket: %s\nHealth: %s   Accounts: %d\nConfig: %s\nPasswords: mail.json (mode 0600, temporary plaintext) or passEnv / OAuth. See docs/mail.md.",
		st.Backend, cli.Socket, health, st.Accounts, ConfigPath(),
	))
	add := widgets.NewButton("Add account…", func() {
		_, _ = OpenAddAccount(a, cli, nil)
	})
	return widgets.NewColumn(
		widgets.NewTitle("Accounts (stores / transports)"),
		info, table, add,
	).WithGap(8)
}

func prefsIdentities(cli *Client) widget.Component {
	idents, _ := cli.Identities("")
	table := widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Width: 160, Sortable: true},
		{Title: "Address", Sortable: true},
		{Title: "Account", Width: 80, Sortable: true},
		{Title: "Default", Width: 70},
	}, len(idents), func(row, col int) string {
		if row < 0 || row >= len(idents) {
			return ""
		}
		id := idents[row]
		switch col {
		case 1:
			return id.Address
		case 2:
			return id.AccountID
		case 3:
			if id.Default {
				return "yes"
			}
			return ""
		default:
			return id.Name
		}
	}, nil)
	table.Selected = 0
	note := widgets.NewLabel("Identities are From personas. Compose picks one; the account maps IMAP/SMTP.")
	return widgets.NewColumn(widgets.NewTitle("Identities"), note, table).WithGap(8)
}

func prefsFilters(cli *Client, win *app.Window) widget.Component {
	rules, _ := cli.Rules()
	var table *widgets.TableView
	refresh := func() {
		rules, _ = cli.Rules()
		if table != nil {
			table.RowCount = len(rules)
			table.Invalidate()
		}
	}
	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "On", Width: 40},
		{Title: "Name", Sortable: true},
		{Title: "When", Width: 200},
		{Title: "Do", Width: 160},
	}, len(rules), func(row, col int) string {
		if row < 0 || row >= len(rules) {
			return ""
		}
		r := rules[row]
		switch col {
		case 0:
			if r.Enabled {
				return "●"
			}
			return "○"
		case 2:
			if len(r.Conditions) == 0 {
				return ""
			}
			c := r.Conditions[0]
			return c.Field + " " + c.Op + " " + c.Value
		case 3:
			if len(r.Actions) == 0 {
				return ""
			}
			return r.Actions[0].Type + " " + r.Actions[0].Tag + string(r.Actions[0].Folder)
		default:
			return r.Name
		}
	}, nil)
	run := widgets.NewButton("Apply rules now", func() {
		n, err := cli.ApplyRules("")
		if err != nil {
			widgets.Warn(win.Content(), "Filters", err.Error(), nil)
			return
		}
		widgets.Info(win.Content(), "Filters", fmt.Sprintf("Applied; %d message(s) changed.", n), nil)
	})
	toggle := widgets.NewButton("Enable / disable selected", func() {
		i := table.Selected
		if i < 0 || i >= len(rules) {
			return
		}
		r := rules[i]
		r.Enabled = !r.Enabled
		_, _ = cli.PutRule(r)
		refresh()
	})
	note := widgets.NewLabel("Sorting Office rules live in mailclientd (MemoryStore or disk). Conditions AND; actions: move, tag, markRead, delete, stop.")
	col := widgets.NewColumn(widgets.NewTitle("Message Filters"), note, table, widgets.NewRow(run, toggle).WithGap(8)).WithGap(8)
	col.AddFlex(table, 1)
	return col
}

func prefsTags(cli *Client) widget.Component {
	tags, _ := cli.Tags()
	table := widgets.NewTableView([]widgets.TableColumn{
		{Title: "Tag", Sortable: true},
		{Title: "Color", Width: 100},
	}, len(tags), func(row, col int) string {
		if row < 0 || row >= len(tags) {
			return ""
		}
		if col == 1 {
			return tags[row].Color
		}
		return tags[row].Name
	}, nil)
	return widgets.NewColumn(
		widgets.NewTitle("Colored tags"),
		widgets.NewLabel("Folder pane → Tags filters the thread list. Message → Tag toggles keywords."),
		table,
	).WithGap(8)
}

func prefsAppearance(a *app.Application, win *app.Window) widget.Component {
	p := loadChromePrefs()
	dens := widgets.NewRadioGroup([]string{"Compact", "Default", "Relaxed"}, densityIndex(p.density()), func(i int) {
		switch i {
		case 0:
			p.Density = "compact"
		case 2:
			p.Density = "relaxed"
		default:
			p.Density = "default"
		}
		saveChromePrefs(p)
	})
	cards := widgets.NewCheckbox("Card view (multi-line list)", p.CardView, func(v bool) {
		p.CardView = v
		saveChromePrefs(p)
	})
	note := widgets.NewLabel("Density and card/table are remembered in mailui.json.\nView menu in the main window applies them immediately.")
	_ = a
	_ = win
	return widgets.NewColumn(widgets.NewTitle("Appearance"), note, dens, cards).WithGap(10)
}

func prefsVIP(cli *Client) widget.Component {
	vips, _ := cli.VIPs()
	table := widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Width: 160, Sortable: true},
		{Title: "Address", Sortable: true},
	}, len(vips), func(row, col int) string {
		if row < 0 || row >= len(vips) {
			return ""
		}
		if col == 1 {
			return vips[row].Address
		}
		return vips[row].Name
	}, nil)
	return widgets.NewColumn(
		widgets.NewTitle("VIP senders"),
		widgets.NewLabel("VIP mail lands in Unified Folders → VIP and can drive VIP-only notifications."),
		table,
	).WithGap(8)
}

func prefsSmart(cli *Client, win *app.Window) widget.Component {
	smart, _ := cli.SmartFolders()
	var table *widgets.TableView
	refresh := func() {
		smart, _ = cli.SmartFolders()
		if table != nil {
			table.RowCount = len(smart)
			table.Invalidate()
		}
	}
	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Sortable: true},
		{Title: "Query", Width: 220},
	}, len(smart), func(row, col int) string {
		if row < 0 || row >= len(smart) {
			return ""
		}
		if col == 1 {
			return smart[row].Filter.Query
		}
		return smart[row].Name
	}, nil)
	del := widgets.NewButton("Delete selected", func() {
		i := table.Selected
		if i < 0 || i >= len(smart) {
			return
		}
		_ = cli.DeleteSmartFolder(smart[i].ID)
		refresh()
	})
	return widgets.NewColumn(
		widgets.NewTitle("Smart / Search folders"),
		widgets.NewLabel("Saved criteria appear in the folder tree. File → New Smart Folder to add one."),
		table, del,
	).WithGap(8)
}

func prefsNotify(cli *Client) widget.Component {
	p, _ := cli.NotifyPrefs()
	en := widgets.NewCheckbox("Notify on new mail", p.Enabled, func(v bool) {
		p.Enabled = v
		_, _ = cli.PutNotifyPrefs(p)
	})
	vip := widgets.NewCheckbox("VIP senders only", p.VIPOnly, func(v bool) {
		p.VIPOnly = v
		_, _ = cli.PutNotifyPrefs(p)
	})
	desk := widgets.NewCheckbox("Desktop notifications (notify-send on Linux)", p.Desktop, func(v bool) {
		p.Desktop = v
		_, _ = cli.PutNotifyPrefs(p)
	})
	return widgets.NewColumn(
		widgets.NewTitle("Notification rules"),
		widgets.NewLabel("mailclientd emits mail.notify and tries notify-send when a display is available."),
		en, vip, desk,
	).WithGap(8)
}

func densityIndex(d style.Density) int {
	switch d {
	case style.DensityCompact:
		return 0
	case style.DensityRelaxed:
		return 2
	default:
		return 1
	}
}
