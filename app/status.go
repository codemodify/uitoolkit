package app

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// NewStatusItem opens a tray / menu-bar icon owned by this application.
// Default chrome is HostMenu (native dbusmenu on Linux). ToolkitMenu
// attaches OnMenu → ShowStatusMenu. Headless apps get a stub unless
// UITK_TRAY=fake (tests). A live item keeps Run going after the last
// window is destroyed until Close.
func (a *Application) NewStatusItem(opts platform.StatusItemOptions) (item platform.StatusItem, err error) {
	defer func() {
		if recover() != nil {
			item, err = platform.NewStatusItem(platform.StatusItemOptions{Stub: true, ID: opts.ID, Title: opts.Title})
		}
	}()
	if a != nil && a.headless && strings.ToLower(strings.TrimSpace(os.Getenv("UITK_TRAY"))) != "fake" {
		opts.Stub = true
	}
	var holder *statusMenuHolder
	if a != nil {
		opts.Dispatch = a.Post
		chrome := demoteStatusMenuChrome(&opts)
		if opts.OnMenu == nil && hasStatusMenu(opts.Menu) && chrome == platform.ToolkitMenu {
			holder = &statusMenuHolder{items: platformCopyMenu(opts.Menu)}
			opts.OnMenu = func(x, y int32) {
				a.ShowStatusMenu(x, y, holder.Items())
			}
		}
	}
	item, err = platform.NewStatusItem(opts)
	if err != nil || item == nil {
		item, err = platform.NewStatusItem(platform.StatusItemOptions{Stub: true, ID: opts.ID, Title: opts.Title})
		return item, err
	}
	if holder != nil {
		item = &trackingStatusItem{StatusItem: item, holder: holder}
	}
	if a != nil {
		a.mu.Lock()
		a.trays = append(a.trays, item)
		a.mu.Unlock()
	}
	return item, nil
}

// demoteStatusMenuChrome settles the chrome this item will really have and
// writes it back into opts, so an item demoted from ToolkitMenu to
// HostMenu exports a real dbusmenu instead of the Menu=/NO_DBUSMENU
// sentinel that asks the host to call ContextMenu. It returns the chrome.
//
// Only a demotion is written back. A HostMenu item on a desktop with no
// dbusmenu (Windows) is served by OnMenu without changing what it asked
// for, as it always was.
func demoteStatusMenuChrome(opts *platform.StatusItemOptions) platform.StatusMenuChrome {
	chrome, why := StatusMenuChromeFor(opts.MenuChrome)
	if opts.MenuChrome == platform.ToolkitMenu && chrome != platform.ToolkitMenu {
		// The app asked for its own chrome and cannot have it. It is
		// entitled to know, once, and not only under UITK_TRAY_DEBUG:
		// this changes what its menu looks like.
		opts.MenuChrome = chrome
		trayChromeFallbackOnce.Do(func() {
			log.Printf("uitk tray: ToolkitMenu asked for, HostMenu used: %s", why)
		})
	}
	return chrome
}

// trayChromeFallbackOnce keeps the demotion notice to one line per
// process, however many tray items the app opens.
var trayChromeFallbackOnce sync.Once

// StatusMenuChromeFor is the tray menu chrome an item that asked for want
// will really get here, and one line saying why. It is what
// [Application.NewStatusItem] decides with, and what a readout should show
// instead of the chrome the app asked for.
//
// ToolkitMenu draws the menu itself, in the app's theme, on a window of
// its own at the point the host clicked. That needs a window the client
// can place. X11 and Windows place windows; a Wayland compositor only
// does where it offers zwlr_layer_shell_v1 (KDE, sway, Hyprland,
// wayfire). On one that does not — GNOME/Mutter — a ToolkitMenu window
// lands wherever the compositor drops it, which for KWin was the middle
// of the screen. A menu in the wrong place is worse than a menu that
// looks like the desktop's, so the answer there is HostMenu.
func StatusMenuChromeFor(want platform.StatusMenuChrome) (platform.StatusMenuChrome, string) {
	if want == platform.ToolkitMenu {
		if !platform.ScreenPlacementAvailable() {
			return platform.HostMenu, "this compositor has no zwlr_layer_shell_v1, so a toolkit menu window cannot be put where the tray asked"
		}
		return platform.ToolkitMenu, "the app asked for it"
	}
	// HostMenu on Linux is dbusmenu. Windows still uses OnMenu (no HMENU).
	if platform.HostMenuNative() {
		return platform.HostMenu, "the desktop draws tray menus (dbusmenu)"
	}
	// Nothing native: the toolkit menu is all there is, placeable or not.
	return platform.ToolkitMenu, "this desktop draws no tray menu of its own"
}

type statusMenuHolder struct {
	items []platform.StatusMenuItem
}

func (h *statusMenuHolder) Items() []platform.StatusMenuItem {
	if h == nil {
		return nil
	}
	return platformCopyMenu(h.items)
}

func (h *statusMenuHolder) Set(items []platform.StatusMenuItem) {
	if h == nil {
		return
	}
	h.items = platformCopyMenu(items)
}

type trackingStatusItem struct {
	platform.StatusItem
	holder *statusMenuHolder
}

func (t *trackingStatusItem) SetMenu(items []platform.StatusMenuItem) error {
	if t.holder != nil {
		t.holder.Set(items)
	}
	return t.StatusItem.SetMenu(items)
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

// StatusMenuFromItems copies widget menu rows into a tray menu,
// cascades included: a MenuItem.Submenu becomes StatusMenuItem.Submenu.
// A parent's OnClick is dropped, because a tray parent is not a command
// (see platform.StatusMenuItem.Submenu) and a PopupMenu never called it
// either.
func StatusMenuFromItems(items []*widgets.MenuItem) []platform.StatusMenuItem {
	out := make([]platform.StatusMenuItem, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		row := platform.StatusMenuItem{
			Text:      it.Text,
			Disabled:  it.Disabled,
			Separator: it.Separator,
			Checked:   it.Checked,
			Icon:      it.Icon,
			OnClick:   it.OnClick,
		}
		if it.HasSubmenu() && !it.Separator {
			row.Submenu = StatusMenuFromItems(it.Submenu)
			row.OnClick = nil
		}
		out = append(out, row)
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

// StatusMenuToItems maps tray rows back to toolkit MenuItems (popup
// chrome), cascades included. This is the ToolkitMenu path on Linux and
// the only path on Windows, which has no dbusmenu.
func StatusMenuToItems(items []platform.StatusMenuItem) []*widgets.MenuItem {
	out := make([]*widgets.MenuItem, 0, len(items))
	for _, it := range items {
		if it.Separator {
			// A separator is a rule, never a parent, even if the app put
			// children on it (platform.StatusMenuItem.Submenu).
			out = append(out, widgets.Sep())
			continue
		}
		if len(it.Submenu) > 0 {
			// widgets.Submenu is the cascade parent: PopupMenu opens the
			// child on hover or click and never calls OnClick, which is
			// the rule the tray row promises.
			row := widgets.Submenu(it.Text, StatusMenuToItems(it.Submenu)...)
			row.Disabled = it.Disabled
			row.Icon = it.Icon
			out = append(out, row)
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
// status-menu window (ToolkitMenu chrome). The window is reused: Hide/Show
// + rebind, never create/destroy per click. x,y are SNI root/screen
// coordinates; X11 places the popup there (typically above a bottom
// panel), and Wayland does through a layer surface
// ([platform.PlaceAtScreen]) where the compositor offers one. Where it
// does not, [Application.NewStatusItem] has already chosen HostMenu, so
// the only way to be here is an app that called this directly — the
// compositor then places the window and the menu may be anywhere.
func (a *Application) ShowStatusMenu(x, y int32, items []platform.StatusMenuItem) {
	if a == nil {
		return
	}
	rows := StatusMenuToItems(items)
	if len(rows) == 0 {
		return
	}
	mw, mh := measureStatusMenu(a.Look(), a.Scale(), rows)
	px, py := statusMenuScreenPos(x, y, mw, mh)
	// The menu is measured in the look's device pixels and the tray names
	// a point in root ones; a window's size and position are stated in
	// logical pixels.
	lw := platform.LogicalPixels(mw, a.Scale())
	lh := platform.LogicalPixels(mh, a.Scale())
	px, py = platform.LogicalPosition(px, a.Scale()), platform.LogicalPosition(py, a.Scale())
	w := a.statusMenu
	if w == nil || w.Closed() {
		opts := platform.WindowOptions{
			Title:     " ",
			Width:     lw,
			Height:    lh,
			MinWidth:  1,
			MinHeight: 1,
			Popup:     true,
			X:         px,
			Y:         py,
			// The menu belongs at the point the tray named, not near it:
			// on Wayland this is what makes the window a layer surface
			// rather than an unplaceable toplevel.
			Place: platform.PlaceAtScreen,
		}
		var err error
		w, err = a.NewWindow(opts)
		if err != nil || w == nil {
			a.showStatusMenuOnPreferred(x, y, rows)
			return
		}
		w.statusMenu = true
		a.statusMenu = w
	} else if err := w.Surface().Resize(lw, lh); err != nil {
		trayStatusLog("ShowStatusMenu resize: %v", err)
	}
	if !a.bindStatusMenu(w, rows, mw, mh, px, py) {
		a.showStatusMenuOnPreferred(x, y, rows)
	}
}

func (a *Application) bindStatusMenu(w *Window, rows []*widgets.MenuItem, mw, mh, px, py int) bool {
	if w == nil || w.Closed() {
		return false
	}
	if pop, ok := w.Popup().(*widgets.PopupMenu); ok && pop != nil {
		pop.OnDismiss = nil
	}
	w.DismissPopup()
	w.statusMenu = true
	w.statusMenuArmed = false
	w.statusMenuArmAt = time.Now().Add(statusMenuArmDelay)
	from := widgets.NewLabel("")
	w.SetContent(from)
	pop := widgets.NewPopupMenu(rows...)
	pop.OnDismiss = func() {
		a.hideStatusMenu()
	}
	widget.PreparePopup(from, pop)
	pop.Arrange(paintengine2d.XYWH(0, 0, float32(mw), float32(mh)))
	if !widget.ShowPopup(from, pop) {
		trayStatusLog("ShowStatusMenu popup layer failed; falling back to main window")
		a.hideStatusMenu()
		return false
	}
	pop.RequestFocus()
	w.Show()
	w.Raise()
	placed := true
	if px != 0 || py != 0 {
		placed = platform.PlaceSurfaceAtScreen(w.Surface(), px, py)
	}
	trayStatusLog("ShowStatusMenu place=%d,%d size=%dx%d popupWin visible=%v placed=%v reused",
		px, py, mw, mh, w.Visible(), placed)
	return true
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

func (a *Application) hideStatusMenu() {
	if a == nil || a.hidingStatusMenu {
		return
	}
	w := a.statusMenu
	if w == nil || w.Closed() {
		return
	}
	a.hidingStatusMenu = true
	defer func() { a.hidingStatusMenu = false }()
	w.statusMenuArmed = false
	w.statusMenuArmAt = time.Time{}
	if pop, ok := w.Popup().(*widgets.PopupMenu); ok && pop != nil {
		pop.OnDismiss = nil
	}
	if w.Popup() != nil {
		w.DismissPopup()
	}
	if w.Visible() {
		w.Hide()
	}
}

// measureStatusMenu sizes the status-menu window. The window holds the
// whole cascade: a child menu is a popup inside this surface, and
// widget.PlacePopupBeside clamps it to the surface box, so a window
// measured for the parent alone would squeeze every submenu on top of
// it. The box is therefore the parent's width plus the widest chain of
// children, and the tallest menu in that chain.
func measureStatusMenu(look style.LookAndFeel, scale float32, items []*widgets.MenuItem) (int, int) {
	host := &statusMeasureHost{look: look, scale: scale}
	fw, fh := statusMenuBox(host, items)
	w, h := int(fw+0.5), int(fh+0.5)
	if w < 80 {
		w = 80
	}
	if h < 24 {
		h = 24
	}
	return w, h
}

// statusMenuBox is the intrinsic size of a menu and everything it can
// cascade into: submenus open to the right, so widths add and heights
// only need the tallest level.
func statusMenuBox(host *statusMeasureHost, items []*widgets.MenuItem) (float32, float32) {
	pop := widgets.NewPopupMenu(items...)
	pop.SetHost(host)
	sz := pop.Measure(layout.Unbounded())
	w, h := sz.X, sz.Y
	var childW, childH float32
	for _, it := range items {
		if !it.HasSubmenu() {
			continue
		}
		cw, ch := statusMenuBox(host, it.Submenu)
		if cw > childW {
			childW = cw
		}
		if ch > childH {
			childH = ch
		}
	}
	if childH > h {
		h = childH
	}
	return w + childW, h
}

type statusMeasureHost struct {
	look  style.LookAndFeel
	scale float32
}

func (h *statusMeasureHost) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *statusMeasureHost) RequestFocus(widget.Component)                   {}
func (h *statusMeasureHost) Focus() widget.Component                         { return nil }
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
