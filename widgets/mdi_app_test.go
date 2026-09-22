package widgets_test

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Through a real window's event path: a press on a child's caption, moves
// past the drag threshold and a release move the child.
func TestMDIDragThroughTheWindow(t *testing.T) {
	var wins []*widgets.MDIWindow
	a, win := shotWindow(t, "", 1, 960, 640, func() widget.Component {
		area := widgets.NewMDIArea()
		for _, name := range []string{"One", "Two", "Three", "Four"} {
			ed := widgets.NewRichText("Type here")
			ed.SetAccessibleName(name)
			body := widgets.NewColumn(widgets.NewRichTextBar(ed), ed).WithGap(4).WithPad(4)
			body.AddFlex(ed, 1)
			w := area.AddWindow(name, body)
			w.SetInitialFocus(ed)
			wins = append(wins, w)
		}
		menu := widgets.NewMenuBar(widgets.NewMenu("&File"), area.WindowMenu())
		root := widgets.NewColumn(menu, area)
		root.AddFlex(area, 1)
		return root
	})
	defer win.Close()
	w := wins[3]
	before := w.Geometry()
	o := widget.DeviceOrigin(w)
	p := o.Add(paintengine2d.Pt(before.Dx()*0.4, 14))
	win.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: p})
	win.Inject(platform.Event{Kind: platform.EventMouseDown, Pos: p, Button: platform.ButtonLeft})
	for i := 1; i <= 10; i++ {
		win.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: p.Add(paintengine2d.Pt(float32(i*10), float32(i*5))), Button: platform.ButtonLeft})
	}
	win.Inject(platform.Event{Kind: platform.EventMouseUp, Pos: p.Add(paintengine2d.Pt(100, 50)), Button: platform.ButtonLeft})
	a.PumpOnce()
	after := w.Geometry()
	if d := after.Min.Sub(before.Min); d.X != 100 || d.Y != 50 {
		t.Fatalf("moved by %v (from %v to %v)", d, before, after)
	}
}
