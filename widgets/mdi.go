package widgets

import (
	"math"
	"strconv"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// MDIArea holds windows inside a window — Qt's QMdiArea, the multiple-
// document interface of Windows applications from Program Manager to
// Office 97. Each MDIWindow wears the look's own window frame (the
// caption, border, buttons and shadow the look gives a real window,
// style.DrawDecorationOf), moves by its caption, resizes by its edges and
// corners, and minimises to a caption strip along the area's foot,
// maximises to fill the area, and restores. Windows stack in activation
// order; Cascade and Tile arrange them.
//
// The keyboard: Ctrl+Tab and Ctrl+F6 go to the next window (Shift: the
// previous), Ctrl+F4 or Ctrl+W asks the active one to close, and Alt+-
// opens its window menu, whose Move and Size let the arrow keys move and
// size it (Return keeps it, Escape puts it back). WindowMenu is the
// Window menu an application puts in its menu bar: arranging, closing,
// and the list of windows.
//
// SetViewMode(MDITabbed) shows the windows as document tabs instead, as
// Qt's tabbed view does: a tab strip across the top and the active window
// filling the rest.
//
// Assistive technology reads the area as a desktop pane and each window
// as an internal frame named by its title, with its caption buttons.
type MDIArea struct {
	widget.Base
	// OnActivate is told when a window becomes the active one.
	OnActivate func(w *MDIWindow)
	// OnWindowsChanged is told when a window opens, closes or is renamed.
	OnWindowsChanged func()
	// ButtonLayout places the windows' caption buttons; the zero layout
	// takes the look's own, else minimise, maximise and close at the right.
	ButtonLayout platform.ButtonLayout

	wins   []*MDIWindow // creation order: what Ctrl+Tab walks
	active *MDIWindow
	mode   MDIViewMode
	tabs   *BrowserTabs
	menus  []*Menu
	placed int     // windows placed so far, for the next one's cascade
	scale  float32 // the look scale window geometry is stored at

	// kb is a keyboard move or size in progress: its window, its kind and
	// the geometry to go back to on Escape.
	kb      *MDIWindow
	kbSize  bool
	kbStart paintengine2d.Rect
	kbFocus widget.Component
	syncing bool
}

// MDIViewMode is how an area shows its windows.
type MDIViewMode uint8

const (
	// MDISubWindows shows each window in a frame of its own.
	MDISubWindows MDIViewMode = iota
	// MDITabbed shows one window at a time, under a strip of document
	// tabs.
	MDITabbed
)

// MDIFeatures are what a window's frame lets the user do.
type MDIFeatures uint8

const (
	MDIClosable MDIFeatures = 1 << iota
	MDIMinimizable
	MDIMaximizable
	MDIResizable
	MDIMovable
	MDIDefaultFeatures = MDIClosable | MDIMinimizable | MDIMaximizable | MDIResizable | MDIMovable
)

// mdiState is a window's size state.
type mdiState uint8

const (
	mdiNormal mdiState = iota
	mdiMinimized
	mdiMaximized
)

// MDIWindow is a window inside an MDIArea.
type MDIWindow struct {
	widget.Base
	// Features are what the frame lets the user do (MDIDefaultFeatures).
	Features MDIFeatures
	// MinSize is the least size, in logical pixels, a resize leaves it
	// (zero: 160 by the caption and 60).
	MinSize paintengine2d.Point
	// OnCloseRequest is asked before the window closes (its close button,
	// Ctrl+F4, Close); returning false keeps it open — to ask about unsaved
	// changes first, and Dismiss it afterwards.
	OnCloseRequest func() bool
	// OnClose is told the window has closed.
	OnClose      func()
	area         *MDIArea
	initialFocus widget.Component
	title        string
	client       *mdiClient
	state        mdiState
	restore      mdiState           // what a minimised window restores to
	normal       paintengine2d.Rect // device pixels in the area, the visible window
	// pointer
	hot, down  platform.CaptionButton
	drag       int // 0 none, 1 move, 2 resize
	edges      style.Edges
	grab       paintengine2d.Point
	start      paintengine2d.Rect
	lastClick  time.Time
	lastFocus  widget.Component
	maxRestore paintengine2d.Rect
}

// mdiClient holds a window's content, cut to the frame's rounded corners.
type mdiClient struct {
	widget.Base
	win     *MDIWindow
	content widget.Component
}

// NewMDIArea is an empty area.
func NewMDIArea() *MDIArea {
	a := &MDIArea{}
	a.Init(a)
	return a
}

// SetHost: an area that arrives in a window with nothing focused yet
// puts the keyboard in its active window, so the window's own first-focus
// rule (the first control in Tab order, the bottom window's) does not
// raise another one over it.
func (a *MDIArea) SetHost(h widget.Host) {
	a.Base.SetHost(h)
	if h != nil && h.Focus() == nil && a.active != nil {
		a.active.takeFocus()
	}
}

// AddWindow opens a window holding content, cascaded from the last one
// and made the active window. Its size is content's natural size, within
// the area.
func (a *MDIArea) AddWindow(title string, content widget.Component) *MDIWindow {
	w := &MDIWindow{area: a, title: title, Features: MDIDefaultFeatures, hot: platform.CaptionNone, down: platform.CaptionNone}
	w.Init(w)
	w.client = &mdiClient{win: w, content: content}
	w.client.Init(w.client)
	if content != nil {
		w.client.Add(content)
	}
	w.Add(w.client)
	a.wins = append(a.wins, w)
	a.Add(w)
	w.normal = a.nextPlace(w)
	a.placed++
	a.syncTabs()
	a.changed()
	a.Activate(w)
	a.RequestLayout()
	return w
}

// nextPlace is where a new window opens: a caption's height down and to
// the right of the last, wrapping back to the corner when it would leave
// the area.
func (a *MDIArea) nextPlace(w *MDIWindow) paintengine2d.Rect {
	lk := a.Look()
	step := w.captionH() + w.spec().Border.Top
	ww, wh := style.Dip(lk, 360), style.Dip(lk, 240)
	if c := w.client.content; c != nil {
		sz := c.Measure(layout.Loose(style.Dip(lk, 640), style.Dip(lk, 480)))
		ww = max(ww*0.6, sz.X+w.frameW())
		wh = max(wh*0.6, sz.Y+w.frameH())
	}
	b := a.LocalBounds()
	if b.Dx() > 0 && b.Dy() > 0 {
		ww, wh = min(ww, b.Dx()*0.8), min(wh, b.Dy()*0.8)
	}
	n := a.placed
	x, y := float32(n)*step, float32(n)*step
	if b.Dx() > 0 && (x+ww > b.Dx() || y+wh > b.Dy()) {
		a.placed = 0
		x, y = 0, 0
	}
	return pxRect(paintengine2d.XYWH(x, y, ww, wh))
}

func pxRect(r paintengine2d.Rect) paintengine2d.Rect {
	rd := func(v float32) float32 { return float32(math.Round(float64(v))) }
	return paintengine2d.Rect{Min: paintengine2d.Pt(rd(r.Min.X), rd(r.Min.Y)), Max: paintengine2d.Pt(rd(r.Max.X), rd(r.Max.Y))}
}

// Windows are the area's windows in the order they were opened.
func (a *MDIArea) Windows() []*MDIWindow { return append([]*MDIWindow(nil), a.wins...) }

// StackingOrder is the windows from the bottom of the stack to the top.
func (a *MDIArea) StackingOrder() []*MDIWindow {
	var out []*MDIWindow
	for _, c := range a.Children() {
		if w, ok := c.(*MDIWindow); ok {
			out = append(out, w)
		}
	}
	return out
}

// ActiveWindow is the active window, or nil.
func (a *MDIArea) ActiveWindow() *MDIWindow { return a.active }

// ViewMode is how the area shows its windows.
func (a *MDIArea) ViewMode() MDIViewMode { return a.mode }

// SetViewMode shows the windows in frames of their own or as tabs.
func (a *MDIArea) SetViewMode(m MDIViewMode) {
	if a.mode == m {
		return
	}
	a.endKeyboard(false)
	a.mode = m
	a.syncTabs()
	a.RequestLayout()
	a.Invalidate()
}

// Activate makes w the active window: raised to the top, its caption
// lit, and the keyboard in it — on the widget that last had it there, or
// its first. A maximised window hands its state on: activating another
// maximises that one instead, as Windows' MDI does.
func (a *MDIArea) Activate(w *MDIWindow) {
	a.activate(w, true)
}

func (a *MDIArea) activate(w *MDIWindow, focus bool) {
	if w == nil || w.area != a {
		return
	}
	prev := a.active
	if prev != w && prev != nil && prev.state == mdiMaximized && w.state == mdiNormal && a.mode == MDISubWindows {
		prev.state = mdiNormal
		w.maxRestore = w.normal
		w.state = mdiMaximized
	}
	a.active = w
	a.raise(w)
	if focus {
		w.takeFocus()
	}
	if a.tabs != nil && a.mode == MDITabbed {
		if i := a.indexOf(w); i >= 0 && a.tabs.Selected() != i {
			a.syncing = true
			a.tabs.Select(i)
			a.syncing = false
		}
	}
	a.refreshMenus()
	a.RequestLayout()
	a.Invalidate()
	if prev != w && a.OnActivate != nil {
		a.OnActivate(w)
	}
}

// raise puts w at the top of the stack (the end of the children, which is
// the order they paint and take the pointer in).
func (a *MDIArea) raise(w widget.Component) {
	kids := a.Children()
	if len(kids) > 0 && kids[len(kids)-1] == w {
		return
	}
	a.Remove(w)
	a.Add(w)
	if a.tabs != nil && w != widget.Component(a.tabs) {
		// The tab strip stays above every window.
		a.Remove(a.tabs)
		a.Add(a.tabs)
	}
}

func (a *MDIArea) indexOf(w *MDIWindow) int {
	for i, x := range a.wins {
		if x == w {
			return i
		}
	}
	return -1
}

// ActivateNext and ActivatePrevious walk the windows in the order they
// were opened (Ctrl+Tab and Ctrl+Shift+Tab).
func (a *MDIArea) ActivateNext()     { a.step(1) }
func (a *MDIArea) ActivatePrevious() { a.step(-1) }

func (a *MDIArea) step(d int) {
	n := len(a.wins)
	if n == 0 {
		return
	}
	i := a.indexOf(a.active)
	if i < 0 {
		i = 0
		if d < 0 {
			i = n - 1
		}
	} else {
		i = (i + d + n) % n
	}
	a.Activate(a.wins[i])
}

// CloseActive asks the active window to close.
func (a *MDIArea) CloseActive() bool {
	if a.active == nil {
		return false
	}
	return a.active.Close()
}

// CloseAll asks every window to close, top first, and stops at the first
// that will not; it reports whether they all did.
func (a *MDIArea) CloseAll() bool {
	st := a.StackingOrder()
	for i := len(st) - 1; i >= 0; i-- {
		if !st[i].Close() {
			return false
		}
	}
	return true
}

// remove takes w out of the area and activates the window under it.
func (a *MDIArea) remove(w *MDIWindow) {
	i := a.indexOf(w)
	if i < 0 {
		return
	}
	if a.kb == w {
		a.endKeyboard(false)
	}
	a.wins = append(a.wins[:i], a.wins[i+1:]...)
	hadFocus := widget.FocusWithin(w)
	wasActive := a.active == w
	a.Remove(w)
	if wasActive {
		a.active = nil
		st := a.StackingOrder()
		for j := len(st) - 1; j >= 0; j-- {
			if st[j].state != mdiMinimized || j == 0 {
				a.activate(st[j], hadFocus)
				break
			}
		}
	}
	if hadFocus && a.active == nil {
		if h := a.Host(); h != nil {
			h.RequestFocus(nil)
		}
	}
	a.syncTabs()
	a.changed()
	a.RequestLayout()
	a.Invalidate()
}

// changed tells the application and the Window menus the list changed.
func (a *MDIArea) changed() {
	a.refreshMenus()
	if a.OnWindowsChanged != nil {
		a.OnWindowsChanged()
	}
}

// ---- arranging ---------------------------------------------------------

// work is the room windows are arranged in: the area less the row of
// minimised windows along its foot.
func (a *MDIArea) work() paintengine2d.Rect {
	b := a.LocalBounds()
	if n := a.minimizedCount(); n > 0 {
		rows := a.iconRows(n)
		b.Max.Y -= float32(rows) * (a.iconH() + a.iconGap())
	}
	return b
}

func (a *MDIArea) minimizedCount() int {
	n := 0
	for _, w := range a.wins {
		if w.state == mdiMinimized {
			n++
		}
	}
	return n
}

// Cascade restores every window and stacks them down and to the right
// from the corner, each a caption's height from the last, in the order
// they were opened with the active one last, on top.
func (a *MDIArea) Cascade() {
	a.endKeyboard(true)
	wins := a.shown()
	if len(wins) == 0 {
		return
	}
	// The active window goes last, at the front of the cascade, where
	// every caption above it still shows.
	for i, w := range wins {
		if w == a.active && i < len(wins)-1 {
			wins = append(append(wins[:i:i], wins[i+1:]...), w)
			break
		}
	}
	wr := a.work()
	w0 := wins[0]
	step := w0.captionH() + w0.spec().Border.Top
	ww, wh := max(wr.Dx()*0.66, w0.minW()), max(wr.Dy()*0.66, w0.minH())
	x, y := float32(0), float32(0)
	for _, w := range wins {
		w.state = mdiNormal
		if x+ww > wr.Dx() || y+wh > wr.Dy() {
			x, y = 0, 0
		}
		w.normal = pxRect(paintengine2d.XYWH(wr.Min.X+x, wr.Min.Y+y, ww, wh))
		x += step
		y += step
	}
	a.placed = len(wins)
	a.restack(wins)
}

// Tile restores every window and shares the area among them in a grid,
// as square as the count allows; a short last row takes the full width.
func (a *MDIArea) Tile() {
	n := len(a.shown())
	if n == 0 {
		return
	}
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	a.tile(cols, int(math.Ceil(float64(n)/float64(cols))))
}

// TileRows stacks the windows one above another, each the area's width
// (Windows' "Tile Horizontally").
func (a *MDIArea) TileRows() { a.tile(1, len(a.shown())) }

// TileColumns puts the windows side by side, each the area's height
// (Windows' "Tile Vertically").
func (a *MDIArea) TileColumns() { a.tile(len(a.shown()), 1) }

func (a *MDIArea) tile(cols, rows int) {
	a.endKeyboard(true)
	wins := a.shown()
	if len(wins) == 0 || cols < 1 || rows < 1 {
		return
	}
	wr := a.work()
	i := 0
	for r := 0; r < rows; r++ {
		inRow := cols
		if left := len(wins) - i; left < cols {
			inRow = left
		}
		y0 := wr.Min.Y + wr.Dy()*float32(r)/float32(rows)
		y1 := wr.Min.Y + wr.Dy()*float32(r+1)/float32(rows)
		for c := 0; c < inRow; c++ {
			x0 := wr.Min.X + wr.Dx()*float32(c)/float32(inRow)
			x1 := wr.Min.X + wr.Dx()*float32(c+1)/float32(inRow)
			w := wins[i]
			w.state = mdiNormal
			w.normal = pxRect(paintengine2d.Rect{Min: paintengine2d.Pt(x0, y0), Max: paintengine2d.Pt(x1, y1)})
			i++
		}
	}
	a.restack(wins)
}

// shown are the windows that are not minimised, in the order they were
// opened.
func (a *MDIArea) shown() []*MDIWindow {
	var out []*MDIWindow
	for _, w := range a.wins {
		if w.state != mdiMinimized {
			out = append(out, w)
		}
	}
	return out
}

// restack stacks wins in their order with the active one on top.
func (a *MDIArea) restack(wins []*MDIWindow) {
	for _, w := range wins {
		a.raise(w)
	}
	if a.active != nil {
		a.raise(a.active)
	}
	a.RequestLayout()
	a.Invalidate()
}

func (a *MDIArea) iconW() float32   { return float32(math.Round(float64(style.Dip(a.Look(), 170)))) }
func (a *MDIArea) iconGap() float32 { return float32(math.Round(float64(style.Dip(a.Look(), 2)))) }
func (a *MDIArea) iconH() float32 {
	if len(a.wins) == 0 {
		return style.Dip(a.Look(), 24)
	}
	w := a.wins[0]
	s := w.specFor(mdiMinimized, false)
	return w.captionHFor(s) + s.Border.Top + s.Border.Bottom
}

func (a *MDIArea) iconRows(n int) int {
	per := max(int((a.LocalBounds().Dx()+a.iconGap())/(a.iconW()+a.iconGap())), 1)
	return (n + per - 1) / per
}

// iconRect is where the k-th minimised window sits: along the area's
// foot, left to right, then a row up.
func (a *MDIArea) iconRect(k int) paintengine2d.Rect {
	b := a.LocalBounds()
	iw, ih, gap := a.iconW(), a.iconH(), a.iconGap()
	per := max(int((b.Dx()+gap)/(iw+gap)), 1)
	row, col := k/per, k%per
	return paintengine2d.XYWH(float32(col)*(iw+gap), b.Max.Y-float32(row+1)*(ih+gap)+gap, iw, ih)
}

func (a *MDIArea) Measure(c layout.Constraints) paintengine2d.Point {
	lk := a.Look()
	return c.Constrain(paintengine2d.Pt(style.Dip(lk, 480), style.Dip(lk, 320)))
}

func (a *MDIArea) Arrange(r paintengine2d.Rect) {
	a.SetBounds(r)
	a.rescale()
	b := a.LocalBounds()
	if a.mode == MDITabbed {
		th := a.tabs.Measure(layout.Loose(b.Dx(), b.Dy())).Y
		a.tabs.Arrange(paintengine2d.XYWH(0, 0, b.Dx(), th))
		body := paintengine2d.XYWH(0, th, b.Dx(), max(b.Dy()-th, 0))
		for _, w := range a.wins {
			w.SetVisible(w == a.active)
			if w == a.active {
				w.Arrange(body)
			}
		}
		return
	}
	k := 0
	for _, w := range a.wins {
		w.SetVisible(true)
		switch w.state {
		case mdiMinimized:
			w.Arrange(a.iconRect(k))
			k++
		case mdiMaximized:
			w.Arrange(w.outset(a.work()))
		default:
			w.Arrange(w.outset(w.normal))
		}
	}
}

// rescale keeps the windows' places when the display scale changes: they
// are stored in device pixels at the scale they were placed at.
func (a *MDIArea) rescale() {
	s := style.LookScale(a.Look())
	if s <= 0 {
		s = 1
	}
	if a.scale == 0 {
		a.scale = s
		return
	}
	if a.scale == s {
		return
	}
	k := s / a.scale
	for _, w := range a.wins {
		w.normal = pxRect(paintengine2d.Rect{Min: w.normal.Min.Mul(k), Max: w.normal.Max.Mul(k)})
		w.maxRestore = pxRect(paintengine2d.Rect{Min: w.maxRestore.Min.Mul(k), Max: w.maxRestore.Max.Mul(k)})
	}
	a.scale = s
}

// Paint fills the area's backdrop: the look's workspace, a shade darker
// than a window's face (Windows' AppWorkspace grey).
func (a *MDIArea) Paint(ctx *paintengine2d.Context) {
	p := a.Look().Palette()
	bg := style.Mix(p.Background, p.Text, 0.12)
	if a.mode == MDITabbed {
		bg = p.Background
	}
	ctx.DrawRect(a.LocalBounds(), paintengine2d.Fill(bg))
}

// ---- tabbed view -------------------------------------------------------

// syncTabs builds the tab strip of the tabbed view from the windows.
func (a *MDIArea) syncTabs() {
	if a.mode != MDITabbed {
		if a.tabs != nil {
			a.tabs.SetVisible(false)
		}
		return
	}
	if a.tabs == nil {
		a.tabs = NewBrowserTabs()
		a.tabs.OnSelect = func(i int) {
			if !a.syncing && i >= 0 && i < len(a.wins) {
				a.Activate(a.wins[i])
			}
		}
		a.tabs.OnClose = func(i int) {
			if i >= 0 && i < len(a.wins) {
				a.wins[i].Close()
			}
		}
		a.tabs.OnReorder = func(from, to int) {
			if from < 0 || from >= len(a.wins) || to < 0 || to >= len(a.wins) {
				return
			}
			w := a.wins[from]
			a.wins = append(a.wins[:from], a.wins[from+1:]...)
			a.wins = append(a.wins[:to], append([]*MDIWindow{w}, a.wins[to:]...)...)
			a.changed()
		}
		a.tabs.SetAccessibleName("Documents")
		a.Add(a.tabs)
	}
	a.syncing = true
	for a.tabs.Len() > 0 {
		a.tabs.RemoveTab(a.tabs.Len() - 1)
	}
	for _, w := range a.wins {
		a.tabs.AddTab(BrowserTab{Title: w.title})
	}
	if i := a.indexOf(a.active); i >= 0 {
		a.tabs.Select(i)
	}
	a.syncing = false
	a.tabs.SetVisible(true)
	a.raise(a.tabs)
}

// ---- keyboard ----------------------------------------------------------

// KeyPress takes the area's keys as they bubble up from the window that
// has the keyboard.
func (a *MDIArea) KeyPress(e widget.KeyEvent) bool {
	if a.kb != nil {
		return a.keyboardMove(e)
	}
	ctrl, shift := e.Mods.Ctrl(), e.Mods.Shift()
	r := e.Rune
	if r == 0 {
		r, _ = platform.KeyRune(e.Key)
	}
	switch {
	case ctrl && (e.Key == platform.KeyTab || e.Key == platform.KeyF6):
		if shift {
			a.ActivatePrevious()
		} else {
			a.ActivateNext()
		}
		return true
	case ctrl && (e.Key == platform.KeyF4 || r == 'w') && !shift:
		return a.CloseActive() || a.active != nil
	case e.Mods.Alt() && !ctrl && r == '-':
		if w := a.active; w != nil && a.mode == MDISubWindows {
			w.showMenu(w.menuAnchor())
			return true
		}
	}
	return false
}

// startKeyboard lets the arrow keys move (or size) w until Return or
// Escape, as a window menu's Move and Size do.
func (a *MDIArea) startKeyboard(w *MDIWindow, size bool) {
	if w == nil || w.state != mdiNormal {
		return
	}
	a.kb, a.kbSize, a.kbStart = w, size, w.normal
	a.kbFocus = widget.FocusOwner(a)
	a.RequestFocus()
	a.Invalidate()
}

// endKeyboard ends a keyboard move or size, keeping the new geometry or
// putting the old one back.
func (a *MDIArea) endKeyboard(keep bool) {
	w := a.kb
	if w == nil {
		return
	}
	if !keep {
		w.normal = a.kbStart
	}
	a.kb = nil
	if a.kbFocus != nil && widget.LiveUnder(a.kbFocus, a) {
		if h := a.Host(); h != nil {
			h.RequestFocus(a.kbFocus)
		}
	} else {
		w.takeFocus()
	}
	a.kbFocus = nil
	a.RequestLayout()
	a.Invalidate()
}

func (a *MDIArea) keyboardMove(e widget.KeyEvent) bool {
	w := a.kb
	step := float32(math.Round(float64(style.Dip(a.Look(), 10))))
	if e.Mods.Ctrl() {
		step = 1
	}
	var dx, dy float32
	switch e.Key {
	case platform.KeyLeft:
		dx = -step
	case platform.KeyRight:
		dx = step
	case platform.KeyUp:
		dy = -step
	case platform.KeyDown:
		dy = step
	case platform.KeyReturn, platform.KeySpace:
		a.endKeyboard(true)
		return true
	case platform.KeyEscape:
		a.endKeyboard(false)
		return true
	default:
		return true // the window keeps the keyboard until it is done
	}
	r := w.normal
	if a.kbSize {
		r.Max.X = max(r.Max.X+dx, r.Min.X+w.minW())
		r.Max.Y = max(r.Max.Y+dy, r.Min.Y+w.minH())
	} else {
		r = r.Translate(paintengine2d.Pt(dx, dy))
	}
	w.normal = a.clampMove(w, r)
	a.RequestLayout()
	a.Invalidate()
	return true
}

// clampMove keeps enough of a window's caption inside the area to grab it
// again: its top edge in, and a caption's worth of its width.
func (a *MDIArea) clampMove(w *MDIWindow, r paintengine2d.Rect) paintengine2d.Rect {
	b := a.work()
	keep := min(style.Dip(a.Look(), 48), r.Dx())
	dx, dy := float32(0), float32(0)
	if r.Min.Y < b.Min.Y {
		dy = b.Min.Y - r.Min.Y
	}
	if top := b.Max.Y - w.captionH(); r.Min.Y > top {
		dy = top - r.Min.Y
	}
	if r.Max.X < b.Min.X+keep {
		dx = b.Min.X + keep - r.Max.X
	}
	if r.Min.X > b.Max.X-keep {
		dx = b.Max.X - keep - r.Min.X
	}
	return r.Translate(paintengine2d.Pt(dx, dy))
}

// FocusMoved follows the keyboard into a window: the window it lands in
// becomes the active one, and remembers where it was for when it is
// active again.
func (a *MDIArea) FocusMoved(now widget.Component) {
	if now == nil || now == widget.Component(a) {
		return
	}
	for _, w := range a.wins {
		if widget.Contains(w, now) {
			w.lastFocus = now
			if a.active != w {
				a.activate(w, false)
			}
			return
		}
	}
}

// ---- the Window menu ---------------------------------------------------

// WindowMenu is a Window menu for the application's menu bar: Cascade,
// Tile, closing, next and previous, and the windows themselves, the
// active one checked. It follows the windows as they open and close.
func (a *MDIArea) WindowMenu() *Menu {
	m := NewMenu("&Window")
	a.menus = append(a.menus, m)
	a.fillMenu(m)
	return m
}

func (a *MDIArea) refreshMenus() {
	for _, m := range a.menus {
		a.fillMenu(m)
	}
}

func (a *MDIArea) fillMenu(m *Menu) {
	none := len(a.wins) == 0
	sub := a.mode == MDISubWindows
	m.Items = []*MenuItem{
		{Text: "&Cascade", Disabled: none || !sub, OnClick: a.Cascade},
		{Text: "&Tile", Disabled: none || !sub, OnClick: a.Tile},
		{Text: "Tile &Rows", Disabled: none || !sub, OnClick: a.TileRows},
		{Text: "Tile C&olumns", Disabled: none || !sub, OnClick: a.TileColumns},
		Sep(),
		{Text: "Cl&ose", Shortcut: "Ctrl+F4", Disabled: none, OnClick: func() { a.CloseActive() }},
		{Text: "Close &All", Disabled: none, OnClick: func() { a.CloseAll() }},
		Sep(),
		{Text: "Ne&xt", Shortcut: "Ctrl+F6", Disabled: len(a.wins) < 2, OnClick: a.ActivateNext},
		{Text: "Pre&vious", Shortcut: "Ctrl+Shift+F6", Disabled: len(a.wins) < 2, OnClick: a.ActivatePrevious},
		Sep(),
		{Text: "&Tabbed View", Checkable: true, Checked: !sub, OnClick: func() {
			if a.mode == MDITabbed {
				a.SetViewMode(MDISubWindows)
			} else {
				a.SetViewMode(MDITabbed)
			}
			a.refreshMenus()
		}},
	}
	if !none {
		m.Items = append(m.Items, Sep())
	}
	for i, w := range a.wins {
		w := w
		label := w.title
		if i < 9 {
			label = "&" + strconv.Itoa(i+1) + " " + label
		}
		m.Items = append(m.Items, RadioItem(label, "mdi-windows", w == a.active, func() { a.Activate(w) }))
	}
}

// ---- accessibility -----------------------------------------------------

// Describe: a desktop pane, the area that holds windows.
func (a *MDIArea) Describe(n *a11y.Node) {
	n.Role = a11y.RoleDesktopPane
	if n.Name == "" {
		n.Name = "Documents"
	}
}

// ==== the windows =======================================================

// Area is the area w is in.
func (w *MDIWindow) Area() *MDIArea { return w.area }

// Title is the window's caption.
func (w *MDIWindow) Title() string { return w.title }

// SetTitle recaptions the window (and its tab, and the Window menu).
func (w *MDIWindow) SetTitle(s string) {
	if w.title == s {
		return
	}
	w.title = s
	if a := w.area; a != nil {
		if i := a.indexOf(w); i >= 0 && a.tabs != nil && i < a.tabs.Len() {
			a.tabs.SetTitle(i, s)
		}
		a.changed()
	}
	w.Invalidate()
}

// Content is what the window holds.
func (w *MDIWindow) Content() widget.Component { return w.client.content }

// IsMinimized and IsMaximized report the window's size state.
func (w *MDIWindow) IsMinimized() bool { return w.state == mdiMinimized }
func (w *MDIWindow) IsMaximized() bool { return w.state == mdiMaximized }

// IsActive reports whether w is its area's active window.
func (w *MDIWindow) IsActive() bool { return w.area != nil && w.area.active == w }

// Geometry is the window's place and size in its area when it is neither
// minimised nor maximised, in logical pixels.
func (w *MDIWindow) Geometry() paintengine2d.Rect {
	s := w.scale()
	return paintengine2d.Rect{Min: w.normal.Min.Mul(1 / s), Max: w.normal.Max.Mul(1 / s)}
}

// SetGeometry places the window at r, in logical pixels of its area.
func (w *MDIWindow) SetGeometry(r paintengine2d.Rect) {
	s := w.scale()
	w.normal = pxRect(paintengine2d.Rect{Min: r.Min.Mul(s), Max: r.Max.Mul(s)})
	if w.area != nil {
		w.area.RequestLayout()
		w.area.Invalidate()
	}
}

func (w *MDIWindow) scale() float32 {
	if w.area != nil && w.area.scale > 0 {
		return w.area.scale
	}
	if s := style.LookScale(w.Look()); s > 0 {
		return s
	}
	return 1
}

// Minimize shrinks the window to a caption strip along the area's foot.
func (w *MDIWindow) Minimize() {
	if w.state == mdiMinimized || w.Features&MDIMinimizable == 0 {
		return
	}
	w.restore = w.state
	w.state = mdiMinimized
	w.relayout()
}

// Maximize makes the window fill the area.
func (w *MDIWindow) Maximize() {
	if w.state == mdiMaximized || w.Features&MDIMaximizable == 0 {
		return
	}
	w.state = mdiMaximized
	if w.area != nil {
		w.area.activate(w, !widget.FocusWithin(w))
	}
	w.relayout()
}

// Restore returns a minimised or maximised window to its place: a
// minimised one to what it was before (maximised, if it was).
func (w *MDIWindow) Restore() {
	switch w.state {
	case mdiMinimized:
		w.state = w.restore
	case mdiMaximized:
		w.state = mdiNormal
	default:
		return
	}
	if w.area != nil {
		w.area.activate(w, !widget.FocusWithin(w))
	}
	w.relayout()
}

// ToggleMaximize maximises a window or restores a maximised one.
func (w *MDIWindow) ToggleMaximize() {
	if w.state == mdiNormal {
		w.Maximize()
	} else {
		w.Restore()
	}
}

func (w *MDIWindow) relayout() {
	if a := w.area; a != nil {
		a.refreshMenus()
		a.RequestLayout()
		a.Invalidate()
	}
}

// Close asks the window to close — OnCloseRequest may refuse — and
// reports whether it did.
func (w *MDIWindow) Close() bool {
	if w.area == nil || w.Features&MDIClosable == 0 {
		return false
	}
	if w.OnCloseRequest != nil && !w.OnCloseRequest() {
		return false
	}
	w.Dismiss()
	return true
}

// Dismiss closes the window without asking.
func (w *MDIWindow) Dismiss() {
	a := w.area
	if a == nil {
		return
	}
	a.remove(w)
	w.area = nil
	if w.OnClose != nil {
		w.OnClose()
	}
}

// SetInitialFocus names what takes the keyboard when the window is
// activated and has not had it yet — the document rather than the tool
// bar above it; otherwise the window's first control does. Called on the
// active window it moves the keyboard there now.
func (w *MDIWindow) SetInitialFocus(c widget.Component) {
	w.initialFocus = c
	if c != nil && w.IsActive() && widget.FocusWithin(w) {
		if h := w.Host(); h != nil {
			h.RequestFocus(c)
			w.lastFocus = c
		}
	}
}

// takeFocus puts the keyboard in the window: where it last was, else on
// its initial focus or its first control, else on the window itself.
func (w *MDIWindow) takeFocus() {
	h := w.Host()
	if h == nil || widget.FocusWithin(w) {
		return
	}
	if w.lastFocus != nil && widget.LiveUnder(w.lastFocus, w) && w.lastFocus.Visible() && w.lastFocus.Enabled() {
		h.RequestFocus(w.lastFocus)
		return
	}
	if f := w.initialFocus; f != nil && widget.LiveUnder(f, w) && f.Visible() && f.Enabled() {
		h.RequestFocus(f)
		widget.MarkKeyboardFocus(f)
		return
	}
	if widget.FocusFirstIn(w.client) {
		return
	}
	h.RequestFocus(w)
}

// WantsFocus: a window whose content takes no keyboard focus takes it
// itself, so Ctrl+Tab and Ctrl+F4 still reach its area.
func (w *MDIWindow) WantsFocus() bool {
	return len(widget.Focusables(w.client)) == 0
}

// ---- frame geometry ----------------------------------------------------

// decoState is the frame state w paints in.
func (w *MDIWindow) decoState() style.DecorationState {
	return style.DecorationState{Active: w.IsActive() && widget.WindowActive(w), Maximized: w.state == mdiMaximized}
}

func (w *MDIWindow) spec() style.DecorationSpec {
	return style.DecorationOf(w.Look(), w.decoState())
}

func (w *MDIWindow) specFor(s mdiState, active bool) style.DecorationSpec {
	return style.DecorationOf(w.Look(), style.DecorationState{Active: active, Maximized: s == mdiMaximized})
}

// captionH is the caption band's height: the look's, room for its
// buttons and, in a merged frame, a line of title.
func (w *MDIWindow) captionH() float32 { return w.captionHFor(w.spec()) }

func (w *MDIWindow) captionHFor(s style.DecorationSpec) float32 {
	lk := w.Look()
	bh := max(s.Button.Y, s.CloseButton.Y)
	if bh <= 0 {
		bh = float32(math.Round(float64(style.Dip(lk, 24))))
	}
	m := max(s.Caption, s.ButtonPad.Top+bh)
	if !s.Stacked {
		if f := lk.BoldFont(); f != nil {
			m = max(m, float32(math.Ceil(float64(f.Height()+style.Dip(lk, 10)))))
		}
	}
	return float32(math.Ceil(float64(m) - 1e-3))
}

// margin is the room around the visible window its shadow takes, and
// never less than a resize handle's grip.
func (w *MDIWindow) margin() style.Insets {
	if w.area == nil || w.area.mode == MDITabbed || w.state != mdiNormal {
		return style.Insets{}
	}
	sh := w.spec().Shadow
	g := float32(math.Round(float64(style.Dip(w.Look(), 4))))
	return style.Insets{Top: max(sh.Top, g), Right: max(sh.Right, g), Bottom: max(sh.Bottom, g), Left: max(sh.Left, g)}
}

// outset is the bounds of a window visible at r: r and its margin.
func (w *MDIWindow) outset(r paintengine2d.Rect) paintengine2d.Rect {
	m := w.margin()
	return paintengine2d.Rect{Min: paintengine2d.Pt(r.Min.X-m.Left, r.Min.Y-m.Top), Max: paintengine2d.Pt(r.Max.X+m.Right, r.Max.Y+m.Bottom)}
}

// win is the visible window in local coordinates.
func (w *MDIWindow) win() paintengine2d.Rect {
	m := w.margin()
	return m.Apply(w.LocalBounds())
}

// frameW and frameH are what the frame adds to the content's size.
func (w *MDIWindow) frameW() float32 {
	b := w.spec().Border
	return b.Left + b.Right
}

func (w *MDIWindow) frameH() float32 {
	s := w.spec()
	return s.Border.Top + w.captionHFor(s) + s.CaptionGap + s.Border.Bottom
}

func (w *MDIWindow) minW() float32 {
	if w.MinSize.X > 0 {
		return style.Dip(w.Look(), w.MinSize.X)
	}
	return style.Dip(w.Look(), 160)
}

func (w *MDIWindow) minH() float32 {
	if w.MinSize.Y > 0 {
		return style.Dip(w.Look(), w.MinSize.Y)
	}
	return w.frameH() + style.Dip(w.Look(), 60)
}

// caption is the caption band in local coordinates.
func (w *MDIWindow) caption() paintengine2d.Rect {
	if w.area != nil && w.area.mode == MDITabbed {
		return paintengine2d.Rect{}
	}
	s := w.spec()
	wr := w.win()
	b := s.Border
	return paintengine2d.XYWH(wr.Min.X+b.Left, wr.Min.Y+b.Top, max(wr.Dx()-b.Left-b.Right, 0), w.captionHFor(s))
}

// contentRect is where the content goes, in local coordinates.
func (w *MDIWindow) contentRect() paintengine2d.Rect {
	if w.area != nil && w.area.mode == MDITabbed {
		return w.LocalBounds()
	}
	if w.state == mdiMinimized {
		return paintengine2d.Rect{}
	}
	s := w.spec()
	wr, cap := w.win(), w.caption()
	in := s.Border
	if s.Split {
		in = s.ContentBorder
	}
	top := cap.Max.Y + s.CaptionGap
	r := paintengine2d.Rect{Min: paintengine2d.Pt(wr.Min.X+in.Left, top), Max: paintengine2d.Pt(wr.Max.X-in.Right, wr.Max.Y-in.Bottom)}
	r.Max.X, r.Max.Y = max(r.Max.X, r.Min.X), max(r.Max.Y, r.Min.Y)
	return r
}

// buttons is the caption buttons on each side, as the area's layout (or
// the look's, or Windows') says, less those the features rule out.
func (w *MDIWindow) buttons() (left, right []platform.CaptionButton) {
	s := w.spec()
	l := platform.ButtonLayout{}
	if w.area != nil {
		l = w.area.ButtonLayout
	}
	if len(l.Left) == 0 && len(l.Right) == 0 {
		if s.Layout != "" {
			l = platform.ParseButtonLayout(s.Layout)
		} else {
			l = platform.ParseButtonLayout(":minimize,maximize,close")
		}
		l = mdiButtonsToSide(withSizeButtons(l), s.ButtonSide)
	}
	keep := func(bs []platform.CaptionButton) []platform.CaptionButton {
		var out []platform.CaptionButton
		for _, b := range bs {
			switch b {
			case platform.CaptionClose:
				if w.Features&MDIClosable == 0 {
					continue
				}
			case platform.CaptionMinimize:
				if w.Features&MDIMinimizable == 0 || w.state == mdiMinimized {
					continue
				}
			case platform.CaptionMaximize:
				if w.Features&MDIMaximizable == 0 {
					continue
				}
			case platform.CaptionNone:
				continue
			}
			out = append(out, b)
		}
		for len(out) > 0 && out[0] == platform.CaptionSpacer {
			out = out[1:]
		}
		for len(out) > 0 && out[len(out)-1] == platform.CaptionSpacer {
			out = out[:len(out)-1]
		}
		return out
	}
	return keep(l.Left), keep(l.Right)
}

// withSizeButtons adds minimise and maximise beside close where a layout
// leaves them out (GNOME's is close alone): a window inside a window is
// not the desktop's to minimise, and without them it could not be.
func withSizeButtons(l platform.ButtonLayout) platform.ButtonLayout {
	if l.Has(platform.CaptionMinimize) && l.Has(platform.CaptionMaximize) {
		return l
	}
	var add []platform.CaptionButton
	for _, b := range []platform.CaptionButton{platform.CaptionMinimize, platform.CaptionMaximize} {
		if !l.Has(b) {
			add = append(add, b)
		}
	}
	for i, b := range l.Left {
		if b == platform.CaptionClose {
			// Close at the left (the Mac): the others follow it.
			l.Left = append(append(append([]platform.CaptionButton(nil), l.Left[:i+1]...), add...), l.Left[i+1:]...)
			return l
		}
	}
	for i, b := range l.Right {
		if b == platform.CaptionClose {
			l.Right = append(append(append([]platform.CaptionButton(nil), l.Right[:i]...), add...), l.Right[i:]...)
			return l
		}
	}
	l.Right = append(l.Right, add...)
	return l
}

// mdiButtonsToSide moves every button to one side, the outermost staying
// outermost (the rule app.Window applies to a look's ButtonSide).
func mdiButtonsToSide(l platform.ButtonLayout, side style.ButtonSide) platform.ButtonLayout {
	turned := func(bs []platform.CaptionButton) []platform.CaptionButton {
		out := make([]platform.CaptionButton, len(bs))
		for i, b := range bs {
			out[len(bs)-1-i] = b
		}
		return out
	}
	switch side {
	case style.ButtonsLeft:
		if len(l.Right) > 0 {
			return platform.ButtonLayout{Left: append(append([]platform.CaptionButton(nil), l.Left...), turned(l.Right)...)}
		}
	case style.ButtonsRight:
		if len(l.Left) > 0 {
			return platform.ButtonLayout{Right: append(turned(l.Left), l.Right...)}
		}
	}
	return l
}

// mdiButton is a caption button and its box.
type mdiButton struct {
	k platform.CaptionButton
	r paintengine2d.Rect
}

// buttonRects are the caption buttons' boxes in local coordinates.
func (w *MDIWindow) buttonRects() []mdiButton {
	cap := w.caption()
	if cap.Empty() {
		return nil
	}
	s := w.spec()
	left, right := w.buttons()
	top := cap.Min.Y + s.ButtonPad.Top
	if s.CenterButtons && cap.Dy() > s.Caption {
		top += float32(math.Round(float64(cap.Dy()-s.Caption) * 0.5))
	}
	spacer := float32(math.Round(float64(style.Dip(w.Look(), 10))))
	size := func(b platform.CaptionButton) paintengine2d.Point {
		if b == platform.CaptionSpacer {
			return paintengine2d.Pt(spacer, 0)
		}
		sz := s.ButtonBox(style.CaptionButton(b))
		if sz.Y <= 0 {
			sz.Y = max(cap.Dy()-s.ButtonPad.Top, 0)
		}
		if sz.X <= 0 {
			sz.X = sz.Y
		}
		return sz
	}
	gapAt := func(a, b platform.CaptionButton) float32 {
		g := s.ButtonGap
		if a == platform.CaptionClose || b == platform.CaptionClose {
			g += s.CloseGap
		}
		return g
	}
	var out []mdiButton
	x := cap.Min.X + s.ButtonPad.Left
	for i, b := range left {
		if i > 0 {
			x += gapAt(left[i-1], b)
		}
		sz := size(b)
		if b != platform.CaptionSpacer {
			out = append(out, mdiButton{b, paintengine2d.XYWH(x, top, sz.X, sz.Y)})
		}
		x += sz.X
	}
	x = cap.Max.X - s.ButtonPad.Right
	for i := len(right) - 1; i >= 0; i-- {
		b := right[i]
		if i < len(right)-1 {
			x -= gapAt(b, right[i+1])
		}
		sz := size(b)
		x -= sz.X
		if b != platform.CaptionSpacer {
			out = append(out, mdiButton{b, paintengine2d.XYWH(x, top, sz.X, sz.Y)})
		}
	}
	return out
}

// titleRect is the caption's free space for the title. A look that
// centres its titles gets a box the title's width on the window's middle,
// pushed aside by the buttons where they leave no room there (as the
// header bar does).
func (w *MDIWindow) titleRect() paintengine2d.Rect {
	cap := w.caption()
	x0, x1 := cap.Min.X, cap.Max.X
	for _, b := range w.buttonRects() {
		if b.r.Min.X < (cap.Min.X+cap.Max.X)*0.5 {
			x0 = max(x0, b.r.Max.X)
		} else {
			x1 = min(x1, b.r.Min.X)
		}
	}
	r := paintengine2d.XYWH(x0, cap.Min.Y, max(x1-x0, 0), cap.Dy())
	if w.spec().CenterTitle {
		lk := w.Look()
		f := lk.BoldFont()
		if f == nil {
			f = lk.Font()
		}
		tw := min(f.Advance(w.title)+2*style.Dip(lk, 12), r.Dx())
		x := min(max((cap.Min.X+cap.Max.X-tw)*0.5, r.Min.X), r.Max.X-tw)
		r.Min.X, r.Max.X = x, x+tw
	}
	return r
}

// buttonAt is the caption button at local p (CaptionNone for none).
func (w *MDIWindow) buttonAt(p paintengine2d.Point) platform.CaptionButton {
	for _, b := range w.buttonRects() {
		if b.r.Contains(p) {
			return b.k
		}
	}
	return platform.CaptionNone
}

// edgesAt is the frame edges local p would resize (none when the window
// cannot be resized there).
func (w *MDIWindow) edgesAt(p paintengine2d.Point) style.Edges {
	if w.state != mdiNormal || w.Features&MDIResizable == 0 || w.area == nil || w.area.mode != MDISubWindows {
		return 0
	}
	wr := w.win()
	lk := w.Look()
	grip := max(w.spec().Border.Left, style.Dip(lk, 4))
	corner := style.Dip(lk, 14)
	var e style.Edges
	switch {
	case p.X < wr.Min.X+grip:
		e |= style.EdgeLeft
	case p.X >= wr.Max.X-grip:
		e |= style.EdgeRight
	}
	switch {
	case p.Y < wr.Min.Y+grip:
		e |= style.EdgeTop
	case p.Y >= wr.Max.Y-grip:
		e |= style.EdgeBottom
	}
	// Near a corner the grip widens along both edges, as a frame's does.
	if e&(style.EdgeLeft|style.EdgeRight) != 0 && e&(style.EdgeTop|style.EdgeBottom) == 0 {
		if p.Y < wr.Min.Y+corner {
			e |= style.EdgeTop
		} else if p.Y >= wr.Max.Y-corner {
			e |= style.EdgeBottom
		}
	}
	if e&(style.EdgeTop|style.EdgeBottom) != 0 && e&(style.EdgeLeft|style.EdgeRight) == 0 {
		if p.X < wr.Min.X+corner {
			e |= style.EdgeLeft
		} else if p.X >= wr.Max.X-corner {
			e |= style.EdgeRight
		}
	}
	return e
}

func (w *MDIWindow) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(w.normal.Dx(), w.normal.Dy()))
}

func (w *MDIWindow) Arrange(r paintengine2d.Rect) {
	w.SetBounds(r)
	cr := w.contentRect()
	w.client.SetVisible(!cr.Empty())
	w.client.Arrange(cr)
}

// HitTest: the resize band belongs to the frame even where the content
// runs to the window's edge, and nothing outside the band takes the
// pointer (the shadow is not the window).
func (w *MDIWindow) HitTest(local paintengine2d.Point) widget.Component {
	if !w.Visible() || !w.LocalBounds().Contains(local) {
		return nil
	}
	if w.edgesAt(local) != 0 {
		return w
	}
	g := float32(math.Round(float64(style.Dip(w.Look(), 4))))
	if !w.win().Inset(-g).Contains(local) {
		return nil
	}
	return w.Base.HitTest(local)
}

// PaintClip cuts a window whose look gives windows a silhouette (a shaped
// skin) to it.
func (w *MDIWindow) PaintClip(ctx *paintengine2d.Context) {
	if s := w.silhouette(); s != nil && s.Path != nil {
		rule := paintengine2d.FillNonZero
		if s.EvenOdd {
			rule = paintengine2d.FillEvenOdd
		}
		ctx.ClipPathRule(s.Path, rule)
	}
}

func (w *MDIWindow) silhouette() *style.Silhouette {
	if w.area != nil && w.area.mode == MDITabbed {
		return nil
	}
	wr := w.win()
	cap := w.caption()
	st := w.decoState()
	s := style.WindowShapeOf(w.Look(), style.DecorationFrame{Window: paintengine2d.XYWH(0, 0, wr.Dx(), wr.Dy()), Caption: cap.Translate(paintengine2d.Pt(-wr.Min.X, -wr.Min.Y))}, st)
	if s == nil || s.Path == nil {
		return nil
	}
	p := s.Path.Clone()
	p.Transform(paintengine2d.Translation(wr.Min.X, wr.Min.Y))
	return &style.Silhouette{Path: p, EvenOdd: s.EvenOdd}
}

func (w *MDIWindow) Paint(ctx *paintengine2d.Context) {
	lk := w.Look()
	if w.area != nil && w.area.mode == MDITabbed {
		ctx.DrawRect(w.LocalBounds(), paintengine2d.Fill(lk.Palette().Background))
		return
	}
	st := w.decoState()
	s := style.DecorationOf(lk, st)
	wr := w.win()
	if w.state == mdiNormal && w.silhouette() == nil {
		style.DrawDecorationShadowOf(lk, ctx, wr, st)
	}
	r := s.Radius
	ctx.DrawRoundRectCorners(wr, r[0], r[1], r[2], r[3], paintengine2d.Fill(lk.Palette().Background))
	cap := w.caption()
	style.DrawDecorationOf(lk, ctx, style.DecorationFrame{Window: wr, Caption: cap}, st)
	style.DrawCaptionTitleOf(lk, ctx, w.titleRect(), w.title, st)
	bst := st
	if w.state == mdiMinimized {
		// A minimised window's maximise button restores it, and shows so.
		bst.Maximized = true
	}
	for _, b := range w.buttonRects() {
		var cs style.ControlState
		if b.k == w.hot {
			cs |= style.StateHovered
		}
		if b.k == w.down && b.k == w.hot {
			cs |= style.StatePressed
		}
		style.DrawCaptionButtonOf(lk, ctx, b.r, style.CaptionButton(b.k), cs, bst)
	}
	if a := w.area; a != nil && a.kb == w {
		// A keyboard move or size in progress: the frame's outline.
		ctx.DrawRect(wr.Inset(1), paintengine2d.StrokePaint(lk.Palette().Focus, max(2, style.Dip(lk, 2))))
	}
}

// ---- pointer -----------------------------------------------------------

// CursorAt: the resize arrows over the frame's edges and corners.
func (w *MDIWindow) CursorAt(p paintengine2d.Point) platform.Cursor {
	return edgeCursor(w.edgesAt(p))
}

func edgeCursor(e style.Edges) platform.Cursor {
	switch e {
	case style.EdgeTop:
		return platform.CursorResizeN
	case style.EdgeBottom:
		return platform.CursorResizeS
	case style.EdgeLeft:
		return platform.CursorResizeW
	case style.EdgeRight:
		return platform.CursorResizeE
	case style.EdgeTop | style.EdgeLeft:
		return platform.CursorResizeNW
	case style.EdgeTop | style.EdgeRight:
		return platform.CursorResizeNE
	case style.EdgeBottom | style.EdgeLeft:
		return platform.CursorResizeSW
	case style.EdgeBottom | style.EdgeRight:
		return platform.CursorResizeSE
	}
	return platform.CursorDefault
}

// KeyPress: with the keyboard on the window itself (a minimised one, or
// one with nothing to type in), Return restores or maximises it.
func (w *MDIWindow) KeyPress(e widget.KeyEvent) bool {
	if !w.Focused() || e.Key != platform.KeyReturn && e.Key != platform.KeySpace {
		return false
	}
	switch w.state {
	case mdiMinimized:
		w.Restore()
		return true
	}
	return false
}

// FocusOnClick is false: pressing the frame activates the window without
// taking the keyboard from the control it was on.
func (w *MDIWindow) FocusOnClick() bool { return false }

// toArea is local p in the area's coordinates.
func (w *MDIWindow) toArea(p paintengine2d.Point) paintengine2d.Point {
	return p.Add(w.Bounds().Min)
}

func (w *MDIWindow) MousePress(e widget.MouseEvent) bool {
	a := w.area
	if a == nil {
		return false
	}
	if a.kb != nil {
		a.endKeyboard(true)
	}
	a.Activate(w)
	if e.Button == platform.ButtonRight && w.caption().Contains(e.Pos) {
		w.showMenu(e.Pos)
		return true
	}
	if e.Button != platform.ButtonLeft {
		return true
	}
	if b := w.buttonAt(e.Pos); b != platform.CaptionNone {
		w.down, w.hot = b, b
		w.Invalidate()
		return true
	}
	if edges := w.edgesAt(e.Pos); edges != 0 {
		w.drag, w.edges = 2, edges
		w.grab, w.start = w.toArea(e.Pos), w.normal
		return true
	}
	if w.caption().Contains(e.Pos) {
		now := time.Now()
		if now.Sub(w.lastClick) < doubleClickInterval {
			w.lastClick = time.Time{}
			switch {
			case w.state == mdiMinimized:
				w.Restore()
			case w.Features&MDIMaximizable != 0:
				w.ToggleMaximize()
			}
			return true
		}
		w.lastClick = now
		if w.state == mdiNormal && w.Features&MDIMovable != 0 && a.mode == MDISubWindows {
			w.drag = 1
			w.grab, w.start = w.toArea(e.Pos), w.normal
		}
		return true
	}
	return true
}

func (w *MDIWindow) MouseMove(e widget.MouseEvent) bool {
	a := w.area
	if a == nil {
		return false
	}
	switch w.drag {
	case 1:
		d := w.toArea(e.Pos).Sub(w.grab)
		w.normal = a.clampMove(w, w.start.Translate(d))
		a.RequestLayout()
		a.Invalidate()
		return true
	case 2:
		w.normal = w.resized(w.toArea(e.Pos).Sub(w.grab))
		a.RequestLayout()
		a.Invalidate()
		return true
	}
	hot := platform.CaptionNone
	if w.down != platform.CaptionNone || e.Button == platform.ButtonNone {
		hot = w.buttonAt(e.Pos)
	}
	if hot != w.hot {
		w.hot = hot
		w.Invalidate()
	}
	return w.down != platform.CaptionNone
}

// resized is the drag's start geometry with its edges moved by d, never
// under the least size nor past the area's edges.
func (w *MDIWindow) resized(d paintengine2d.Point) paintengine2d.Rect {
	r := w.start
	b := w.area.work()
	if w.edges&style.EdgeLeft != 0 {
		r.Min.X = min(max(r.Min.X+d.X, b.Min.X-r.Dx()), r.Max.X-w.minW())
	}
	if w.edges&style.EdgeRight != 0 {
		r.Max.X = max(r.Max.X+d.X, r.Min.X+w.minW())
	}
	if w.edges&style.EdgeTop != 0 {
		r.Min.Y = min(max(r.Min.Y+d.Y, b.Min.Y), r.Max.Y-w.minH())
	}
	if w.edges&style.EdgeBottom != 0 {
		r.Max.Y = max(r.Max.Y+d.Y, r.Min.Y+w.minH())
	}
	return pxRect(r)
}

func (w *MDIWindow) MouseRelease(e widget.MouseEvent) bool {
	if w.drag != 0 {
		w.drag = 0
		return true
	}
	b := w.down
	w.down = platform.CaptionNone
	w.Invalidate()
	if b != platform.CaptionNone && b == w.buttonAt(e.Pos) {
		w.runButton(b)
	}
	return true
}

func (w *MDIWindow) MouseExit() {
	if w.hot != platform.CaptionNone {
		w.hot = platform.CaptionNone
		w.Invalidate()
	}
	w.Base.MouseExit()
}

// runButton does what caption button b does.
func (w *MDIWindow) runButton(b platform.CaptionButton) {
	switch b {
	case platform.CaptionClose:
		w.Close()
	case platform.CaptionMinimize:
		w.Minimize()
	case platform.CaptionMaximize:
		if w.state == mdiNormal {
			w.Maximize()
		} else {
			w.Restore()
		}
	case platform.CaptionMenu:
		w.showMenu(w.menuAnchor())
	}
}

// menuAnchor is where the window menu drops from: the menu button, or the
// caption's left end.
func (w *MDIWindow) menuAnchor() paintengine2d.Point {
	for _, b := range w.buttonRects() {
		if b.k == platform.CaptionMenu {
			return paintengine2d.Pt(b.r.Min.X, b.r.Max.Y)
		}
	}
	c := w.caption()
	return paintengine2d.Pt(c.Min.X, c.Max.Y)
}

// showMenu opens the window menu at local p: Restore, Move, Size,
// Minimise, Maximise, Close, and Next.
func (w *MDIWindow) showMenu(p paintengine2d.Point) {
	a := w.area
	if a == nil {
		return
	}
	normal := w.state == mdiNormal
	items := []*MenuItem{
		{Text: "&Restore", Disabled: normal, OnClick: w.Restore},
		{Text: "&Move", Disabled: !normal || w.Features&MDIMovable == 0, OnClick: func() { a.startKeyboard(w, false) }},
		{Text: "&Size", Disabled: !normal || w.Features&MDIResizable == 0, OnClick: func() { a.startKeyboard(w, true) }},
		{Text: "Mi&nimize", Disabled: w.state == mdiMinimized || w.Features&MDIMinimizable == 0, OnClick: w.Minimize},
		{Text: "Ma&ximize", Disabled: w.state == mdiMaximized || w.Features&MDIMaximizable == 0, OnClick: w.Maximize},
		Sep(),
		{Text: "&Close", Shortcut: "Ctrl+F4", Disabled: w.Features&MDIClosable == 0, OnClick: func() { w.Close() }},
		Sep(),
		{Text: "Nex&t", Shortcut: "Ctrl+F6", Disabled: len(a.wins) < 2, OnClick: a.ActivateNext},
	}
	ShowContextMenu(w, widget.DeviceOrigin(w).Add(p), items...)
}

// ---- accessibility -----------------------------------------------------

// Describe: an internal frame named by its title; its state in the
// description, as a screen reader should say it.
func (w *MDIWindow) Describe(n *a11y.Node) {
	n.Role = a11y.RoleInternalFrame
	if n.Name == "" {
		n.Name = w.title
	}
	switch w.state {
	case mdiMinimized:
		n.Description = "minimized"
	case mdiMaximized:
		n.Description = "maximized"
	}
	if w.IsActive() {
		n.State |= a11y.StateSelected
	}
	n.State |= a11y.StateSelectable
	n.Actions = n.Actions.With(a11y.ActionFocus).With(a11y.ActionDefault)
}

// captionName is what caption button b is called.
func (w *MDIWindow) captionName(b platform.CaptionButton) string {
	switch b {
	case platform.CaptionClose:
		return "Close"
	case platform.CaptionMinimize:
		return "Minimize"
	case platform.CaptionMaximize:
		if w.state != mdiNormal {
			return "Restore"
		}
		return "Maximize"
	case platform.CaptionMenu:
		return "Window Menu"
	}
	return ""
}

// AccessibleItems are the caption buttons.
func (w *MDIWindow) AccessibleItems() []*a11y.Node {
	bs := w.buttonRects()
	out := make([]*a11y.Node, 0, len(bs))
	o := widget.DeviceOrigin(w)
	for i, b := range bs {
		out = append(out, &a11y.Node{ID: widget.ItemID(w, i), Role: a11y.RoleButton, Name: w.captionName(b.k),
			Bounds: b.r.Translate(o), Actions: a11y.Actions(0).With(a11y.ActionDefault)})
	}
	return out
}

// AccessibleAction presses caption button item, or (item < 0) activates
// the window.
func (w *MDIWindow) AccessibleAction(item int, act a11y.Action) bool {
	if item < 0 {
		if w.area == nil || (act != a11y.ActionDefault && act != a11y.ActionFocus) {
			return false
		}
		w.area.Activate(w)
		return true
	}
	bs := w.buttonRects()
	if item >= len(bs) || act != a11y.ActionDefault {
		return false
	}
	w.runButton(bs[item].k)
	return true
}

// ---- the client --------------------------------------------------------

func (c *mdiClient) Measure(cons layout.Constraints) paintengine2d.Point {
	if c.content == nil {
		return cons.Constrain(paintengine2d.Point{})
	}
	return c.content.Measure(cons)
}

func (c *mdiClient) Arrange(r paintengine2d.Rect) {
	c.SetBounds(r)
	if c.content != nil {
		c.content.Arrange(c.LocalBounds())
	}
}

// PaintClip rounds the content's lower corners with the frame's, where
// the frame has no border of its own there to cover them.
func (c *mdiClient) PaintClip(ctx *paintengine2d.Context) {
	w := c.win
	if w.area == nil || w.area.mode == MDITabbed || w.state != mdiNormal {
		return
	}
	s := w.spec()
	br, bl := s.Radius[2]-s.Border.Bottom, s.Radius[3]-s.Border.Bottom
	if br <= 0 && bl <= 0 {
		return
	}
	b := c.LocalBounds()
	p := paintengine2d.NewPath()
	p.AddRoundRectCorners(b, 0, 0, max(br, 0), max(bl, 0))
	ctx.ClipPath(p)
}
