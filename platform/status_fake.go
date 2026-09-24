package platform

import "sync"

// FakeStatusItem records tray calls for tests (UITK_TRAY=fake).
type FakeStatusItem struct {
	mu      sync.Mutex
	opts    StatusItemOptions
	icon    StatusIcon
	tooltip string
	title   string
	menu    []StatusMenuItem
	Notes   []Notification
	Clicks  int
	closed  bool
}

func newFakeStatusItem(opts StatusItemOptions) *FakeStatusItem {
	return &FakeStatusItem{
		opts:    opts,
		icon:    opts.Icon,
		tooltip: opts.Tooltip,
		title:   opts.Title,
		menu:    copyMenu(opts.Menu),
	}
}

func (f *FakeStatusItem) SetIcon(icon StatusIcon) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.icon = icon
	return nil
}

func (f *FakeStatusItem) SetTooltip(s string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tooltip = s
	return nil
}

func (f *FakeStatusItem) SetTitle(s string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.title = s
	return nil
}

func (f *FakeStatusItem) SetMenu(items []StatusMenuItem) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.menu = copyMenu(items)
	return nil
}

func (f *FakeStatusItem) Notify(n Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Notes = append(f.Notes, n)
	return nil
}

func (f *FakeStatusItem) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *FakeStatusItem) Backend() string { return "fake" }

func (f *FakeStatusItem) Alive() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return !f.closed
}

// Icon / Tooltip / Title / Menu are test snapshots.
func (f *FakeStatusItem) Icon() StatusIcon {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.icon
}

func (f *FakeStatusItem) Tooltip() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tooltip
}

func (f *FakeStatusItem) Title() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.title
}

func (f *FakeStatusItem) Menu() []StatusMenuItem {
	f.mu.Lock()
	defer f.mu.Unlock()
	return copyMenu(f.menu)
}

// MenuChrome is the chrome selected at construction (tests).
func (f *FakeStatusItem) MenuChrome() StatusMenuChrome {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.opts.MenuChrome
}

func (f *FakeStatusItem) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// Click fires the primary callback (as a compositor would).
func (f *FakeStatusItem) Click() {
	f.mu.Lock()
	f.Clicks++
	menuOnly := f.opts.ItemIsMenu
	fn := f.opts.OnClick
	menuFn := f.opts.OnMenu
	dispatch := f.opts.Dispatch
	f.mu.Unlock()
	if menuOnly {
		if menuFn != nil {
			invokeStatus(dispatch, func() { menuFn(0, 0) })
		}
		return
	}
	invokeStatus(dispatch, fn)
}

// ClickNotify activates the last notification.
func (f *FakeStatusItem) ClickNotify() {
	f.mu.Lock()
	var fn func()
	if n := len(f.Notes); n > 0 {
		fn = f.Notes[n-1].OnClick
	}
	if fn == nil {
		fn = f.opts.OnNotifyClick
	}
	if fn == nil {
		fn = f.opts.OnClick
	}
	f.mu.Unlock()
	invokeStatus(f.opts.Dispatch, fn)
}

// ContextClick is a tray right-click (StatusNotifierItem.ContextMenu).
func (f *FakeStatusItem) ContextClick(x, y int32) {
	f.mu.Lock()
	fn := f.opts.OnMenu
	f.mu.Unlock()
	if fn == nil {
		return
	}
	invokeStatus(f.opts.Dispatch, func() { fn(x, y) })
}

// ClickMenu activates top-level menu row i.
func (f *FakeStatusItem) ClickMenu(i int) { f.ClickMenuPath(i) }

// ClickMenuPath activates a row addressed by index at each level:
// ClickMenuPath(2, 0) is the first row of the third row's submenu. A
// cascade parent is not a command, so a path that stops on one fires
// nothing — the same rule the dbusmenu exporter enforces.
func (f *FakeStatusItem) ClickMenuPath(path ...int) {
	f.mu.Lock()
	rows := f.menu
	var fn func()
	for depth, i := range path {
		if i < 0 || i >= len(rows) {
			break
		}
		it := rows[i]
		if depth == len(path)-1 {
			if menuItemClickable(it) {
				fn = it.OnClick
			}
			break
		}
		rows = menuItemChildren(it)
	}
	f.mu.Unlock()
	invokeStatus(f.opts.Dispatch, fn)
}
