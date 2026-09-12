package app

import (
	"os"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// NewStatusItem opens a tray / menu-bar icon owned by this application.
// Headless apps get a stub unless UITK_TRAY=fake (tests). A live item
// keeps Run going after the last window is destroyed until Close.
func (a *Application) NewStatusItem(opts platform.StatusItemOptions) (item platform.StatusItem, err error) {
	defer func() {
		if recover() != nil {
			item, err = platform.NewStatusItem(platform.StatusItemOptions{Stub: true, ID: opts.ID, Title: opts.Title})
		}
	}()
	if a != nil && a.headless && strings.ToLower(strings.TrimSpace(os.Getenv("UITK_TRAY"))) != "fake" {
		opts.Stub = true
	}
	if a != nil {
		opts.Dispatch = a.Post
		if opts.OnMenu == nil && hasStatusMenu(opts.Menu) {
			menu := platformCopyMenu(opts.Menu)
			opts.OnMenu = func(x, y int32) {
				a.ShowStatusMenu(x, y, menu)
			}
		}
	}
	item, err = platform.NewStatusItem(opts)
	if err != nil || item == nil {
		item, err = platform.NewStatusItem(platform.StatusItemOptions{Stub: true, ID: opts.ID, Title: opts.Title})
		return item, err
	}
	if a != nil {
		a.mu.Lock()
		a.trays = append(a.trays, item)
		a.mu.Unlock()
	}
	return item, nil
}

// StatusItems is the tray icons still tracked by this application.
func (a *Application) StatusItems() []platform.StatusItem {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]platform.StatusItem, len(a.trays))
	copy(out, a.trays)
	return out
}

func (a *Application) trayHolds() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.trays[:0]
	for _, t := range a.trays {
		if t != nil && t.Alive() {
			out = append(out, t)
		}
	}
	a.trays = out
	return len(out) > 0
}

// StatusMenuFromItems copies widget menu rows into a tray menu.
func StatusMenuFromItems(items []*widgets.MenuItem) []platform.StatusMenuItem {
	out := make([]platform.StatusMenuItem, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		out = append(out, platform.StatusMenuItem{
			Text:      it.Text,
			Disabled:  it.Disabled,
			Separator: it.Separator,
			Checked:   it.Checked,
			Icon:      it.Icon,
			OnClick:   it.OnClick,
		})
	}
	return out
}

// StatusIconFromTool rasterizes a toolkit ToolIcon into a tray image.
func StatusIconFromTool(icon style.ToolIcon, look style.LookAndFeel, size int) platform.StatusIcon {
	if size < 16 {
		size = 22
	}
	img := style.DrawToolIconImage(icon, look, size)
	return platform.StatusIcon{Name: style.ToolIconThemeName(icon), Image: img}
}

func hasStatusMenu(items []platform.StatusMenuItem) bool {
	for _, it := range items {
		if it.Text != "" || it.Separator || it.OnClick != nil {
			return true
		}
	}
	return false
}

func platformCopyMenu(in []platform.StatusMenuItem) []platform.StatusMenuItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]platform.StatusMenuItem, len(in))
	copy(out, in)
	return out
}

// StatusMenuToItems maps tray rows back to toolkit MenuItems (popup chrome).
func StatusMenuToItems(items []platform.StatusMenuItem) []*widgets.MenuItem {
	out := make([]*widgets.MenuItem, 0, len(items))
	for _, it := range items {
		if it.Separator {
			out = append(out, widgets.Sep())
			continue
		}
		out = append(out, &widgets.MenuItem{
			Text:      it.Text,
			Disabled:  it.Disabled,
			Checked:   it.Checked,
			Checkable: it.Checked,
			Icon:      it.Icon,
			OnClick:   it.OnClick,
		})
	}
	return out
}

// ShowStatusMenu opens a toolkit PopupMenu on a live window (Show/Raise if
// the window was hidden to the tray). x,y are SNI root coordinates; they
// are clamped to the surface so a panel click lands near the matching edge.
func (a *Application) ShowStatusMenu(x, y int32, items []platform.StatusMenuItem) {
	if a == nil {
		return
	}
	rows := StatusMenuToItems(items)
	if len(rows) == 0 {
		return
	}
	w := a.preferredStatusWindow()
	if w == nil || w.Closed() {
		return
	}
	if !w.Visible() {
		w.Show()
		w.Raise()
	}
	from := w.Content()
	if from == nil {
		return
	}
	origin := paintengine2d.Pt(float32(x), float32(y))
	if x <= 0 && y <= 0 {
		ww, _ := w.SurfaceSize()
		origin = paintengine2d.Pt(float32(ww)-12, 8)
	}
	widgets.ShowContextMenu(from, origin, rows...)
}

func (a *Application) preferredStatusWindow() *Window {
	var hidden *Window
	for _, w := range a.Windows() {
		if w == nil || w.Closed() {
			continue
		}
		if w.Visible() {
			return w
		}
		if hidden == nil {
			hidden = w
		}
	}
	return hidden
}
