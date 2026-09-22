package platform

import (
	"errors"

	"github.com/codemodify/paintengine2d"
)

// Popups on the offscreen desktop: off unless a test turns them on
// (SimulatePopups), because a headless window is exactly where the app
// keeps drawing its popups inside the window. Turned on, the desktop places
// them the way X11 does — SolvePopup within a work area the test chooses —
// and every event injected into one arrives on its root window.

// offscreenPopups is one Offscreen's part in the simulation.
type offscreenPopups struct {
	// On a top level: whether popups open as surfaces (on), whether the
	// desktop refuses every one it is asked for (refuse), the work area in
	// the desktop's device pixels, and how many were opened.
	on     bool
	refuse bool
	area   FrameRect
	opened int
	// On a popup: what it hangs from and where it went.
	popup *offscreenPopup
	// On either: the popups open from it, oldest first.
	kids []*Offscreen
}

type offscreenPopup struct {
	root, parent *Offscreen
	opts         PopupOptions
	placed       FrameRect
	grabbed      bool
}

// errPopupsOff is OpenPopup on an offscreen desktop that does not simulate
// popups (the default), or one told to refuse them.
var errPopupsOff = errors.New("platform: this desktop does not open popups")

// SimulatePopups makes the offscreen desktop open popups as surfaces of
// their own, placed within area — the work area, in the desktop's device
// pixels, the same unit the window's position is in. An empty area turns
// the simulation off again.
func (o *Offscreen) SimulatePopups(area FrameRect) {
	if o == nil {
		return
	}
	o.pops.on = area.W > 0 && area.H > 0
	o.pops.area = area
}

// SimulatePopupsRefused makes the desktop say it opens popups and then
// refuse every one — what a compositor that will not map a popup does —
// so a test can watch the app fall back to drawing them inside the window.
func (o *Offscreen) SimulatePopupsRefused(on bool) {
	if o != nil {
		o.pops.refuse = on
	}
}

// PopupsOpened is how many popups have been opened from this window and its
// popups (tests).
func (o *Offscreen) PopupsOpened() int {
	if o == nil {
		return 0
	}
	return o.popRoot().pops.opened
}

// Popups is the popups open from this surface, oldest first (tests).
func (o *Offscreen) Popups() []*Offscreen {
	if o == nil {
		return nil
	}
	return append([]*Offscreen(nil), o.pops.kids...)
}

// PopupOptions is what a popup was opened with (zero for a top level).
func (o *Offscreen) PopupOptions() PopupOptions {
	if o == nil || o.pops.popup == nil {
		return PopupOptions{}
	}
	return o.pops.popup.opts
}

// SimulatePopupDone takes the popup down the way a compositor does when the
// user clicks another application: its root window hears EventPopupDone.
func (o *Offscreen) SimulatePopupDone() {
	if o == nil || o.pops.popup == nil || o.closed {
		return
	}
	r := o.pops.popup.root
	r.queue = append(r.queue, Event{Kind: EventPopupDone, Popup: o})
}

// popRoot is the top level o belongs to (o itself for a top level).
func (o *Offscreen) popRoot() *Offscreen {
	if o.pops.popup != nil && o.pops.popup.root != nil {
		return o.pops.popup.root
	}
	return o
}

// PopupsSupported reports whether the simulated desktop opens popups
// (PopupOpener).
func (o *Offscreen) PopupsSupported() bool {
	return o != nil && !o.closed && o.popRoot().pops.on
}

// OpenPopup opens a popup from o, placed by SolvePopup in the simulated
// work area (PopupOpener).
func (o *Offscreen) OpenPopup(opts PopupOptions) (PopupSurface, error) {
	if !o.PopupsSupported() {
		return nil, errPopupsOff
	}
	root := o.popRoot()
	if root.pops.refuse {
		return nil, errPopupsOff
	}
	pl := opts.Placement
	p := NewOffscreen(WindowOptions{Width: max(pl.W, 1), Height: max(pl.H, 1), Scale: root.Scale(), Popup: true})
	p.pops.popup = &offscreenPopup{root: root, parent: o, opts: opts, grabbed: opts.Role == PopupRoleMenu}
	p.SetFrame(opts.Frame)
	p.place(pl)
	o.pops.kids = append(o.pops.kids, p)
	root.pops.opened++
	return p, nil
}

// place solves p within the work area and sizes the popup to the answer.
func (o *Offscreen) place(pl PopupPlacement) {
	pp := o.pops.popup
	area, _ := pp.root.PopupWorkArea()
	pp.opts.Placement = pl
	pp.placed = SolvePopup(pl, area)
	if o.frame.geomLW != pp.placed.W || o.frame.geomLH != pp.placed.H {
		o.frame.geomLW, o.frame.geomLH = pp.placed.W, pp.placed.H
		o.frame.geomW = DevicePixels(pp.placed.W, o.Scale())
		o.frame.geomH = DevicePixels(pp.placed.H, o.Scale())
		o.resizeSurface()
	}
}

// Reposition places the popup afresh (PopupSurface).
func (o *Offscreen) Reposition(pl PopupPlacement) bool {
	if o == nil || o.pops.popup == nil || o.closed {
		return false
	}
	o.place(pl)
	return true
}

// Placed is where the simulated desktop put the popup (PopupSurface).
func (o *Offscreen) Placed() FrameRect {
	if o == nil || o.pops.popup == nil {
		return FrameRect{}
	}
	return o.pops.popup.placed
}

// Origin is the popup's buffer origin in its root's buffer (PopupSurface).
func (o *Offscreen) Origin() paintengine2d.Point {
	if o == nil || o.pops.popup == nil {
		return paintengine2d.Point{}
	}
	r := o.pops.popup.root
	m := r.frame.frame.Margin
	win := paintengine2d.Pt(float32(m.Left), float32(m.Top))
	return PopupOrigin(win, o.pops.popup.placed, o.frame.frame.Margin, r.Scale())
}

// Root is the top level the popup belongs to (PopupSurface).
func (o *Offscreen) Root() Surface {
	if o == nil {
		return nil
	}
	return o.popRoot()
}

// Grabbed reports whether the popup took the pointer and the keyboard, as
// a menu does and a tooltip does not (tests).
func (o *Offscreen) Grabbed() bool {
	return o != nil && o.pops.popup != nil && o.pops.popup.grabbed && !o.closed
}

// PopupWorkArea is the simulated work area in logical pixels relative to
// the window's visible box — where its popups may go (PopupWorkArea).
func (o *Offscreen) PopupWorkArea() (FrameRect, bool) {
	r := o.popRoot()
	if !r.pops.on {
		return FrameRect{}, false
	}
	sc := r.Scale()
	a := r.pops.area
	x0 := ceilDiv(a.X-r.posX, sc)
	y0 := ceilDiv(a.Y-r.posY, sc)
	x1 := floorDiv(a.X+a.W-r.posX, sc)
	y1 := floorDiv(a.Y+a.H-r.posY, sc)
	return FrameRect{X: x0, Y: y0, W: max(x1-x0, 0), H: max(y1-y0, 0)}, true
}

// closePopups closes every popup open from o, the newest first, as a
// compositor insists a nested popup goes before its parent.
func (o *Offscreen) closePopups() {
	for i := len(o.pops.kids) - 1; i >= 0; i-- {
		_ = o.pops.kids[i].Close()
	}
	o.pops.kids = nil
	if pp := o.pops.popup; pp != nil && pp.parent != nil {
		kids := pp.parent.pops.kids
		for i, k := range kids {
			if k == o {
				pp.parent.pops.kids = append(kids[:i:i], kids[i+1:]...)
				break
			}
		}
	}
}

var (
	_ PopupOpener   = (*Offscreen)(nil)
	_ PopupSurface  = (*Offscreen)(nil)
	_ PopupWorkArea = (*Offscreen)(nil)
)
