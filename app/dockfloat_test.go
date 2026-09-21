package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
)

// A floating panel's window is placed and sized in logical pixels, the
// unit the dock states its geometry in, so what the dock asked for is what
// the window reports back at any display scale — which is what a saved
// layout brings back.
func TestDockWindowGeometryIsLogical(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, scale := range []float32{1, 1.75, 2} {
		a := New(Options{Look: style.DarkLook(), Headless: true, Scale: scale, DisableLookWatch: true})
		want := paintengine2d.XYWH(30, 40, 300, 200)
		fw, err := DockWindows(a).OpenFloat("Panel", want)
		if err != nil {
			t.Fatal(err)
		}
		dw := fw.(*dockWindow)
		if w, h := dw.win.Size(); w != 300 || h != 200 {
			t.Errorf("%.2fx: the window is %dx%d, asked for 300x200", scale, w, h)
		}
		if x, y, ok := dw.win.Position(); !ok || x != 30 || y != 40 {
			t.Errorf("%.2fx: the window is at %d,%d (ok=%v), asked for 30,40", scale, x, y, ok)
		}
		if got := fw.Geometry(); got != want {
			t.Errorf("%.2fx: the window reports %v, asked for %v", scale, got, want)
		}
		fw.Close()
	}
}
