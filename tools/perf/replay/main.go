// replay records one window's scene and replays it on the CPU device and
// on the GPU device, printing its op count and the median frame of each.
//
//	go run ./tools/perf/replay list-hover
//	go run ./tools/perf/replay settings 61
//
// It is the answer to "is one fat draw op better than a hundred thin
// ones", which is not the same answer on the two devices: the GPU pays
// per draw call and the CPU rasterizer pays per pixel and per overlap
// test. paintengine2d's GPU device renders surfacelessly, so this needs a
// DRM render node but no compositor and no display — run it through
// tools/testenv.sh like everything else.
package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/cmd/uitoolkit-settings/settingsapp"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/showcase"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// scene is one of the windows this can record, and what to do to it
// before the recording is taken.
type scene struct {
	name  string
	w, h  int
	build func(*app.Application, *app.Window) widget.Component
	// hover moves the pointer into the content before recording, which is
	// what puts a focus ring and a hovered row in the scene.
	hover bool
}

var scenes = []scene{
	{name: "list", w: 280, h: 220, build: listView},
	{name: "list-hover", w: 280, h: 220, build: listView, hover: true},
	{name: "showcase", w: 1100, h: 720, build: func(a *app.Application, win *app.Window) widget.Component {
		return showcase.App(a, win, false)
	}},
	{name: "settings", w: 1280, h: 800, build: settingsapp.SettingsApp},
}

func listView(*app.Application, *app.Window) widget.Component {
	return widgets.NewListView(80, func(i int) string { return fmt.Sprintf("row %02d body", i) }, nil)
}

func median(d []time.Duration) time.Duration {
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[len(d)/2]
}

// record opens s headless and returns the scene of one full repaint.
func record(s scene) *paintengine2d.Scene {
	a := app.New(app.Options{Look: style.DarkLook(), Headless: true})
	win, err := a.NewWindow(platform.WindowOptions{Width: s.w, Height: s.h, Headless: true})
	if err != nil {
		panic(err)
	}
	win.SetContent(s.build(a, win))
	a.PumpOnce()
	if s.hover {
		o := widget.DeviceOrigin(win.Content())
		win.Inject(platform.Event{Kind: platform.EventMouseMove, Pos: paintengine2d.Pt(o.X+40, o.Y+40)})
	} else {
		win.Invalidate(nil, paintengine2d.Rect{})
	}
	a.PumpOnce()
	return win.Scene()
}

// bench replays sc on dev reps times, 20 replays a sample.
func bench(sc *paintengine2d.Scene, dev paintengine2d.Device, reps int, frame func()) time.Duration {
	var out []time.Duration
	for r := 0; r < reps; r++ {
		frame()
		st := time.Now()
		for i := 0; i < 20; i++ {
			paintengine2d.DrawScene(sc, dev)
		}
		out = append(out, time.Since(st)/20)
	}
	return median(out)
}

// walk counts the groups and the draw ops a finished scene replays.
// Scene.Nodes is what the last recording had to build; this is what the
// replay costs whether it was rebuilt or attached from the cache.
func walk(g *paintengine2d.GroupNode) (groups, ops int) {
	if g == nil {
		return 0, 0
	}
	groups = 1
	for _, ch := range g.Children {
		if sub, ok := ch.(*paintengine2d.GroupNode); ok {
			sg, so := walk(sub)
			groups += sg
			ops += so
			continue
		}
		ops++
	}
	return
}

func main() {
	want := ""
	if len(os.Args) > 1 {
		want = os.Args[1]
	}
	reps := 41
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
			reps = n
		}
	}
	for _, s := range scenes {
		if want != "" && want != "all" && want != s.name {
			continue
		}
		sc := record(s)
		if sc == nil {
			fmt.Printf("%-12s no scene (is UITK_SCENE off?)\n", s.name)
			continue
		}
		img := paintengine2d.NewImage(s.w, s.h)
		cpu := bench(sc, paintengine2d.NewCPUDevice(img), reps, func() {})
		groups, ops := walk(sc.Root)
		line := fmt.Sprintf("%-12s %4dx%-4d ops=%-5d groups=%-4d re-recorded=%-5d cpu %8.1f us",
			s.name, s.w, s.h, ops, groups, sc.Nodes, float64(cpu.Nanoseconds())/1000)
		if g, err := paintengine2d.NewGPUDevice(s.w, s.h); err == nil {
			gpu := bench(sc, g, reps, func() { _ = g.EndFrame(); _ = g.BeginFrame() })
			_ = g.EndFrame()
			_ = g.Close()
			line += fmt.Sprintf("   gpu %8.1f us", float64(gpu.Nanoseconds())/1000)
		} else {
			line += fmt.Sprintf("   gpu unavailable (%v)", err)
		}
		fmt.Println(line)
	}
}
