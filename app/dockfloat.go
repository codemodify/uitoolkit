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
	opts := platform.WindowOptions{
		Title: title, Width: w, Height: h,
		MinWidth: 120, MinHeight: 80,
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
	// want is the geometry the window was asked for. A client is not told
	// where the desktop actually put it — on Wayland it cannot even ask —
	// so the origin a layout saves is the one we asked for, while the size
	// is the real one.
	want paintengine2d.Rect
}

func (d *dockWindow) SetTitle(title string) { d.win.SetTitle(title) }

func (d *dockWindow) SetContent(c widget.Component) { d.win.SetContent(c) }

func (d *dockWindow) SetOnCloseRequest(fn func() bool) { d.win.SetOnCloseRequest(fn) }

func (d *dockWindow) Show() { d.win.Show() }

func (d *dockWindow) Hide() { d.win.Hide() }

func (d *dockWindow) Raise() { d.win.Raise() }

func (d *dockWindow) Close() { d.win.Close() }

// Geometry is the window's real size at the position it was asked for.
func (d *dockWindow) Geometry() paintengine2d.Rect {
	if d.win == nil || d.win.Closed() {
		return d.want
	}
	w, h := d.win.Size()
	if w < 1 || h < 1 {
		return d.want
	}
	return paintengine2d.XYWH(d.want.Min.X, d.want.Min.Y, float32(w), float32(h))
}
