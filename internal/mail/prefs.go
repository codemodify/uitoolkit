package mail

import (
	"fmt"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// OpenPrefs opens Accounts / Tags.
func OpenPrefs(a *app.Application, cli *Client, onChange func()) (*app.Window, error) {
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Preferences", Width: 640, Height: 480, MinWidth: 440, MinHeight: 320,
	})
	if err != nil {
		return nil, err
	}
	win.SetContent(PrefsApp(a, win, cli, onChange))
	return win, nil
}

// OpenFilters is the former Tools → Message Filters entry (same window, Tags tab).
func OpenFilters(a *app.Application, cli *Client) (*app.Window, error) {
	return OpenPrefs(a, cli, nil)
}

// PrefsApp is tabbed: Accounts and Tags (sidebar Tags / filter pins share this model).
func PrefsApp(a *app.Application, win *app.Window, cli *Client, onChange func()) widget.Component {
	st, _ := cli.Status()
	status := widgets.NewStatusBar("mailclientd settings.", st.Backend, "v"+uitoolkit.Version)

	accountsTab := prefsAccounts(a, win, cli, st, onChange)
	tagsTab := prefsTags(cli)

	tabs := widgets.NewTabView(
		widgets.Tab{Title: "Accounts", Content: widgets.NewPad(10, accountsTab)},
		widgets.Tab{Title: "Tags", Content: widgets.NewPad(10, tagsTab)},
	)
	closeBtn := widgets.NewButton("Close", func() { win.Close() })
	tools := widgets.NewRow(widgets.NewSpacer(), closeBtn).WithGap(8)
	chrome := widgets.NewTitleBar("Preferences", "accounts · tags · v"+uitoolkit.Version)
	root := widgets.NewColumn(chrome, tabs, tools, status).WithGap(0)
	root.AddFlex(tabs, 1)
	return root
}

func prefsAccounts(a *app.Application, win *app.Window, cli *Client, st DaemonStatus, onChange func()) widget.Component {
	accounts, _ := cli.Accounts()
	var table *widgets.TableView
	refresh := func() {
		accounts, _ = cli.Accounts()
		if table != nil {
			table.RowCount = len(accounts)
			if table.Selected >= len(accounts) {
				table.Selected = 0
			}
			table.Invalidate()
		}
	}
	table = widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Width: 160, Sortable: true},
		{Title: "Address", Sortable: true},
		{Title: "Protocol", Width: 100, Sortable: true},
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
			if lab := ProtocolLabel(a); lab != "" {
				return lab
			}
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
		_, _ = OpenAddAccount(a, cli, func() {
			refresh()
			if onChange != nil {
				onChange()
			}
		})
	})
	remove := widgets.NewButton("Remove account…", func() {
		i := table.Selected
		if i < 0 || i >= len(accounts) {
			widgets.Warn(win.Content(), "Remove account", "Select an account first.", nil)
			return
		}
		acct := accounts[i]
		confirmRemoveAccount(win.Content(), acct, func() {
			if err := cli.DeleteAccount(acct.ID); err != nil {
				widgets.Warn(win.Content(), "Remove account", err.Error(), nil)
				return
			}
			refresh()
			if onChange != nil {
				onChange()
			}
		})
	})
	return widgets.NewColumn(
		widgets.NewTitle("Accounts (stores / transports)"),
		info, table, widgets.NewRow(add, remove).WithGap(8),
	).WithGap(8)
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
		widgets.NewTitle("Tags"),
		widgets.NewLabel("The sidebar Tags group (Unread / Starred / Attachment pins plus colored keywords) is this list. Message → Tag toggles keywords."),
		table,
	).WithGap(8)
}
