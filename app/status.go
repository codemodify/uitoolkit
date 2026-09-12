package app

import (
	"log"
	"os"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
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

// ShowStatusMenu opens a toolkit PopupMenu on a dedicated top-level
// status-menu window so the menu is visible when the main window is
// hidden or minimized to the tray. x,y are SNI root/screen coordinates;
// X11 places the popup there (typically above a bottom panel). Wayland
// cannot position a toplevel; the compositor still maps a small window.
func (a *Application) ShowStatusMenu(x, y int32, items []platform.StatusMenuItem) {
	if a == nil {
		return
	}
	rows := StatusMenuToItems(items)
	if len(rows) == 0 {
		return
	}
	a.dismissStatusMenu()
	mw, mh := measureStatusMenu(a.Look(), a.Scale(), rows)
	px, py := statusMenuScreenPos(x, y, mw, mh)
	opts := platform.WindowOptions{
		Title:     " ",
		Width:     mw,
		Height:    mh,
		MinWidth:  1,
		MinHeight: 1,
		Popup:     true,
		X:         px,
		Y:         py,
	}
	w, err := a.NewWindow(opts)
	if err != nil || w == nil {
		a.showStatusMenuOnPreferred(x, y, rows)
		return
	}
	w.statusMenu = true
	a.statusMenu = w
	from := widgets.NewLabel("")
	w.SetContent(from)
	pop := widgets.NewPopupMenu(rows...)
	pop.OnDismiss = func() {
		if a.statusMenu == w {
			a.statusMenu = nil
		}
		if !w.Closed() {
			w.Close()
		}
	}
	widget.PreparePopup(from, pop)
	pop.Arrange(paintengine2d.XYWH(0, 0, float32(mw), float32(mh)))
	if !widget.ShowPopup(from, pop) {
		trayStatusLog("ShowStatusMenu popup layer failed; falling back to main window")
		w.Close()
		a.statusMenu = nil
		a.showStatusMenuOnPreferred(x, y, rows)
		return
	}
	pop.RequestFocus()
	w.Show()
	w.Raise()
	if px != 0 || py != 0 {
		platform.MoveSurface(w.Surface(), px, py)
	}
	trayStatusLog("ShowStatusMenu screen=%d,%d place=%d,%d size=%dx%d popupWin visible=%v",
		x, y, px, py, mw, mh, w.Visible())
}

func (a *Application) showStatusMenuOnPreferred(x, y int32, rows []*widgets.MenuItem) {
	w := a.preferredStatusWindow()
	if w == nil || w.Closed() {
		trayStatusLog("ShowStatusMenu no window")
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
	ww, hh := w.SurfaceSize()
	origin := statusMenuOrigin(ww, hh, x, y)
	trayStatusLog("ShowStatusMenu fallback screen=%d,%d window=%dx%d origin=%.0f,%.0f visible=%v",
		x, y, ww, hh, origin.X, origin.Y, w.Visible())
	widgets.ShowContextMenu(from, origin, rows...)
}

func (a *Application) dismissStatusMenu() {
	if a == nil {
		return
	}
	w := a.statusMenu
	a.statusMenu = nil
	if w == nil || w.Closed() {
		return
	}
	if w.Popup() != nil {
		w.DismissPopup()
	}
	if !w.Closed() {
		w.Close()
	}
}

func measureStatusMenu(look style.LookAndFeel, scale float32, items []*widgets.MenuItem) (int, int) {
	pop := widgets.NewPopupMenu(items...)
	pop.SetHost(&statusMeasureHost{look: look, scale: scale})
	sz := pop.Measure(layout.Unbounded())
	w, h := int(sz.X+0.5), int(sz.Y+0.5)
	if w < 80 {
		w = 80
	}
	if h < 24 {
		h = 24
	}
	return w, h
}

type statusMeasureHost struct {
	look  style.LookAndFeel
	scale float32
}

func (h *statusMeasureHost) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *statusMeasureHost) RequestFocus(widget.Component)                  {}
func (h *statusMeasureHost) Focus() widget.Component                        { return nil }
func (h *statusMeasureHost) Scale() float32 {
	if h == nil || h.scale <= 0 {
		return 1
	}
	return h.scale
}
func (h *statusMeasureHost) Look() style.LookAndFeel {
	if h != nil && h.look != nil {
		return h.look
	}
	return style.DarkLook()
}
func (h *statusMeasureHost) RequestLayout() {}

func trayStatusLog(format string, args ...any) {
	if os.Getenv("UITK_TRAY_DEBUG") == "" {
		return
	}
	log.Printf("uitk tray: "+format, args...)
}

// statusMenuOrigin maps SNI ContextMenu coordinates into window space.
// Screen/root coords outside the surface become the bottom-right corner
// so the popup stays visible (Plasma panel clicks are typically 1k–4k).
func statusMenuOrigin(winW, winH int, screenX, screenY int32) paintengine2d.Point {
	const inset = 8
	ww, hh := float32(winW), float32(winH)
	if ww < 1 {
		ww = 400
	}
	if hh < 1 {
		hh = 240
	}
	if screenX > 0 && screenY > 0 && int(screenX) < winW && int(screenY) < winH {
		return paintengine2d.Pt(float32(screenX), float32(screenY))
	}
	return paintengine2d.Pt(ww-inset, hh-inset)
}

// statusMenuScreenPos maps SNI click coords to the top-left of a
// menu-sized popup. Clicks on a bottom panel (large Y) open above.
func statusMenuScreenPos(screenX, screenY int32, menuW, menuH int) (int, int) {
	if screenX == 0 && screenY == 0 {
		return 0, 0
	}
	x, y := int(screenX), int(screenY)
	if menuH > 0 && y >= menuH {
		y -= menuH
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

func (a *Application) preferredStatusWindow() *Window {
	var hidden *Window
	for _, w := range a.Windows() {
		if w == nil || w.Closed() || w.statusMenu {
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
