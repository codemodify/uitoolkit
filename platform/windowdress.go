package platform

import (
	"github.com/codemodify/paintengine2d"
)

// This file holds what a window tells the desktop about how to dress it
// when the desktop draws the frame, or lists the window: the colour scheme
// KWin paints its own title bar in, and the window's icon. Both are
// [WindowFrame] requests (SetPalette, SetIcon) gated by FramePalette and
// FrameIcon; this file is only the *encoding*, which has no cgo and no
// build tag so it is tested headless. The backends put it on the wire
// (wayland_dress_linux.go, x11_dress_linux.go).

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
