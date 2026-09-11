package mail

import (
	"fmt"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// OpenPrefs opens the Preferences / Account Settings stub window.
func OpenPrefs(a *app.Application, cli *Client) (*app.Window, error) {
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Preferences", Width: 640, Height: 480, MinWidth: 420, MinHeight: 320,
	})
	if err != nil {
		return nil, err
	}
	win.SetContent(PrefsApp(a, win, cli))
	return win, nil
}

// PrefsApp lists daemon accounts and backend (no IMAP in this process).
func PrefsApp(a *app.Application, win *app.Window, cli *Client) widget.Component {
	accounts, _ := cli.Accounts()
	st, _ := cli.Status()
	status := widgets.NewStatusBar("Accounts live in mailclientd.", st.Backend, "v"+uitoolkit.Version)

	table := widgets.NewTableView([]widgets.TableColumn{
		{Title: "Name", Width: 180, Sortable: true},
		{Title: "Address", Sortable: true},
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
		"Backend: %s   Socket: %s\nHealth: %s   Accounts: %d\nUITK_MAIL=imap wires IMAP inside mailclientd only.",
		st.Backend, cli.Socket, health, st.Accounts,
	))

	closeBtn := widgets.NewButton("Close", func() { win.Close() })
	addBtn := widgets.NewButton("Add Account…", func() {
		widgets.Info(win.Content(), "Add Account",
			"Account creation is a stub. Demo identities come from MemoryStore.\nIMAP: set UITK_MAIL_USER on mailclientd.", nil)
	})
	tools := widgets.NewRow(addBtn, widgets.NewSpacer(), closeBtn).WithGap(8)

	chrome := widgets.NewTitleBar("Preferences", "accounts  ·  mailclientd  ·  v"+uitoolkit.Version)
	body := widgets.NewColumn(
		widgets.NewTitle("Accounts"),
		info,
		table,
		tools,
	).WithGap(8).WithPad(12)
	body.AddFlex(table, 1)

	root := widgets.NewColumn(chrome, body, status).WithGap(0)
	root.AddFlex(body, 1)
	_ = a
	return root
}
