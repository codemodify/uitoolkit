package main

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	uitoolkit "github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// BenchmarkGalleryRepaint repaints the whole gallery once per iteration in
// each engine (headless: the CPU rasterizer), so a slow or allocation-heavy
// engine shows up next to the stock look.
func BenchmarkGalleryRepaint(b *testing.B) {
	for _, name := range []string{"dark", "light", "win95", "luna", "aqua", "brushed-metal", "platinum", "motif", "cde", "next", "wmaker-default"} {
		b.Run(name, func(b *testing.B) {
			p, ok := style.LoadTheme(name)
			if !ok {
				b.Skipf("no pack %q", name)
			}
			a := uitoolkit.New(uitoolkit.Options{Look: p.Look(), Headless: true})
			w, err := a.NewWindow(platform.WindowOptions{Width: 1000, Height: 760, Headless: true})
			if err != nil {
				b.Fatal(err)
			}
			w.SetContent(buildGallery(a, w, p.Palette == style.ThemeLight))
			a.PumpOnce()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				w.Invalidate(nil, paintengine2d.Rect{})
				a.PumpOnce()
			}
		})
	}
}

// BenchmarkGallerySmallRepaint is the steady state: one button repaints
// (a hover) in an otherwise unchanged gallery.
func BenchmarkGallerySmallRepaint(b *testing.B) {
	for _, name := range []string{"dark", "win95", "luna", "aqua", "brushed-metal", "next"} {
		b.Run(name, func(b *testing.B) {
			p, ok := style.LoadTheme(name)
			if !ok {
				b.Skipf("no pack %q", name)
			}
			a := uitoolkit.New(uitoolkit.Options{Look: p.Look(), Headless: true})
			w, err := a.NewWindow(platform.WindowOptions{Width: 1000, Height: 760, Headless: true})
			if err != nil {
				b.Fatal(err)
			}
			w.SetContent(buildGallery(a, w, p.Palette == style.ThemeLight))
			a.PumpOnce()
			var btn widget.Component
			widget.Walk(w.Content(), func(c widget.Component) {
				if bt, ok := c.(*widgets.Button); ok && btn == nil {
					btn = bt
				}
			})
			if btn == nil {
				b.Fatal("no button")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				btn.Invalidate()
				a.PumpOnce()
			}
		})
	}
}
