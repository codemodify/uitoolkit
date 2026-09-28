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
