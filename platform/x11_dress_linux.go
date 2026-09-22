//go:build linux && cgo

package platform

/*
#include <stdlib.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

// Format-32 property data is an array of C longs whatever their width, so
// the icon's CARDINALs are widened here rather than in Go.
static void ui_x_set_cardinals(Display *d, Window w, Atom prop, const unsigned int *v, int n) {
	long *buf = (long *)malloc(sizeof(long) * (size_t)(n > 0 ? n : 1));
	if (!buf) return;
	for (int i = 0; i < n; i++) buf[i] = (long)v[i];
	XChangeProperty(d, w, prop, XA_CARDINAL, 32, PropModeReplace, (unsigned char *)buf, n);
	free(buf);
}
static void ui_x_set_string(Display *d, Window w, Atom prop, const char *s) {
	XChangeProperty(d, w, prop, XA_STRING, 8, PropModeReplace, (const unsigned char *)s, (int)strlen(s));
}
static void ui_x_delete(Display *d, Window w, Atom prop) { XDeleteProperty(d, w, prop); }
static void ui_x_flush_dress(Display *d) { XFlush(d); }
*/
import "C"

import (
	"strings"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// The window's dress on X11 (windowdress.go): KWin's _KDE_NET_WM_COLOR_SCHEME
// and EWMH's _NET_WM_ICON. Both are properties on the client window, so a
// window re-made on another visual (x11_linux.go, recreateOnVisual) is given
// them again.

// x11Dress is a surface's share of it: the palette and icon it was given.
type x11Dress struct {
	palette string
	icon    []uint32
}

// dressAtomsLocked interns the atoms the first time they are needed.
func (c *x11Conn) dressAtomsLocked() {
	if c.atomColorScheme != 0 {
		return
	}
	c.atomColorScheme = internAtom(c.dpy, "_KDE_NET_WM_COLOR_SCHEME")
	c.atomIcon = internAtom(c.dpy, "_NET_WM_ICON")
}

// kwinLocked reports whether the window manager is KWin, the one that reads
// _KDE_NET_WM_COLOR_SCHEME (it does not list it in _NET_SUPPORTED).
func (c *x11Conn) kwinLocked() bool {
	c.supportsLocked(c.atomSupported) // reads the manager's name once
	return strings.EqualFold(c.wmName, "KWin")
}

// DecorationPaletteSupported reports whether the window manager is KWin
// (DecorationPaletteSurface).
func (s *x11Surface) DecorationPaletteSupported() bool {
	if s == nil || s.popup || s.conn == nil || s.conn.dpy == nil {
		return false
	}
	x11Mu.Lock()
	defer x11Mu.Unlock()
	return s.conn.kwinLocked()
}

// SetDecorationPalette sets _KDE_NET_WM_COLOR_SCHEME, the colour-scheme file
// KWin paints the window's frame with; "" removes it
// (DecorationPaletteSurface).
func (s *x11Surface) SetDecorationPalette(path string) {
	if s == nil || s.closed || s.popup || s.win == 0 || s.conn == nil || path == s.dress.palette {
		return
	}
	s.dress.palette = path
	x11Mu.Lock()
	s.applyPaletteLocked()
	C.ui_x_flush_dress(s.conn.dpy)
	x11Mu.Unlock()
}

func (s *x11Surface) applyPaletteLocked() {
	c := s.conn
	c.dressAtomsLocked()
	if s.dress.palette == "" {
		C.ui_x_delete(c.dpy, s.win, c.atomColorScheme)
		return
	}
	cp := C.CString(s.dress.palette)
	C.ui_x_set_string(c.dpy, s.win, c.atomColorScheme, cp)
	C.free(unsafe.Pointer(cp))
}

// SetIcon sets _NET_WM_ICON (IconSurface); no images removes it.
func (s *x11Surface) SetIcon(images []*paintengine2d.Image) {
	if s == nil || s.closed || s.popup || s.win == 0 || s.conn == nil {
		return
	}
	s.dress.icon = netWMIcon(images)
	x11Mu.Lock()
	s.applyIconLocked()
	C.ui_x_flush_dress(s.conn.dpy)
	x11Mu.Unlock()
}

func (s *x11Surface) applyIconLocked() {
	c := s.conn
	c.dressAtomsLocked()
	if len(s.dress.icon) == 0 {
		C.ui_x_delete(c.dpy, s.win, c.atomIcon)
		return
	}
	v := s.dress.icon
	C.ui_x_set_cardinals(c.dpy, s.win, c.atomIcon, (*C.uint)(unsafe.Pointer(&v[0])), C.int(len(v)))
}

// reapplyDressLocked puts the palette and the icon on a window made afresh.
func (s *x11Surface) reapplyDressLocked() {
	if s.dress.palette != "" {
		s.applyPaletteLocked()
	}
	if len(s.dress.icon) > 0 {
		s.applyIconLocked()
	}
}

var (
	_ DecorationPaletteSurface = (*x11Surface)(nil)
	_ IconSurface              = (*x11Surface)(nil)
)
