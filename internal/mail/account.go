package mail

import (
	"fmt"
	"strings"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// FirstRunPrompt is the empty-account modal copy (Yes / No).
const FirstRunPrompt = "There are no accounts, want to add one?\n\nType the IMAP/SMTP password in the form. It is saved in mail.json (mode 0600) until a secret store exists."

// OpenAddAccount opens the IMAP/SMTP / OAuth setup window.
func OpenAddAccount(a *app.Application, cli *Client, onSaved func()) (*app.Window, error) {
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Add account", Width: 600, Height: 640, MinWidth: 460, MinHeight: 480,
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
	pass := widgets.NewPasswordField("IMAP/SMTP password or app password", nil)
	clientID := widgets.NewTextField("", "OAuth client id (required for Google/Microsoft)", nil)
	clientSecret := widgets.NewPasswordField("OAuth client secret (optional for public clients)", nil)
	hint := widgets.NewLabel("Type an email — IMAP/SMTP are guessed from the domain when possible.")
	oauthNote := widgets.NewLabel(
		"Password is stored in mail.json (mode 0600) — temporary plaintext until a secret store.\n" +
			"OAuth: register your own Google Cloud / Azure app.\n" +
			"Redirect: http://127.0.0.1:<port>/oauth/callback (loopback) or device code.\n" +
			"Or set UITK_MAIL_OAUTH_GOOGLE_CLIENT_ID / UITK_MAIL_OAUTH_MS_CLIENT_ID.\n" +
			"Tokens: encrypted files under the data dir (AES-GCM; secret-tool if present).\n" +
			"Optional: UITK_MAIL_PASS / passEnv still works if the password field is empty.",
	)
	status := widgets.NewStatusBar("IMAP + SMTP · typed password or OAuth", ConfigPath(), "v"+uitoolkit.Version)

	applyGuess := func(email string) {
		g := GuessMailHosts(email)
		if g.IMAP != "" {
			imapHost.SetText(g.IMAP)
		}
		if g.SMTP != "" {
			smtpHost.SetText(g.SMTP)
		}
		msg := "Guessed " + g.IMAP + " / " + g.SMTP
		if g.AuthHint != "" {
			msg += "\n" + g.AuthHint
		}
		if g.Provider != "" {
			msg += "  ·  provider " + g.Provider
		}
		hint.SetText(msg)
		if win != nil {
			win.RequestLayout()
		}
	}
	addr.OnChange = func(s string) {
		if strings.Contains(s, "@") {
			applyGuess(s)
		}
	}

	savePass := func() {
		cfg := AccountConfig{
			Name:    strings.TrimSpace(name.Text),
			Address: strings.TrimSpace(addr.Text),
			IMAP: ServerConfig{
				Host: strings.TrimSpace(imapHost.Text),
				User: strings.TrimSpace(user.Text),
				Pass: pass.Text,
			},
			SMTP: ServerConfig{
				Host: strings.TrimSpace(smtpHost.Text),
				User: strings.TrimSpace(user.Text),
				Pass: pass.Text,
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
			fmt.Sprintf("%s <%s>\n\nPassword is in %s (mode 0600). Get Messages to connect.",
				acct.Name, acct.Address, ConfigPath()),
			func() { win.Close() })
	}

	runOAuth := func(provider, flow string) {
		email := strings.TrimSpace(addr.Text)
		if email == "" {
			widgets.Warn(win.Content(), "OAuth", "Enter the email address first.", nil)
			return
		}
		st, err := cli.StartOAuth(provider, email, strings.TrimSpace(name.Text),
			strings.TrimSpace(clientID.Text), strings.TrimSpace(clientSecret.Text), flow)
		if err != nil {
			widgets.Warn(win.Content(), "OAuth", err.Error()+"\n\nSee docs/mail.md — you must register an app and supply a client id.", nil)
			return
		}
		hint.SetText(st.Message + "\n" + st.AuthURL)
		status.Set(0, "Waiting for "+provider+"…")
		go func() {
			deadline := time.Now().Add(3 * time.Minute)
			for time.Now().Before(deadline) {
				poll, err := cli.PollOAuth(st.SessionID)
				if err != nil {
					return
				}
				if poll.Pending {
					time.Sleep(800 * time.Millisecond)
					continue
				}
				if poll.Error != "" {
					return
				}
				if poll.Done {
					if onSaved != nil {
						onSaved()
					}
					return
				}
			}
		}()
		msg := st.Message
		if st.UserCode != "" {
			msg += "\nUser code: " + st.UserCode
		}
		widgets.Info(win.Content(), "Sign in with "+provider, msg+"\n\n"+st.AuthURL, func() {
			if onSaved != nil {
				onSaved()
			}
		})
	}

	google := widgets.NewButton("Sign in with Google", func() { runOAuth("google", "loopback") })
	ms := widgets.NewButton("Sign in with Microsoft", func() { runOAuth("microsoft", "loopback") })
	device := widgets.NewButton("Device code…", func() {
		p := providerForAddress(addr.Text)
		if p == "" {
			p = "google"
		}
		runOAuth(p, "device")
	})
	save := widgets.NewButton("Save account", savePass)
	save.Primary = true
	cancel := widgets.NewButton("Cancel", func() { win.Close() })

	form := widgets.NewColumn(
		widgets.NewTitle("Add account"),
		labeled("Name", name),
		labeled("Email", addr),
		hint,
		labeled("IMAP host", imapHost),
		labeled("SMTP host", smtpHost),
		labeled("Username", user),
		labeled("Password", pass),
		widgets.NewLabel("OAuth (Google / Microsoft) — your client id, not ours"),
		labeled("OAuth client id", clientID),
		labeled("OAuth client secret", clientSecret),
		oauthNote,
	).WithGap(6)
	tools := widgets.NewRow(google, ms, device, widgets.NewSpacer(), cancel, save).WithGap(8)
	chrome := widgets.NewTitleBar("Add account", "auto-guess · typed password or OAuth · mail.json 0600")
	pad := widgets.NewPad(12, form)
	root := widgets.NewColumn(chrome, pad, tools, status).WithGap(0)
	root.AddFlex(pad, 1)
	return root
}

func labeled(title string, field widget.Component) widget.Component {
	return widgets.NewColumn(widgets.NewLabel(title), field).WithGap(2)
}
