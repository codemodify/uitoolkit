package app

import (
	"os"
	"strings"

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
	name := style.ToolIconName(icon)
	if name == "" {
		name = "application-default-icon"
	}
	return platform.StatusIcon{Name: name, Image: img}
}
