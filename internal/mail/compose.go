package mail

import (
	"fmt"
	"strings"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// ComposeOptions configure a Write / Reply / Forward / Draft window.
type ComposeOptions struct {
	ReplyTo  *Message
	Forward  *Message
	Draft    *Message
	OnChange func() // refresh the 3-pane after send / save
}

// OpenCompose opens a second Window. Headless apps still get a surface.
func OpenCompose(a *app.Application, store Store, opts ComposeOptions) (*app.Window, error) {
	title := "Write: (no subject)"
	if opts.ReplyTo != nil && strings.TrimSpace(opts.ReplyTo.Subject) != "" {
		title = "Write: Re: " + stripRe(opts.ReplyTo.Subject)
	} else if opts.Forward != nil && strings.TrimSpace(opts.Forward.Subject) != "" {
		title = "Write: Fwd: " + strings.TrimSpace(opts.Forward.Subject)
	} else if opts.Draft != nil && strings.TrimSpace(opts.Draft.Subject) != "" {
		title = "Write: " + strings.TrimSpace(opts.Draft.Subject)
	}
	win, err := a.NewWindow(platform.WindowOptions{
		Title: title, Width: 760, Height: 640, MinWidth: 520, MinHeight: 400,
	})
	if err != nil {
		return nil, err
	}
	win.SetContent(ComposeApp(a, win, store, opts))
	return win, nil
}

// ComposeApp is the Write window: From / To / Cc / Bcc / Subject / body.
func ComposeApp(a *app.Application, win *app.Window, store Store, opts ComposeOptions) widget.Component {
	accounts := store.Accounts()
	fromItems := make([]string, 0, len(accounts))
	for _, acct := range accounts {
		fromItems = append(fromItems, fmt.Sprintf("%s <%s>", acct.Name, acct.Address))
	}
	if len(fromItems) == 0 {
		fromItems = []string{"(no account)"}
	}

	to0, cc0, bcc0, subj0, body0 := "", "", "", "", ""
	fromIdx := 0
	draftID := MessageID("")
	if opts.Draft != nil {
		d := opts.Draft
		to0, cc0, bcc0, subj0, body0 = d.To, d.Cc, d.Bcc, d.Subject, d.Body
		draftID = d.ID
		fromIdx = indexFrom(fromItems, d.From)
	} else if opts.ReplyTo != nil {
		m := opts.ReplyTo
		to0 = firstAddr(m.From)
		if strings.TrimSpace(m.Cc) != "" {
			cc0 = m.Cc
		}
		subj0 = "Re: " + stripRe(m.Subject)
		body0 = quoteBody(*m)
		fromIdx = indexFrom(fromItems, m.To)
	} else if opts.Forward != nil {
		m := opts.Forward
		subj0 = "Fwd: " + strings.TrimSpace(m.Subject)
		body0 = forwardBody(*m)
		fromIdx = indexFrom(fromItems, m.To)
	}

	status := widgets.NewStatusBar("Write a message.", "Offline demo", "v"+uitoolkit.Version)
	from := widgets.NewComboBox(fromItems, fromIdx, nil)
	to := widgets.NewTextField(to0, "To", nil)
	cc := widgets.NewTextField(cc0, "Cc", nil)
	bcc := widgets.NewTextField(bcc0, "Bcc", nil)
	subject := widgets.NewTextField(subj0, "Subject", func(s string) {
		t := strings.TrimSpace(s)
		if t == "" {
			t = "(no subject)"
		}
		win.SetTitle("Write: " + t)
	})
	body := widgets.NewTextArea(body0, "Compose in Titillium Web. Source-style quotes stay readable.", nil)
	body.MinRows = 10
	body.Wrap = true

	accountID := func() string {
		i := from.Selected
		if i >= 0 && i < len(accounts) {
			return accounts[i].ID
		}
		if len(accounts) > 0 {
			return accounts[0].ID
		}
		return ""
	}
	fromText := func() string {
		if from.Selected >= 0 && from.Selected < len(fromItems) {
			return fromItems[from.Selected]
		}
		return ""
	}

	collect := func() Message {
		return Message{
			From:    fromText(),
			To:      to.Text,
			Cc:      cc.Text,
			Bcc:     bcc.Text,
			Subject: subject.Text,
			Body:    body.Text,
			Read:    true,
		}
	}

	saveDraft := func() {
		acct := accountID()
		if acct == "" {
			widgets.Warn(win.Content(), "Save Draft", "No account is configured.", nil)
			return
		}
		drafts, ok := specialFolder(store, acct, FolderDrafts)
		if !ok {
			widgets.Warn(win.Content(), "Save Draft", "This account has no Drafts folder.", nil)
			return
		}
		msg := collect()
		var err error
		if draftID != "" {
			err = store.Update(draftID, msg)
		} else {
			draftID, err = store.Append(drafts.ID, msg)
		}
		if err != nil {
			widgets.Warn(win.Content(), "Save Draft", err.Error(), nil)
			return
		}
		status.Set(0, "Saved draft")
		if opts.OnChange != nil {
			opts.OnChange()
		}
	}

	send := func() {
		if strings.TrimSpace(to.Text) == "" {
			widgets.Warn(win.Content(), "Send", "Please enter a To: address.", nil)
			return
		}
		acct := accountID()
		sent, ok := specialFolder(store, acct, FolderSent)
		if !ok {
			widgets.Warn(win.Content(), "Send", "This account has no Sent folder.", nil)
			return
		}
		msg := collect()
		if _, err := store.Append(sent.ID, msg); err != nil {
			widgets.Warn(win.Content(), "Send", err.Error(), nil)
			return
		}
		if draftID != "" {
			_ = store.Delete([]MessageID{draftID})
		}
		if opts.OnChange != nil {
			opts.OnChange()
		}
		widgets.Info(win.Content(), "Sent",
			"Message filed in Sent (demo Store — no SMTP).\nIMAP/SMTP can implement Store.Send later.",
			func() { win.Close() })
	}

	closeWin := func() {
		if strings.TrimSpace(body.Text) == "" && strings.TrimSpace(subject.Text) == "" {
			win.Close()
			return
		}
		widgets.Confirm(win.Content(), "Close write window?",
			"Save this message as a draft?",
			func(yes bool) {
				if yes {
					saveDraft()
				}
				win.Close()
			})
	}

	menubar := widgets.NewMenuBar(
		widgets.NewMenu("&File",
			widgets.ItemAccel("&Send Now", "Ctrl+Enter", send),
			widgets.ItemAccel("Save as &Draft", "Ctrl+S", saveDraft),
			widgets.Sep(),
			widgets.Item("Close", closeWin),
		),
		widgets.NewMenu("&Edit",
			widgets.ItemAccel("Select &All", "Ctrl+A", func() {
				body.SetSelection(0, len([]rune(body.Text)))
			}),
			&widgets.MenuItem{Text: "Undo", Shortcut: "Ctrl+Z", Disabled: true},
		),
		widgets.NewMenu("&View",
			widgets.Item("Body as plain text", func() { status.Set(0, "Plain text (demo)") }),
		),
		widgets.NewMenu("&Insert",
			widgets.Item("File…", func() {
				widgets.Info(win.Content(), "Attach",
					"FileDialog is a stub in v1. Mark HasAttach on a Store message later.", nil)
			}),
		),
		widgets.NewMenu("&Help",
			widgets.Item("About Write", func() {
				widgets.Info(win.Content(), "Write",
					"Compose window on uitoolkit.\nUI: Titillium Web.\nSend files to Sent in the demo Store.", nil)
			}),
		),
	)

	sendBtn := widgets.ToolIconBtn(style.IconNew, "Send", send)
	sendBtn.Tip = "Send Now (demo: file in Sent)"
	draftBtn := widgets.ToolIconBtn(style.IconSave, "Save", saveDraft)
	draftBtn.Tip = "Save as Draft"
	attachBtn := widgets.ToolIconBtn(style.IconOpen, "Attach", func() {
		widgets.Info(win.Content(), "Attach", "Attachment picker is a stub. Use the demo store’s HasAttach flag.", nil)
	})
	attachBtn.Tip = "Attach file (stub)"
	tools := widgets.NewToolBar(sendBtn, draftBtn, widgets.ToolDivider(), attachBtn)

	labeled := func(name string, field widget.Component) widget.Component {
		row := widgets.NewRow(widgets.NewLabel(name), field).WithGap(8).WithAlign(uitoolkit.AlignCenter)
		row.AddFlex(field, 1)
		return row
	}

	fields := widgets.NewColumn(
		labeled("From", from),
		labeled("To", to),
		labeled("Cc", cc),
		labeled("Bcc", bcc),
		labeled("Subject", subject),
	).WithGap(6).WithPad(10)

	chrome := widgets.NewTitleBar("Write", "compose  ·  Titillium Web  ·  v"+uitoolkit.Version)
	bodyPad := widgets.NewPad(8, body)
	root := widgets.NewColumn(menubar, tools, chrome, fields, bodyPad, status).WithGap(0)
	root.AddFlex(bodyPad, 1)
	_ = a
	return root
}

func specialFolder(store Store, accountID string, kind FolderKind) (Folder, bool) {
	for _, f := range store.Folders(accountID) {
		if f.Kind == kind && f.Parent == "" {
			return f, true
		}
	}
	return Folder{}, false
}

func stripRe(s string) string {
	s = strings.TrimSpace(s)
	for {
		low := strings.ToLower(s)
		if strings.HasPrefix(low, "re:") {
			s = strings.TrimSpace(s[3:])
			continue
		}
		if strings.HasPrefix(low, "fwd:") {
			s = strings.TrimSpace(s[4:])
			continue
		}
		if strings.HasPrefix(low, "fw:") {
			s = strings.TrimSpace(s[3:])
			continue
		}
		return s
	}
}

func quoteBody(m Message) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n\nOn %s, %s wrote:\n", m.Date.Format("Mon 2 Jan 2006 15:04"), DisplayName(m.From)))
	for _, line := range strings.Split(m.Body, "\n") {
		b.WriteString("> ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func forwardBody(m Message) string {
	return fmt.Sprintf("\n\n-------- Forwarded Message --------\nFrom: %s\nDate: %s\nSubject: %s\nTo: %s\n\n%s",
		m.From, m.Date.Format("Mon 2 Jan 2006 15:04 MST"), m.Subject, m.To, m.Body)
}

func indexFrom(items []string, from string) int {
	from = strings.TrimSpace(from)
	if from == "" {
		return 0
	}
	want := strings.ToLower(DisplayName(from))
	addr := strings.ToLower(from)
	for i, it := range items {
		low := strings.ToLower(it)
		if low == addr || strings.Contains(low, want) {
			return i
		}
	}
	return 0
}
