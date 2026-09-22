package platform

import (
	"github.com/codemodify/paintengine2d"
)

// This file holds what a window tells the desktop about how to dress it
// when the desktop draws the frame, or lists the window: the colour scheme
// KWin paints its own title bar in, and the window's icon. Like frame.go it
// has no cgo and no build tag, so the encodings are tested headless; the
// backends put them on the wire (wayland_dress_linux.go, x11_dress_linux.go).

// DecorationPaletteSurface is an optional Surface capability: the desktop's
// own frame takes a colour scheme per window, so a window under the
// desktop's title bar can have it in its own colours. KWin is the desktop
// that offers it — org_kde_kwin_server_decoration_palette on Wayland,
// _KDE_NET_WM_COLOR_SCHEME on X11 — and it names a KDE colour-scheme file
// (style.KDEColorScheme writes one from a look).
type DecorationPaletteSurface interface {
	// DecorationPaletteSupported reports whether the desktop takes a
	// palette: KWin's global is advertised (Wayland), KWin is the window
	// manager (X11).
	DecorationPaletteSupported() bool
	// SetDecorationPalette names the colour-scheme file, an absolute path,
	// the desktop's frame paints the window with; "" goes back to the
	// desktop's own colours. It is sent only when it changes, and never
	// where the desktop does not take one.
	SetDecorationPalette(path string)
}

// SurfaceSetDecorationPalette names s's frame palette where s's desktop
// takes one, and reports whether it does.
func SurfaceSetDecorationPalette(s Surface, path string) bool {
	p, ok := s.(DecorationPaletteSurface)
	if !ok || !p.DecorationPaletteSupported() {
		return false
	}
	p.SetDecorationPalette(path)
	return true
}

// palettePlan is what a surface puts on the wire to take a palette: create
// the per-surface palette object (once), then set_palette.
type palettePlan struct {
	create, set bool
}

// planPalette decides the palette requests: nothing where the compositor
// advertises no palette manager (global false) or the palette is the one
// already sent; the object is made the first time there is a palette to
// send, and kept — going back to the desktop's colours is set_palette("")
// on it, which KWin reads as its own scheme.
func planPalette(global, have bool, sent, want string) palettePlan {
	if !global || want == sent {
		return palettePlan{}
	}
	if !have && want == "" {
		return palettePlan{}
	}
	return palettePlan{create: !have, set: true}
}

// IconSurface is an optional Surface capability: the window's own icon,
// which the desktop shows in its title bar, its task switcher and its task
// bar — xdg-toplevel-icon-v1 on Wayland, _NET_WM_ICON on X11. Without it a
// desktop falls back on the application's desktop entry, which a program
// run from its build tree has none of.
type IconSurface interface {
	// SetIcon gives the window its icon as square images at the sizes the
	// app has; the desktop picks the size it shows. None goes back to the
	// desktop's default icon.
	SetIcon(images []*paintengine2d.Image)
}

// SurfaceSetIcon gives s its icon where it can take one, and reports
// whether it could.
func SurfaceSetIcon(s Surface, images []*paintengine2d.Image) bool {
	is, ok := s.(IconSurface)
	if !ok {
		return false
	}
	is.SetIcon(images)
	return true
}

// iconImages is images as a window icon holds them: square and non-empty,
// one per size (the first of a size wins), smallest first. Anything else is
// dropped: both protocols state an icon as a square.
func iconImages(images []*paintengine2d.Image) []*paintengine2d.Image {
	var out []*paintengine2d.Image
	seen := map[int]bool{}
	for _, im := range images {
		if im == nil || im.Width < 1 || im.Width != im.Height || seen[im.Width] {
			continue
		}
		seen[im.Width] = true
		out = append(out, im)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Width < out[j-1].Width; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// netWMIcon is images as _NET_WM_ICON's CARDINAL array (EWMH 1.5): for each
// image its width, its height and then its pixels row by row, each one
// packed ARGB — alpha in the high byte, blue in the low one — and not
// premultiplied.
func netWMIcon(images []*paintengine2d.Image) []uint32 {
	var out []uint32
	for _, im := range iconImages(images) {
		out = append(out, uint32(im.Width), uint32(im.Height))
		for y := 0; y < im.Height; y++ {
			for x := 0; x < im.Width; x++ {
				p := im.NRGBAAt(x, y)
				out = append(out, uint32(p.A)<<24|uint32(p.R)<<16|uint32(p.G)<<8|uint32(p.B))
			}
		}
	}
	return out
}

// wlIconPixels is im as a wl_shm ARGB8888 buffer holds it: 32-bit
// little-endian pixels — blue, green, red, alpha in memory — premultiplied
// by alpha, as every wl_shm alpha format is, rows packed (stride 4 × width).
func wlIconPixels(im *paintengine2d.Image) []byte {
	out := make([]byte, im.Width*im.Height*4)
	i := 0
	for y := 0; y < im.Height; y++ {
		for x := 0; x < im.Width; x++ {
			r, g, b, a := im.PremulAt(x, y)
			out[i], out[i+1], out[i+2], out[i+3] = b, g, r, a
			i += 4
		}
	}
	return out
}
