// uitk-smoke opens one window on whatever backend this machine has and
// reports what it saw, as text.
//
// It exists because "a window appeared" is not evidence. It prints the
// backend, its capabilities, the scale, the window's position and every
// event it received, so a run on a machine you are not sitting at comes
// back as something you can read on the machine you are sitting at.
//
// It paints four quadrants in known colours — red, green, blue, white —
// because the failure a new backend is most likely to have is in the
// byte order of its present. On Windows a 32-bit DIB is BGRX where
// paintengine2d keeps RGBA, so red and blue trade places; a photograph
// of the window is enough to see it.
//
//	uitk-smoke                 open a window for five seconds and report
//	uitk-smoke -hold 20s       keep it up longer
//	uitk-smoke -quiet          capabilities only, no window
//	uitk-smoke -watchdog 15s   panic after this long, to dump the stacks
//	                           of a window that will not open
//	uitk-smoke -exercise       ask the frame and geometry seams to do
//	                           every thing they claim they can, and
//	                           report what actually happened
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

func main() {
	hold := flag.Duration("hold", 5*time.Second, "how long to keep the window up")
	quiet := flag.Bool("quiet", false, "do not open a window")
	watchdog := flag.Duration("watchdog", 0, "panic after this long, to dump the stacks of a window that will not open")
	exercise := flag.Bool("exercise", false, "call every capability the frame and geometry seams claim")
	flag.Parse()

	if *watchdog > 0 {
		// Run this with GOTRACEBACK=all: a window can wedge inside a
		// synchronous platform call, and the stack of the thread that
		// is stuck is the only thing that says where.
		go func() {
			time.Sleep(*watchdog)
			panic("uitk-smoke: watchdog expired")
		}()
	}

	b := platform.Default(false)
	fmt.Printf("backend      %s\n", b.Name())
	fmt.Printf("backend caps %s\n", b.Caps())
	if b.Name() == "offscreen" {
		fmt.Println("NOT a native backend — there is no window system here")
		os.Exit(1)
	}
	if *quiet {
		return
	}

	surf, err := b.NewSurface(platform.WindowOptions{
		Title: "uitoolkit — smoke test", Width: 640, Height: 400,
	})
	if err != nil {
		fmt.Println("NewSurface failed:", err)
		os.Exit(1)
	}
	defer surf.Close()

	w, h := surf.Size()
	fmt.Printf("window       %dx%d device pixels at scale %g\n", w, h, surf.Scale())
	if img := surf.Buffer(); img != nil {
		fmt.Printf("buffer       %dx%d, stride %d, %d bytes\n", img.Width, img.Height, img.Stride, len(img.Pix))
	}
	fmt.Printf("frame caps   %s\n", platform.FrameOf(surf).FrameCaps())
	fmt.Printf("geometry     %s\n", platform.GeometryOf(surf).GeometryCaps())
	if x, y, ok := platform.GeometryOf(surf).Position(); ok {
		fmt.Printf("position     %d,%d\n", x, y)
	} else {
		fmt.Println("position     unknown")
	}

	if *exercise {
		exerciseSeams(surf)
	}

	paint := func() {
		img := surf.Buffer()
		if img == nil {
			return
		}
		ctx := paintengine2d.NewContext(img)
		w, h := float32(img.Width), float32(img.Height)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w/2, h/2), paintengine2d.Fill(paintengine2d.RGB(0.85, 0.15, 0.15)))
		ctx.DrawRect(paintengine2d.XYWH(w/2, 0, w/2, h/2), paintengine2d.Fill(paintengine2d.RGB(0.15, 0.7, 0.2)))
		ctx.DrawRect(paintengine2d.XYWH(0, h/2, w/2, h/2), paintengine2d.Fill(paintengine2d.RGB(0.15, 0.3, 0.9)))
		ctx.DrawRect(paintengine2d.XYWH(w/2, h/2, w/2, h/2), paintengine2d.Fill(paintengine2d.RGB(0.95, 0.95, 0.95)))
		if err := surf.Present(nil); err != nil {
			fmt.Println("present      FAILED:", err)
		}
	}
	paint()
	fmt.Println("painted      top-left RED, top-right GREEN, bottom-left BLUE, bottom-right WHITE")
	fmt.Println("             (red and blue swapped means the present's byte order is wrong)")

	seen := map[platform.EventKind]int{}
	deadline := time.Now().Add(*hold)
	for time.Now().Before(deadline) {
		for _, ev := range surf.Poll() {
			seen[ev.Kind]++
			switch ev.Kind {
			case platform.EventResize:
				fmt.Printf("resize       %dx%d logical\n", ev.Width, ev.Height)
				paint()
			case platform.EventScale:
				fmt.Printf("scale        now %g\n", surf.Scale())
				paint()
			case platform.EventExpose:
				paint()
			case platform.EventClose:
				fmt.Println("close        the window system asked")
				deadline = time.Now()
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println("events seen:")
	for k, n := range seen {
		fmt.Printf("  %-3d %v\n", n, k)
	}
	fmt.Println("done")
}

// exerciseSeams calls every frame and geometry capability the surface
// claims, and reports what came back.
//
// The point is the gap between the claim and the fact. A backend says
// it can minimize a window; this asks it to, and asks again what state
// the window is in. A capability that answers true and changes nothing
// is the failure worth catching, and it is invisible from the caps
// word alone.
func exerciseSeams(surf platform.Surface) {
	fr, geo := platform.FrameOf(surf), platform.GeometryOf(surf)
	step := func(name string, ok bool, note string) {
		mark := "no"
		if ok {
			mark = "yes"
		}
		fmt.Printf("  %-16s %-4s %s\n", name, mark, note)
	}
	pump := func() {
		for range 20 {
			surf.Poll()
			time.Sleep(10 * time.Millisecond)
		}
	}

	fmt.Println("exercise     geometry")
	if x, y, ok := geo.Position(); ok {
		moved := geo.Move(x+40, y+30)
		pump()
		nx, ny, _ := geo.Position()
		step("Move", moved, fmt.Sprintf("%d,%d -> %d,%d (asked %d,%d)", x, y, nx, ny, x+40, y+30))
		geo.Move(x, y)
		pump()
	}
	step("Raise", geo.Raise(), "")
	step("Visible", geo.Visible(), "")
	before := geo.SizeLimits()
	step("SetSizing fixed", geo.SetSizing(platform.SizingFixed),
		fmt.Sprintf("limits %+v -> %+v", before, geo.SizeLimits()))
	step("caps when fixed", true, fr.FrameCaps().String())
	geo.SetSizing(platform.SizingResizable)

	fmt.Println("exercise     frame")
	st := func() string { return fmt.Sprintf("%+v", fr.WindowState()) }
	fmt.Printf("  %-16s      %s\n", "state", st())
	step("SetMaximized", fr.SetMaximized(true), "")
	pump()
	fmt.Printf("  %-16s      %s\n", "state", st())
	fr.SetMaximized(false)
	pump()
	step("SetKeepAbove", fr.SetKeepAbove(true), "")
	fr.SetKeepAbove(false)
	step("Lower", fr.Lower(), "")
	geo.Raise()
	// The ones expected to refuse, so a backend that starts claiming
	// them has to say so here too.
	step("StartResize", fr.StartResize(platform.EdgeBottom|platform.EdgeRight), "refused on macOS: no API")
	step("ShowMenu", fr.ShowMenu(paintengine2d.Pt(10, 10)), "refused on macOS: no window menu")
	step("SetShadedHeight", fr.SetShadedHeight(30), "refused on macOS: no window shade")
	step("SetPalette", fr.SetPalette("/tmp/x.colors"), "refused on macOS: KDE colour schemes")
	step("MaximizeAxis", fr.MaximizeAxis(true), "refused on macOS: zoom is both ways")
	icon := paintengine2d.NewImage(64, 64)
	c := paintengine2d.NewContext(icon)
	c.DrawRect(paintengine2d.XYWH(0, 0, 64, 64), paintengine2d.Fill(paintengine2d.RGB(0.2, 0.5, 0.9)))
	step("SetIcon", fr.SetIcon([]*paintengine2d.Image{icon}), "the Dock tile on macOS")
	step("RequestDecorations", true, "client, then back")
	fr.RequestDecorations(platform.DecorationsClient)
	pump()
	fmt.Printf("  %-16s      %s\n", "decorations", fr.Decorations())
	fr.RequestDecorations(platform.DecorationsServer)
	pump()
	fmt.Printf("  %-16s      %s\n", "decorations", fr.Decorations())
}
