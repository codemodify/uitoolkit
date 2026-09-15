package themesheet

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The window-frames sheet: real offscreen windows whose frame uitoolkit
// draws in a look, in the states a frame has to get right. Engine authors
// check their DecorationEngine (or the adapter over their in-app window
// frame) with it: go run ./cmd/uitk-themesheet -frames -theme luna.

// FramesWidth and FramesHeight are the frames sheet's size at 1x.
const (
	FramesWidth  = 1180
	FramesHeight = 560
)

// frame window size at 1x.
const (
	frameW = 370
	frameH = 150
)

// FrameCase is one window of the frames sheet.
type FrameCase struct {
	Name string
	// Header gives the window the app's own title bar (tabs and a tool
	// bar); otherwise the frame's default caption shows the window title.
	Header bool
	State  platform.WindowState
	// Hover / Press put the pointer on / press this caption button.
	Hover, Press platform.CaptionButton
	// Layout overrides the desktop's button layout (GNOME syntax).
	Layout string
}

// FrameCases are the windows the sheet shows, in order.
var FrameCases = []FrameCase{
	{Name: "active", State: platform.WindowState{Activated: true}},
	{Name: "backdrop"},
	{Name: "maximized", State: platform.WindowState{Activated: true, Maximized: true}},
	{Name: "close hot", State: platform.WindowState{Activated: true}, Hover: platform.CaptionClose},
	{Name: "close pressed", State: platform.WindowState{Activated: true}, Press: platform.CaptionClose},
	{Name: "maximize hot", State: platform.WindowState{Activated: true}, Hover: platform.CaptionMaximize},
	{Name: "title bar with tabs", Header: true, State: platform.WindowState{Activated: true}},
	{Name: "tabs, backdrop", Header: true},
	{Name: "KDE layout: menu left, close hot", State: platform.WindowState{Activated: true}, Hover: platform.CaptionClose, Layout: "icon:minimize,maximize,close"},
}

// RenderFrames paints the frames sheet of look (a 1x look) at display
// scale scale.
func RenderFrames(look style.LookAndFeel, scale float32, title string) *paintengine2d.Image {
	if scale <= 0 {
		scale = 1
	}
	lk := style.WithScale(look, scale)
	img := paintengine2d.NewImage(int(FramesWidth*scale+0.5), int(FramesHeight*scale+0.5))
	ctx := paintengine2d.NewContext(img)
	s := &sheet{ctx: ctx, lk: lk, sc: scale, p: lk.Palette()}
	// A neutral desk, so the frame's own edge shows.
	desk := paintengine2d.RGB(0.44, 0.47, 0.52)
	if style.Luma(s.p.Background) < 0.4 {
		desk = paintengine2d.RGB(0.62, 0.65, 0.70)
	}
	ctx.Clear(desk)
	f := lk.BoldFont()
	if f == nil {
		f = lk.Font()
	}
	label := func(x, y float32, text string) {
		if f != nil {
			f.Draw(ctx, text, paintengine2d.Pt(s.u(x), s.u(y)), paintengine2d.RGB(1, 1, 1))
		}
	}
	label(16, 10, title)
	a := app.New(app.Options{Look: look, Headless: true, Scale: scale})
	for i, fc := range FrameCases {
		col, row := i%3, i/3
		x, y := float32(16+col*(frameW+18)), float32(40+row*(frameH+36))
		label(x, y, fc.Name)
		if shot := FrameShot(a, fc, scale); shot != nil {
			ctx.DrawImage(shot, s.u(x), s.u(y+22))
		}
	}
	return img
}

// FrameShot paints one frames-sheet window with a's look: frameW×frameH at
// 1x, scaled.
func FrameShot(a *app.Application, fc FrameCase, scale float32) *paintengine2d.Image {
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Window", Width: int(frameW*scale + 0.5), Height: int(frameH*scale + 0.5),
		Headless: true, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		return nil
	}
	defer w.Close()
	p := platform.DefaultTitleBarPrefs("")
	if fc.Layout != "" {
		p.Layout = platform.ParseButtonLayout(fc.Layout)
	}
	a.SetTitleBarPrefs(p)
	body := widgets.NewColumn(widgets.NewLabel("Window content"), widgets.NewButton("Button", nil)).WithPad(12)
	w.SetContent(body)
	if fc.Header {
		tabs := widgets.NewTabBar("Documents", "Pictures")
		tools := widgets.NewToolBar(widgets.ToolIconBtn(style.IconSearch, "", nil))
		w.SetTitleBar(widgets.NewHeaderBar(nil, tabs, []widget.Component{tools}))
	}
	o := w.Surface().(*platform.Offscreen)
	if !fc.State.Activated {
		// The desktop says the window is active first: a window it never
		// said anything about paints as active.
		o.SimulateWindowState(platform.WindowState{Activated: true})
		a.PumpOnce()
	}
	o.SimulateWindowState(fc.State)
	a.PumpOnce()
	w.Capture()
	point := func(b platform.CaptionButton) (paintengine2d.Point, bool) {
		hb := w.Caption()
		if hb == nil {
			return paintengine2d.Point{}, false
		}
		left, right := hb.Controls()
		for _, c := range []*widgets.WindowControls{left, right} {
			if r := c.ButtonRect(b); !r.Empty() {
				o := widget.DeviceOrigin(c)
				return paintengine2d.Pt(o.X+(r.Min.X+r.Max.X)*0.5, o.Y+(r.Min.Y+r.Max.Y)*0.5), true
			}
		}
		return paintengine2d.Point{}, false
	}
	if pt, ok := point(fc.Hover); ok && fc.Hover != platform.CaptionNone {
		w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: pt})
	}
	if pt, ok := point(fc.Press); ok && fc.Press != platform.CaptionNone {
		w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: pt})
		w.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: pt, Button: platform.ButtonLeft})
	}
	a.PumpOnce()
	return w.Capture()
}
