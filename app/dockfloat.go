package app

import (
	"errors"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/dock"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// errNoApplication is returned when an opener outlives its application.
var errNoApplication = errors.New("app: the dock host has no application to open windows in")

// DockWindows gives a dock host real windows for its floating panels. The
// windows are ordinary toplevels of a, so they follow the app's decoration
// setting — the toolkit's frame or the desktop's, exactly as the main
// window does (see docs/decorations.md) — and the desktop moves, resizes
// and stacks them.
//
//	host := dock.NewHost(editor)
//	host.SetWindowOpener(app.DockWindows(a))
//
// Without an opener a host's panels cannot float and no float button
// appears, so a headless app needs no special case.
func DockWindows(a *Application) dock.WindowOpener { return &dockOpener{app: a} }

// DockHost wires a host into win: panels float as windows of win's app,
// and closing win docks them all back first, so no panel is left in a
// window of its own keeping a finished app alive.
//
// It goes through [Window.SetOnCloseRequest], so it sees the desktop's
// close button, Alt+F4 and the window menu; an app that calls
// [Window.Close] itself should call [dock.Host.CloseFloating] first.
func DockHost(win *Window, host *dock.Host) {
	if win == nil || host == nil || win.app == nil {
		return
	}
	host.SetWindowOpener(DockWindows(win.app))
	prev := win.onCloseRequest
	win.SetOnCloseRequest(func() bool {
		host.CloseFloating()
		if prev != nil {
			return prev()
		}
		return true
	})
}

// dockOpener opens one toplevel per floating panel.
type dockOpener struct{ app *Application }

// OpenFloat implements dock.WindowOpener.
func (o *dockOpener) OpenFloat(title string, geom paintengine2d.Rect) (dock.FloatWindow, error) {
	if o.app == nil {
		return nil, errNoApplication
	}
	w, h := int(geom.Dx()), int(geom.Dy())
	if w < 1 || h < 1 {
		w, h = 320, 400
	}
	// The dock lays out in device pixels; a window's size is stated in
	// logical ones. X and Y are root pixels either way.
	sc := o.app.Scale()
	opts := platform.WindowOptions{
		Title: title,
		Width: platform.LogicalPixels(w, sc), Height: platform.LogicalPixels(h, sc),
		MinWidth: platform.LogicalPixels(120, sc), MinHeight: platform.LogicalPixels(80, sc),
		X: int(geom.Min.X), Y: int(geom.Min.Y),
	}
	win, err := o.app.NewWindow(opts)
	if err != nil {
		return nil, err
	}
	// The desktop's close button hides the panel rather than destroying
	// the window, so showing the panel again brings the same window back.
	win.SetCloseHides(true)
	return &dockWindow{win: win, want: geom}, nil
}

// dockWindow is one floating panel's toplevel.
type dockWindow struct {
	win *Window
	// want is the geometry the window was asked for, which is what the
	// origin falls back to where the backend is not told where the
	// desktop put the window (Wayland, always). The size is always the
	// real one.
	want paintengine2d.Rect
}

func (d *dockWindow) SetTitle(title string) { d.win.SetTitle(title) }

func (d *dockWindow) SetContent(c widget.Component) { d.win.SetContent(c) }

func (d *dockWindow) SetOnCloseRequest(fn func() bool) { d.win.SetOnCloseRequest(fn) }

// StartMove hands the window to the desktop's interactive move.
func (d *dockWindow) StartMove() bool { return d.win.StartMove() }

func (d *dockWindow) Show() { d.win.Show() }

func (d *dockWindow) Hide() { d.win.Hide() }

func (d *dockWindow) Raise() { d.win.Raise() }

func (d *dockWindow) Close() { d.win.Close() }

// TearOffWindow is the toplevel as a drag can carry it, so dragging the
// panel's title bar over its host docks it back there.
func (d *dockWindow) TearOffWindow() widget.TearOffWindow {
	if d.win == nil || d.win.Closed() {
		return nil
	}
	return d.win
}

// Geometry is the window's real size, at the position the desktop put it
// where the backend is told (X11) and the one it was asked for where it
// is not (Wayland, where a toplevel has no position at all).
func (d *dockWindow) Geometry() paintengine2d.Rect {
	if d.win == nil || d.win.Closed() {
		return d.want
	}
	// Root device pixels, to go with the position below.
	w, h := d.win.PixelSize()
	if w < 1 || h < 1 {
		return d.want
	}
	at := d.want.Min
	if x, y, ok := d.win.Position(); ok {
		at = paintengine2d.Pt(float32(x), float32(y))
	}
	return paintengine2d.XYWH(at.X, at.Y, float32(w), float32(h))
}
