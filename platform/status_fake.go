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

func (f *FakeStatusItem) Closed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

// Click fires the primary callback (as a compositor would).
func (f *FakeStatusItem) Click() {
	f.mu.Lock()
	f.Clicks++
	fn := f.opts.OnClick
	f.mu.Unlock()
	invokeStatus(f.opts.Dispatch, fn)
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

// ClickMenu activates menu row i.
func (f *FakeStatusItem) ClickMenu(i int) {
	f.mu.Lock()
	var fn func()
	if i >= 0 && i < len(f.menu) {
		fn = f.menu[i].OnClick
	}
	f.mu.Unlock()
	invokeStatus(f.opts.Dispatch, fn)
}
