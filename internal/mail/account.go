package mail

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// FirstRunPrompt is the empty-account modal copy (Yes / No).
const FirstRunPrompt = "There are no accounts, want to add one?"

// OpenAddAccount opens the IMAP/SMTP setup window. Secrets are not written:
// passEnv names an environment variable (default UITK_MAIL_PASS).
func OpenAddAccount(a *app.Application, cli *Client, onSaved func()) (*app.Window, error) {
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Add account", Width: 560, Height: 520, MinWidth: 420, MinHeight: 400,
	})
	if err != nil {
		return nil, err
	}
	win.SetContent(AddAccountApp(win, cli, onSaved))
	return win, nil
}

// AddAccountApp is the first-run / File → Add Account form.
func AddAccountApp(win *app.Window, cli *Client, onSaved func()) widget.Component {
	name := widgets.NewTextField("", "Display name", nil)
	addr := widgets.NewTextField("", "you@example.com", nil)
	imapHost := widgets.NewTextField("imap.example.com:993", "imap.host:993", nil)
	smtpHost := widgets.NewTextField("smtp.example.com:587", "smtp.host:587", nil)
	user := widgets.NewTextField("", "IMAP/SMTP username (defaults to address)", nil)
	passEnv := widgets.NewTextField(EnvPass, "Env var holding the password (not the password)", nil)
	note := widgets.NewLabel(
		"Passwords are never stored in mail.json.\n" +
			"Set the environment variable named above (default UITK_MAIL_PASS),\n" +
			"then Get Messages. Config: " + ConfigPath(),
	)
	status := widgets.NewStatusBar("IMAP + SMTP account.", ConfigPath(), "v"+uitoolkit.Version)

	save := widgets.NewButton("Save", func() {
		cfg := AccountConfig{
			Name:    strings.TrimSpace(name.Text),
			Address: strings.TrimSpace(addr.Text),
			IMAP: ServerConfig{
				Host:    strings.TrimSpace(imapHost.Text),
				User:    strings.TrimSpace(user.Text),
				PassEnv: strings.TrimSpace(passEnv.Text),
			},
			SMTP: ServerConfig{
				Host:    strings.TrimSpace(smtpHost.Text),
				User:    strings.TrimSpace(user.Text),
				PassEnv: strings.TrimSpace(passEnv.Text),
			},
		}
		acct, err := cli.PutAccount(cfg)
		if err != nil {
			widgets.Warn(win.Content(), "Add account", err.Error(), nil)
			return
		}
		if onSaved != nil {
			onSaved()
		}
		widgets.Info(win.Content(), "Account saved",
			fmt.Sprintf("%s <%s>\n\nSet %s in the environment, restart mailclientd if it was started without it, then Get Messages.",
				acct.Name, acct.Address, envVarName(cfg.IMAP.PassEnv)),
			func() { win.Close() })
	})
	save.Primary = true
	cancel := widgets.NewButton("Cancel", func() { win.Close() })

	form := widgets.NewColumn(
		widgets.NewTitle("Add account"),
		labeled("Name", name),
		labeled("Email", addr),
		labeled("IMAP host", imapHost),
		labeled("SMTP host", smtpHost),
		labeled("Username", user),
		labeled("Password env var", passEnv),
		note,
	).WithGap(8)
	tools := widgets.NewRow(widgets.NewSpacer(), cancel, save).WithGap(8)
	chrome := widgets.NewTitleBar("Add account", "passEnv · no secrets in the file")
	pad := widgets.NewPad(12, form)
	root := widgets.NewColumn(chrome, pad, tools, status).WithGap(0)
	root.AddFlex(pad, 1)
	return root
}

func labeled(title string, field widget.Component) widget.Component {
	return widgets.NewColumn(widgets.NewLabel(title), field).WithGap(2)
}
