//go:build windows

// Command uitk-winsmoke is the Win32 backend's first-run report.
//
//	uitk-winsmoke.exe            open a window for 5 seconds and report
//	uitk-winsmoke.exe -hold 20   keep it up longer
//	uitk-winsmoke.exe -quiet     no window: capabilities only
//
// It exists because "a window appeared" is not evidence. The backend
// makes claims — which capabilities Windows grants, what scale the
// monitor reports, that resizing and closing arrive as events — and this
// prints what actually happened so a run in a VM comes back as text that
// can be read here.
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
		// Run this with GOTRACEBACK=all: a Win32 window can wedge inside
		// a synchronous message, and the stack of the thread that is
		// stuck is the only thing that says where.
		go func() {
			time.Sleep(*watchdog)
			panic("uitk-winsmoke: watchdog expired")
		}()
	}

	b := platform.Default(false)
	fmt.Printf("backend      %s\n", b.Name())
	fmt.Printf("backend caps %s\n", b.Caps())
	if b.Name() != "win32" {
		fmt.Println("NOT the Win32 backend — nothing below is about Windows")
		os.Exit(1)
	}
	if *quiet {
		return
	}

	surf, err := b.NewSurface(platform.WindowOptions{
		Title: "uitoolkit — Win32 smoke test", Width: 640, Height: 400,
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
		fmt.Println("position     unknown — which on Windows is a bug")
	}

	// Something to look at, and something to prove the blit works: four
	// quadrants in known colours, so a wrong swizzle is obvious (red and
	// blue trading places is the failure this backend is most likely to
	// have).
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
	fmt.Println("             (red and blue swapped means the DIB swizzle is wrong)")

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
				fmt.Println("close        the window manager asked")
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
