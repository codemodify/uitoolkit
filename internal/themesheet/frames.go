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
		if shot, org := FrameShot(a, fc, scale); shot != nil {
			// The shot carries the shadow's margin around the window: land
			// the window where the column is, shadow and all.
			ctx.DrawImage(shot, s.u(x)-org.X, s.u(y+22)-org.Y)
		}
	}
	return img
}

// OverviewCases are the states the frames overview shows for every pack.
var OverviewCases = []FrameCase{FrameCases[0], FrameCases[1], FrameCases[2], FrameCases[3]}

// overview window size at 1x, and the name column's width. The windows hold
// one line of content: room for it under the tallest caption (GNOME's).
const (
	overviewW    = 270
	overviewH    = 92
	overviewName = 150
)

// RenderFramesOverview paints the frames of several packs, one row each:
// the pack's name, then its frame in OverviewCases (active, backdrop,
// maximized, the close button hot), at display scale scale.
func RenderFramesOverview(packs []style.ThemePack, scale float32) *paintengine2d.Image {
	if scale <= 0 {
		scale = 1
	}
	rowH := float32(overviewH + 10)
	w := float32(overviewName + len(OverviewCases)*(overviewW+10) + 10)
	h := float32(34) + float32(len(packs))*rowH
	img := paintengine2d.NewImage(int(w*scale+0.5), int(h*scale+0.5))
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.RGB(0.4, 0.43, 0.48))
	var head style.LookAndFeel = style.WithScale(style.LightLook(), scale)
	f := head.BoldFont()
	u := func(v float32) float32 { return v * scale }
	for i, fc := range OverviewCases {
		f.Draw(ctx, fc.Name, paintengine2d.Pt(u(float32(overviewName+i*(overviewW+10))), u(10)), paintengine2d.RGB(1, 1, 1))
	}
	for r, p := range packs {
		y := 34 + float32(r)*rowH
		name := p.Name
		if FrameKind(p.Look()) == "adapted" {
			name += " (adapted)"
		}
		f.Draw(ctx, name, paintengine2d.Pt(u(10), u(y+overviewH*0.4)), paintengine2d.RGB(1, 1, 1))
		a := app.New(app.Options{Look: p.Look(), Headless: true, Scale: scale})
		for i, fc := range OverviewCases {
			if shot, org := frameShot(a, fc, scale, overviewW, overviewH, true); shot != nil {
				ctx.DrawImage(shot, u(float32(overviewName+i*(overviewW+10)))-org.X, u(y)-org.Y)
			}
		}
	}
	return img
}

// FrameKind says how lk paints window frames: "native" (its engine's own
// DecorationEngine), "plain" (the base look's hairline frame) or "adapted"
// (its in-app window frame).
func FrameKind(lk style.LookAndFeel) string {
	switch {
	case style.NativeDecoration(lk):
		return "native"
	case style.LookTokens(lk).Engine == "" || style.LookTokens(lk).Engine == "base":
		return "plain"
	}
	return "adapted"
}

// FrameShot paints one frames-sheet window with a's look: frameW×frameH at
// 1x, scaled. The image is the whole surface — the window with the margin
// its shadow lives in — and the point is where the visible window starts
// inside it, so a caller lands the window itself where it means to.
func FrameShot(a *app.Application, fc FrameCase, scale float32) (*paintengine2d.Image, paintengine2d.Point) {
	return frameShot(a, fc, scale, frameW, frameH, false)
}

// frameShot paints a fw×fh (at 1x) window of case fc; a compact one holds a
// label only.
func frameShot(a *app.Application, fc FrameCase, scale float32, fw, fh int, compact bool) (*paintengine2d.Image, paintengine2d.Point) {
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Window", Width: int(float32(fw)*scale + 0.5), Height: int(float32(fh)*scale + 0.5),
		Headless: true, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		return nil, paintengine2d.Point{}
	}
	defer w.Close()
	p := platform.DefaultTitleBarPrefs("")
	if fc.Layout != "" {
		p.Layout = platform.ParseButtonLayout(fc.Layout)
	}
	a.SetTitleBarPrefs(p)
	var body widget.Component = widgets.NewColumn(widgets.NewLabel("Window content"), widgets.NewButton("Button", nil)).WithPad(12)
	if compact {
		body = widgets.NewPad(10, widgets.NewLabel("Window content"))
	}
	w.SetContent(body)
	if fc.Header {
		tabs := widgets.NewBrowserTabs("Documents", "Pictures")
		tabs.OnNew = func() {}
		tabs.MinTabWidth = 60
		w.SetTitleBar(widgets.NewHeaderBar(nil, tabs, nil))
		body = widgets.NewColumn(widgets.NewToolBar(widgets.ToolText("Back", nil), widgets.ToolText("Forward", nil)),
			widgets.NewPad(12, widgets.NewLabel("Window content")))
		w.SetContent(body)
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
	w.CaptureSurface()
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
	return w.CaptureSurface(), w.WindowRect().Min
}
