package mail

import (
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func (s *session) attachTray() {
	if s == nil || s.app == nil || s.tray != nil {
		return
	}
	look := s.app.Look()
	item, err := s.app.NewStatusItem(platform.StatusItemOptions{
		ID:         "mailclientui",
		Title:      "Mail",
		Tooltip:    "Mail",
		MenuChrome: platform.HostMenu,
		Icon:       app.StatusIconFromTool(style.IconMail, look, 22),
		Menu: app.StatusMenuFromItems([]*widgets.MenuItem{
			widgets.ItemIcon(style.IconMail, "Show Mail", s.showMain),
			widgets.Sep(),
			widgets.Item("Quit", s.quitFromTray),
		}),
		OnClick:       s.showMain,
		OnNotifyClick: s.showMain,
	})
	if err != nil || item == nil {
		return
	}
	s.tray = item
	if s.win != nil && item.Alive() {
		s.win.SetCloseHides(true)
	}
	if s.cli != nil {
		s.cli.OnEvent(s.onDaemonEvent)
	}
}

func (s *session) showMain() {
	if s == nil || s.app == nil {
		return
	}
	if s.win == nil || s.win.Closed() {
		win, err := s.app.NewWindow(platform.WindowOptions{
			Title: "Mail", Width: 1280, Height: 800, MinWidth: 860, MinHeight: 560,
		})
		if err != nil {
			return
		}
		s.win = win
		win.SetContent(s.build())
		if s.tray != nil && s.tray.Alive() {
			win.SetCloseHides(true)
		}
	}
	s.win.Show()
	s.win.Raise()
}

func (s *session) quitFromTray() {
	if s.win != nil {
		s.win.SetCloseHides(false)
		s.win.Close()
	}
	if s.tray != nil {
		_ = s.tray.Close()
		s.tray = nil
	}
	if s.app != nil {
		s.app.Quit()
	}
}

func (s *session) onDaemonEvent(ev Event) {
	if ev.Method != EventNotify || s.tray == nil {
		return
	}
	title := ev.Title
	if title == "" {
		title = "New mail"
	}
	_ = s.tray.Notify(platform.Notification{
		Title:   title,
		Body:    ev.Body,
		OnClick: s.showMain,
	})
}
