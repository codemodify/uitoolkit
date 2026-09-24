package demo

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// roomyControls is the control height past which a pack's Controls tab
// is taller than Settings' preview at the default split: Material's,
// shadcn's and Geist's 40- to 48-pixel touch targets. Those packs clip
// the tab's last row in the preview; every other pack shows all of it.
const roomyControls = 40

// The preview's Controls tab shows every row — the last radio included —
// at Settings' default size and split, in every pack with desktop-sized
// controls, and nothing in it runs off the tab's right edge either.
func TestSettingsPreviewShowsEveryRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := style.SaveAppearance(style.DefaultAppearance()); err != nil {
		t.Fatal(err)
	}
	checked, roomy := 0, 0
	for _, pack := range style.ListThemes() {
		a := uitoolkit.New(uitoolkit.Options{Look: style.LightLook(), Headless: true, Scale: 1, DisableLookWatch: true})
		w, err := a.NewWindow(platform.WindowOptions{Title: "Settings", Width: 1024, Height: 860, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		w.SetContent(SettingsAppStaged(a, w, pack.Name))
		a.PumpOnce()
		scope := previewScope(t, w)
		if scope.Theme().Metrics().ControlH >= roomyControls {
			roomy++
			w.Close()
			continue
		}
		checked++
		var tabs *widgets.TabView
		widget.Walk(scope, func(c widget.Component) {
			if tv, ok := c.(*widgets.TabView); ok {
				tabs = tv
			}
		})
		if tabs == nil {
			t.Fatalf("%s: no tab view in the preview", pack.Name)
		}
		box := deviceRect(tabs)
		n := 0
		widget.Walk(tabs, func(c widget.Component) {
			switch c.(type) {
			case *widgets.Button, *widgets.Checkbox, *widgets.RadioButton, *widgets.TextField,
				*widgets.ComboBox, *widgets.NumberField, *widgets.Switch, *widgets.Slider, *widgets.ProgressBar:
			default:
				return
			}
			if !c.Visible() || c.LocalBounds().Empty() {
				return // a control of a tab not showing
			}
			n++
			if r := deviceRect(c); r.Max.Y > box.Max.Y+0.5 || r.Max.X > box.Max.X+0.5 {
				t.Errorf("%s: %T %q at %v runs out of the tab view %v", pack.Name, c, widgetText(c), r, box)
			}
		})
		if n < 12 {
			t.Errorf("%s: only %d controls on the Controls tab", pack.Name, n)
		}
		w.Close()
	}
	t.Logf("%d packs checked, %d with roomy controls left out", checked, roomy)
	if checked < 100 {
		t.Errorf("only %d packs have desktop-sized controls (%d roomy)", checked, roomy)
	}
}

func deviceRect(c widget.Component) paintengine2d.Rect {
	o, b := widget.DeviceOrigin(c), c.LocalBounds()
	return paintengine2d.XYWH(o.X, o.Y, b.Dx(), b.Dy())
}

func widgetText(c widget.Component) string {
	switch v := c.(type) {
	case *widgets.Button:
		return v.Text
	case *widgets.Checkbox:
		return v.Text
	case *widgets.RadioButton:
		return v.Text
	}
	return ""
}
