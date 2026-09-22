//go:build linux && cgo

package platform

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// wlFake is a Wayland compositor just big enough to hold a window up and
// read back what the client says: it advertises the globals it is given,
// answers wl_display.sync, and logs the requests a test cares about —
// KWin's decoration palette and xdg-toplevel-icon — with the pixels of
// every wl_shm buffer made, read through the pool's own file descriptor.
// The client reaches it through WAYLAND_SOCKET, so nothing on the user's
// desktop is touched.
type wlFake struct {
	fd      int
	globals []string // "interface:version"
	mu      sync.Mutex
	log     []string
	objs    map[uint32]string // client object id -> interface
	fds     []int             // received, not yet claimed by a request
	pools   map[uint32][]byte // wl_shm_pool id -> its mapping
	bufs    map[uint32][]byte // wl_buffer id -> its bytes
	done    chan struct{}
}

// startWlFake starts the fake on one end of a socket pair and points
// WAYLAND_SOCKET at the other.
func startWlFake(t *testing.T, globals ...string) *wlFake {
	t.Helper()
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	f := &wlFake{
		fd:      pair[0],
		globals: append([]string{"wl_compositor:4", "wl_shm:1", "xdg_wm_base:1"}, globals...),
		objs:    map[uint32]string{1: "wl_display"},
		pools:   map[uint32][]byte{},
		bufs:    map[uint32][]byte{},
		done:    make(chan struct{}),
	}
	t.Setenv("WAYLAND_SOCKET", fmt.Sprint(pair[1]))
	t.Setenv(EnvPaint, "cpu")
	go f.serve()
	t.Cleanup(func() {
		syscall.Close(f.fd)
		<-f.done
		for _, m := range f.pools {
			syscall.Munmap(m)
		}
	})
	return f
}

// requests is the log so far.
func (f *wlFake) requests() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.log...)
}

func (f *wlFake) buffer(id string) []byte {
	var n uint32
	fmt.Sscan(id, &n)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.bufs[n]
}

func (f *wlFake) logf(format string, args ...any) {
	f.log = append(f.log, fmt.Sprintf(format, args...))
}

func (f *wlFake) serve() {
	defer close(f.done)
	var pending []byte
	buf := make([]byte, 1<<16)
	oob := make([]byte, syscall.CmsgSpace(4*28))
	for {
		n, oobn, _, _, err := syscall.Recvmsg(f.fd, buf, oob, 0)
		if err != nil || n <= 0 {
			return
		}
		if oobn > 0 {
			if msgs, err := syscall.ParseSocketControlMessage(oob[:oobn]); err == nil {
				for _, m := range msgs {
					if fds, err := syscall.ParseUnixRights(&m); err == nil {
						f.fds = append(f.fds, fds...)
					}
				}
			}
		}
		pending = append(pending, buf[:n]...)
		for len(pending) >= 8 {
			size := int(binary.LittleEndian.Uint32(pending[4:]) >> 16)
			if size < 8 || len(pending) < size {
				break
			}
			f.request(binary.LittleEndian.Uint32(pending), uint16(binary.LittleEndian.Uint32(pending[4:])), pending[8:size])
			pending = pending[size:]
		}
	}
}

// wlArgs reads a request's arguments.
type wlArgs struct{ b []byte }

func (a *wlArgs) u() uint32 {
	if len(a.b) < 4 {
		return 0
	}
	v := binary.LittleEndian.Uint32(a.b)
	a.b = a.b[4:]
	return v
}

func (a *wlArgs) s() string {
	n := int(a.u())
	if n == 0 || len(a.b) < n {
		return ""
	}
	s := string(a.b[:n-1])
	a.b = a.b[(n+3)&^3:]
	return s
}

func (f *wlFake) send(obj uint32, op uint16, args ...any) {
	var body []byte
	for _, a := range args {
		switch v := a.(type) {
		case uint32:
			body = binary.LittleEndian.AppendUint32(body, v)
		case string:
			body = binary.LittleEndian.AppendUint32(body, uint32(len(v)+1))
			body = append(body, v...)
			body = append(body, 0)
			for len(body)%4 != 0 {
				body = append(body, 0)
			}
		}
	}
	msg := binary.LittleEndian.AppendUint32(nil, obj)
	msg = binary.LittleEndian.AppendUint32(msg, uint32(8+len(body))<<16|uint32(op))
	syscall.Write(f.fd, append(msg, body...))
}

func (f *wlFake) request(obj uint32, op uint16, body []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a := &wlArgs{body}
	iface := f.objs[obj]
	switch {
	case iface == "wl_display" && op == 0: // sync
		cb := a.u()
		f.send(cb, 0, uint32(1))
		f.send(1, 1, cb) // delete_id
	case iface == "wl_display" && op == 1: // get_registry
		reg := a.u()
		f.objs[reg] = "wl_registry"
		for i, g := range f.globals {
			name, ver, _ := strings.Cut(g, ":")
			var v uint32
			fmt.Sscan(ver, &v)
			f.send(reg, 0, uint32(i+1), name, v)
		}
	case iface == "wl_registry" && op == 0: // bind
		a.u()
		name, _, id := a.s(), a.u(), a.u()
		f.objs[id] = name
		f.logf("bind %s", name)
	case iface == "wl_compositor" && op == 0:
		f.objs[a.u()] = "wl_surface"
	case iface == "xdg_wm_base" && op == 2:
		f.objs[a.u()] = "xdg_surface"
	case iface == "xdg_surface" && op == 1:
		f.objs[a.u()] = "xdg_toplevel"
	case iface == "wl_shm" && op == 0: // create_pool(id, fd, size)
		id, size := a.u(), a.u()
		if len(f.fds) > 0 {
			fd := f.fds[0]
			f.fds = f.fds[1:]
			if m, err := syscall.Mmap(fd, 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED); err == nil {
				f.pools[id] = m
			}
			syscall.Close(fd)
		}
		f.objs[id] = "wl_shm_pool"
	case iface == "wl_shm_pool" && op == 0: // create_buffer(id, offset, w, h, stride, format)
		id, off, w, h, stride, format := a.u(), a.u(), a.u(), a.u(), a.u(), a.u()
		f.objs[id] = "wl_buffer"
		if m := f.pools[obj]; int(off+stride*h) <= len(m) {
			f.bufs[id] = append([]byte(nil), m[off:off+stride*h]...)
		}
		f.logf("wl_buffer %d %dx%d stride %d format %d", id, w, h, stride, format)
	case iface == "wl_buffer" && op == 0:
		f.logf("wl_buffer.destroy %d", obj)
	case iface == "org_kde_kwin_server_decoration_palette_manager" && op == 0:
		id, surf := a.u(), a.u()
		f.objs[id] = "org_kde_kwin_server_decoration_palette"
		f.logf("palette.create on a %s", f.objs[surf])
	case iface == "org_kde_kwin_server_decoration_palette" && op == 0:
		f.logf("palette.set_palette %q", a.s())
	case iface == "org_kde_kwin_server_decoration_palette" && op == 1:
		f.logf("palette.release")
	case iface == "xdg_toplevel_icon_manager_v1" && op == 1:
		id := a.u()
		f.objs[id] = "xdg_toplevel_icon_v1"
		f.logf("icon.create %d", id)
	case iface == "xdg_toplevel_icon_manager_v1" && op == 2:
		top, icon := a.u(), a.u()
		f.logf("icon.set_icon on a %s: %d", f.objs[top], icon)
	case iface == "xdg_toplevel_icon_v1" && op == 0:
		f.logf("icon.destroy %d", obj)
	case iface == "xdg_toplevel_icon_v1" && op == 2:
		b, scale := a.u(), a.u()
		f.logf("icon.add_buffer %d scale %d", b, scale)
	}
}

// fakeSurface is a Wayland window on the fake compositor.
func fakeSurface(t *testing.T) *wlSurface {
	t.Helper()
	wlMu.Lock()
	busy := wlc != nil
	wlMu.Unlock()
	if busy {
		t.Skip("a Wayland connection is already open in this process")
	}
	s, err := WaylandBackend{}.NewSurface(WindowOptions{Title: "fake", Width: 200, Height: 100})
	if err != nil {
		t.Fatal(err)
	}
	ws := s.(*wlSurface)
	t.Cleanup(func() { ws.Close() })
	return ws
}

// inOrder reports whether log holds a line starting with each of want, in
// order.
func inOrder(log []string, want ...string) bool {
	i := 0
	for _, l := range log {
		if i < len(want) && strings.HasPrefix(l, want[i]) {
			i++
		}
	}
	return i == len(want)
}

// Where KWin advertises its palette manager, the window's palette is put on
// the wire once per change: the object on the window's wl_surface, then
// set_palette; the same palette again says nothing; "" asks for the
// desktop's colours back; closing the window releases the object.
func TestWaylandDecorationPaletteOnTheWire(t *testing.T) {
	f := startWlFake(t, "org_kde_kwin_server_decoration_palette_manager:1")
	s := fakeSurface(t)
	if !s.DecorationPaletteSupported() {
		t.Fatal("the palette manager was advertised but not bound")
	}
	s.SetDecorationPalette("/cache/uitk-a.colors")
	s.SetDecorationPalette("/cache/uitk-a.colors")
	s.SetDecorationPalette("/cache/uitk-b.colors")
	s.SetDecorationPalette("")
	s.conn.roundtrip()
	log := f.requests()
	if !inOrder(log,
		"bind org_kde_kwin_server_decoration_palette_manager",
		"palette.create on a wl_surface",
		`palette.set_palette "/cache/uitk-a.colors"`,
		`palette.set_palette "/cache/uitk-b.colors"`,
		`palette.set_palette ""`) {
		t.Fatalf("requests:\n%s", strings.Join(log, "\n"))
	}
	if n := strings.Count(strings.Join(log, "\n"), "palette.create"); n != 1 {
		t.Fatalf("%d palette objects, want one:\n%s", n, strings.Join(log, "\n"))
	}
	if n := strings.Count(strings.Join(log, "\n"), "uitk-a.colors"); n != 1 {
		t.Fatalf("the same palette was sent %d times", n)
	}
	s.Close()
	// Close flushed the release; the socket's far end is now closed, and
	// the fake has read everything the client sent.
	waitFake(t, f, "palette.release")
}

// Without the global nothing is sent, and the surface says so.
func TestWaylandDecorationPaletteNeedsKWin(t *testing.T) {
	f := startWlFake(t)
	s := fakeSurface(t)
	if s.DecorationPaletteSupported() {
		t.Fatal("a palette is supported with no palette manager")
	}
	s.SetDecorationPalette("/cache/uitk-a.colors")
	s.conn.roundtrip()
	for _, l := range f.requests() {
		if strings.Contains(l, "palette") {
			t.Fatalf("sent %q to a compositor without the palette manager", l)
		}
	}
}

// The icon: one wl_shm buffer per size, square ARGB8888 with a packed
// stride, holding the premultiplied pixels; an icon object with each
// buffer at scale 1, set on the toplevel; a new icon replaces it and the
// old icon goes before its buffers.
func TestWaylandIconOnTheWire(t *testing.T) {
	f := startWlFake(t, "xdg_toplevel_icon_manager_v1:1")
	s := fakeSurface(t)
	small, big := iconTestImage(3), iconTestImage(5)
	s.SetIcon([]*paintengine2d.Image{big, small})
	s.conn.roundtrip()
	log := f.requests()
	if !inOrder(log, "bind xdg_toplevel_icon_manager_v1", "wl_buffer", "wl_buffer", "icon.create", "icon.add_buffer", "icon.add_buffer", "icon.set_icon on a xdg_toplevel") {
		t.Fatalf("requests:\n%s", strings.Join(log, "\n"))
	}
	var bufs []string
	for _, l := range log {
		if strings.HasPrefix(l, "wl_buffer ") {
			bufs = append(bufs, l)
		}
	}
	for i, im := range []*paintengine2d.Image{small, big} {
		var id, w, h, stride, format uint32
		fmt.Sscanf(bufs[i], "wl_buffer %d %dx%d stride %d format %d", &id, &w, &h, &stride, &format)
		if w != uint32(im.Width) || h != w || stride != 4*w || format != 0 {
			t.Fatalf("buffer %q, want %dx%d stride %d ARGB8888 (0)", bufs[i], im.Width, im.Width, 4*im.Width)
		}
		got, want := f.buffer(fmt.Sprint(id)), wlIconPixels(im)
		if string(got) != string(want) {
			t.Fatalf("buffer %d holds % x, want % x", id, got, want)
		}
	}
	s.SetIcon([]*paintengine2d.Image{small})
	s.conn.roundtrip()
	log = f.requests()
	if !inOrder(log, "icon.set_icon", "icon.create", "icon.add_buffer", "icon.set_icon", "icon.destroy", "wl_buffer.destroy", "wl_buffer.destroy") {
		t.Fatalf("replacing the icon:\n%s", strings.Join(log, "\n"))
	}
}

// waitFake waits until the fake has logged a line starting with want.
func waitFake(t *testing.T, f *wlFake, want string) {
	t.Helper()
	<-f.done
	for _, l := range f.requests() {
		if strings.HasPrefix(l, want) {
			return
		}
	}
	t.Fatalf("no %q:\n%s", want, strings.Join(f.requests(), "\n"))
}
