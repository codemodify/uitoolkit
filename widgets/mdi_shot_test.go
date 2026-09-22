package widgets_test

import (
	"fmt"
	"testing"

	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// mdiSample is an area of four document windows: one minimised, three
// cascaded, the last active.
func mdiSample() (*widgets.MDIArea, []*widgets.MDIWindow) {
	a := widgets.NewMDIArea()
	var wins []*widgets.MDIWindow
	for i, name := range []string{"Budget.txt", "Notes.txt", "Letter.txt", "Todo.txt"} {
		body := widgets.NewTextArea(fmt.Sprintf("%s\n\nDocument %d of four.", name, i+1), "", nil)
		body.SetAccessibleName(name)
		wins = append(wins, a.AddWindow(name, widgets.NewPad(6, body)))
	}
	return a, wins
}

func TestMDIShots(t *testing.T) {
	for _, pack := range breadthLooks {
		for _, scale := range []float32{1, 1.75} {
			if testing.Short() && scale != 1 {
				continue
			}
			name := fmt.Sprintf("mdi-%s@%g", pack, scale)
			t.Run(name, func(t *testing.T) {
				var area *widgets.MDIArea
				var wins []*widgets.MDIWindow
				a, win := shotWindow(t, pack, scale, 640, 460, func() widget.Component {
					area, wins = mdiSample()
					return area
				})
				defer win.Close()
				wins[0].Minimize()
				area.Cascade()
				area.Activate(wins[2])
				a.PumpOnce()
				a.PumpOnce()
				writeShot(t, win, name)
				// Maximised, and the tabbed view.
				wins[2].Maximize()
				a.PumpOnce()
				writeShot(t, win, name+"-max")
				area.SetViewMode(widgets.MDITabbed)
				a.PumpOnce()
				a.PumpOnce()
				writeShot(t, win, name+"-tabs")
			})
		}
	}
}
