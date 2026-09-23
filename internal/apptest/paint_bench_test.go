package apptest

import (
	"fmt"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/examples/mail/mailapp"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func benchWindow(b *testing.B, w, h int, content func(*app.Application, *app.Window) widget.Component) (*app.Application, *app.Window) {
	b.Helper()
	a := app.New(app.Options{Look: style.DarkLook(), Headless: true})
	win, err := a.NewWindow(platform.WindowOptions{Width: w, Height: h, Headless: true})
	if err != nil {
		b.Fatal(err)
	}
	win.SetContent(content(a, win))
	a.PumpOnce()
	return a, win
}

func BenchmarkMailFullPaint(b *testing.B) {
	mailapp.IsolateTestEnvTB(b)
	a, w := benchWindow(b, 1280, 800, mailapp.MailApp)
	b.ReportAllocs()
	b.ResetTimer()
	reused := 0
	for i := 0; i < b.N; i++ {
		w.Invalidate(nil, paintengine2d.Rect{})
		a.PumpOnce()
		if sc := w.Scene(); sc != nil {
			reused += sc.Reused
		}
	}
	b.ReportMetric(float64(reused)/float64(b.N), "reused/op")
}

func BenchmarkGalleryFullPaint(b *testing.B) {
	a, w := benchWindow(b, 1100, 720, func(a *app.Application, win *app.Window) widget.Component {
		return demo.Gallery(a, win, false)
	})
	b.ReportAllocs()
	b.ResetTimer()
	reused := 0
	for i := 0; i < b.N; i++ {
		w.Invalidate(nil, paintengine2d.Rect{})
		a.PumpOnce()
		if sc := w.Scene(); sc != nil {
			reused += sc.Reused
		}
	}
	b.ReportMetric(float64(reused)/float64(b.N), "reused/op")
}

func BenchmarkListHover(b *testing.B) {
	a, w := benchWindow(b, 280, 220, func(*app.Application, *app.Window) widget.Component {
		return widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d body", i) }, nil)
	})
	o := widget.DeviceOrigin(w.Content())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		y := o.Y + 12 + float32(i%8)*28
		w.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+40, y)})
		a.PumpOnce()
	}
}

func BenchmarkListScroll(b *testing.B) {
	a, w := benchWindow(b, 400, 360, func(*app.Application, *app.Window) widget.Component {
		return widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d body", i) }, nil)
	})
	list := w.Content().(*widgets.ListView)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			list.ScrollTo(40)
		} else {
			list.ScrollTo(0)
		}
		a.PumpOnce()
	}
}
