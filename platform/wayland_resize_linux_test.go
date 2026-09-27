//go:build linux && cgo

package platform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// configureToplevel is a compositor resizing the window: xdg_toplevel.configure
// with the new logical size and no states, made current by xdg_surface.configure.
func (f *wlFake) configureToplevel(s *wlSurface, w, h int, serial uint32) {
	f.mu.Lock()
	top, xdg := f.ids["xdg_toplevel"], f.ids["xdg_surface"]
	f.mu.Unlock()
	if len(top) == 0 || len(xdg) == 0 {
		panic("no toplevel on the fake compositor")
	}
	f.emit(top[0], 0, uint32(w), uint32(h), uint32(0))
	f.emit(xdg[0], 0, serial)
	// Dispatch until the ack is on the wire rather than assuming one
	// pass carries it: the fake writes its events from a goroutine of
	// its own, so on a loaded machine the first roundtrip can return
	// before both are readable. Bounded, so a real failure to ack still
	// fails the test that called this rather than hanging it.
	want := fmt.Sprintf("ack_configure %d", serial)
	for i := 0; i < 50; i++ {
		s.conn.roundtrip()
		_ = s.Poll()
		if inOrder(f.requests(), want) {
			return
		}
	}
}

// lastBuffer is the geometry of the last wl_shm buffer the client made.
func lastBuffer(log []string) (w, h, stride int, ok bool) {
	for i := len(log) - 1; i >= 0; i-- {
		if !strings.HasPrefix(log[i], "wl_buffer ") || strings.HasPrefix(log[i], "wl_buffer.destroy") {
			continue
		}
		var id, format int
		if _, err := fmt.Sscanf(log[i], "wl_buffer %d %dx%d stride %d format %d", &id, &w, &h, &stride, &format); err == nil {
			return w, h, stride, true
		}
	}
	return 0, 0, 0, false
}

// damageBoxes is every wl_surface.damage_buffer box since the marker index.
func damageBoxes(log []string, from int) [][4]int {
	var out [][4]int
	for _, l := range log[min(from, len(log)):] {
		var b [4]int
		if _, err := fmt.Sscanf(l, "damage_buffer %d %d %d %d", &b[0], &b[1], &b[2], &b[3]); err == nil {
			out = append(out, b)
		}
	}
	return out
}

func paintWhole(s *wlSurface) {
	ctx := NewPaintContext(s)
	if ctx == nil {
		return
	}
	ctx.Clear(paintengine2d.RGB(0.2, 0.4, 0.7))
}

// A resize is a configure the client acks. The frame it commits afterwards
// must be a buffer of exactly that size, its stride packed to that width,
// and every damage box inside it: a buffer attached at one size while the
// compositor reads it at another is the shear that made a dragged window
// dissolve into vertical columns.
func TestWaylandResizeCommitsTheAckedSize(t *testing.T) {
	f := startWlFake(t)
	s := fakeSurface(t)
	paintWhole(s)
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	s.conn.roundtrip()

	serial := uint32(100)
	for _, sz := range [][2]int{{320, 200}, {200, 320}, {321, 201}, {97, 43}} {
		serial++
		mark := len(f.requests())
		f.configureToplevel(s, sz[0], sz[1], serial)
		if !inOrder(f.requests()[mark:], fmt.Sprintf("ack_configure %d", serial)) {
			t.Fatalf("configure %dx%d was never acked:\n%s", sz[0], sz[1], strings.Join(f.requests()[mark:], "\n"))
		}
		// The application answers the resize event the configure queued.
		_ = s.Resize(sz[0], sz[1])
		paintWhole(s)
		if err := s.Present(nil); err != nil {
			t.Fatal(err)
		}
		s.conn.roundtrip()

		log := f.requests()
		w, h, stride, ok := lastBuffer(log)
		if !ok {
			t.Fatalf("%dx%d: no buffer was made", sz[0], sz[1])
		}
		wantW, wantH := s.bufferWH()
		if w != wantW || h != wantH {
			t.Fatalf("configure %dx%d: buffer %dx%d, want %dx%d", sz[0], sz[1], w, h, wantW, wantH)
		}
		if stride != w*4 {
			t.Fatalf("configure %dx%d: buffer %dx%d has stride %d, want %d", sz[0], sz[1], w, h, stride, w*4)
		}
		boxes := damageBoxes(log, mark)
		if len(boxes) == 0 {
			t.Fatalf("configure %dx%d: nothing was damaged", sz[0], sz[1])
		}
		for _, b := range boxes {
			if b[0] < 0 || b[1] < 0 || b[0]+b[2] > w || b[1]+b[3] > h {
				t.Fatalf("configure %dx%d: damage %v reaches outside the %dx%d buffer", sz[0], sz[1], b, w, h)
			}
		}
	}
}

// Between the configure and the repaint the application has not drawn at
// the new size yet. Presenting there must not commit the empty buffer the
// resize just allocated: an unpainted target is a black flash on the CPU
// and recycled video memory on the GPU.
func TestWaylandPresentSkipsTheUnpaintedBuffer(t *testing.T) {
	f := startWlFake(t)
	s := fakeSurface(t)
	paintWhole(s)
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	s.conn.roundtrip()

	f.configureToplevel(s, 640, 480, 7)
	_ = s.Resize(640, 480)
	mark := len(f.requests())
	// No paint in between: this present has nothing to show.
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	s.conn.roundtrip()
	for _, l := range f.requests()[mark:] {
		if strings.HasPrefix(l, "attach") || strings.HasPrefix(l, "commit") {
			t.Fatalf("an unpainted buffer was committed: %q", l)
		}
	}
	// Once the frame is painted it goes out.
	paintWhole(s)
	if err := s.Present(nil); err != nil {
		t.Fatal(err)
	}
	s.conn.roundtrip()
	if !inOrder(f.requests()[mark:], "attach", "commit") {
		t.Fatalf("the painted frame was not committed:\n%s", strings.Join(f.requests()[mark:], "\n"))
	}
}

// The GPU target is the same rule with no pixels to look at: a device
// resized since the last painted frame holds undefined memory, so a
// present that painted nothing may not swap it. A flushOnly present is a
// flush of what was already drawn, so it cannot mark the target painted,
// and a reallocation after a paint re-arms the flag because that paint
// went into the target it replaced.
func TestWaylandGPUTargetNotShownBeforeItIsPainted(t *testing.T) {
	var s wlSurface
	if s.gpuUnpainted {
		t.Fatal("a surface with no GPU device starts unpainted")
	}
	s.noteGPUTargetAllocated()
	if !s.gpuUnpainted {
		t.Fatal("a freshly allocated target must not be shown")
	}
	// A flush of a frame painted into the old target proves nothing.
	s.notePainted(true)
	if !s.gpuUnpainted {
		t.Fatal("a flushOnly present marked an unpainted target ready")
	}
	s.notePainted(false)
	if s.gpuUnpainted {
		t.Fatal("a painted frame did not mark the target ready")
	}
	// A reallocation after that paint arms it again.
	s.noteGPUTargetAllocated()
	if !s.gpuUnpainted {
		t.Fatal("a target reallocated after the paint must not be shown")
	}
}
