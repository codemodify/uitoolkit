package app

import (
	"fmt"
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// benchDamageWindow is a realistically busy 1080p-ish window.
func benchDamageWindow(b *testing.B) (*Application, *Window, *widgets.Button, *widgets.TextField) {
	b.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 1280, Height: 800, Headless: true})
	if err != nil {
		b.Fatal(err)
	}
	btn := widgets.NewButton("Send", nil)
	tf := widgets.NewTextField("subject", "", nil)
	side := widgets.NewColumn()
	for i := 0; i < 12; i++ {
		side.Add(widgets.NewButton(fmt.Sprintf("folder %d", i), nil))
	}
	table := widgets.NewTableView(
		[]widgets.TableColumn{{Title: "From", Width: 200}, {Title: "Subject", Width: 420}, {Title: "Date", Width: 140}},
		500,
		func(r, c int) string { return fmt.Sprintf("cell %d/%d", r, c) }, nil)
	w.SetContent(widgets.NewRow(side, widgets.NewColumn(widgets.NewRow(btn, tf), table)))
	a.PumpOnce()
	return a, w, btn, tf
}

func benchSmallInvalidate(b *testing.B, fullFrame bool) {
	fullFrameOnce.once.Do(func() {})
	saved := fullFrameOnce.on
	fullFrameOnce.on = fullFrame
	defer func() { fullFrameOnce.on = saved }()

	a, w, btn, _ := benchDamageWindow(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		btn.Invalidate()
		a.PumpOnce()
	}
	b.StopTimer()
	_ = w
}

// One hovered button on a 1280x800 window.
func BenchmarkSmallInvalidatePartial(b *testing.B) { benchSmallInvalidate(b, false) }
func BenchmarkSmallInvalidateFull(b *testing.B)    { benchSmallInvalidate(b, true) }

func benchCaretBlink(b *testing.B, fullFrame bool) {
	fullFrameOnce.once.Do(func() {})
	saved := fullFrameOnce.on
	fullFrameOnce.on = fullFrame
	defer func() { fullFrameOnce.on = saved }()

	a, w, _, tf := benchDamageWindow(b)
	w.RequestFocus(tf)
	a.PumpOnce()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.toggleBlink()
		a.PumpOnce()
	}
}

// A caret blink is the smallest possible repaint and the one that happens
// twice a second forever.
func BenchmarkCaretBlinkPartial(b *testing.B) { benchCaretBlink(b, false) }
func BenchmarkCaretBlinkFull(b *testing.B)    { benchCaretBlink(b, true) }
