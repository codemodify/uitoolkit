package platform

import (
	"os"
	"strings"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// UITK_WAYLAND_PRESENT selects the Wayland buffer import path.
//
//	auto   (default) — wl_shm XRGB8888 (opaque). linux-dmabuf is opt-in until
//	                   the GBM/heap upload is proven opaque on Mutter/Weston.
//	shm              — force wl_shm
//	dmabuf           — try linux-dmabuf (XRGB/XBGR when advertised); fall back
//	                   to wl_shm on alloc failure or a blank first upload
const (
	EnvWaylandPresent    = "UITK_WAYLAND_PRESENT"
	WaylandPresentAuto   = "auto"
	WaylandPresentSHM    = "shm"
	WaylandPresentDmabuf = "dmabuf"
)

// DRM fourcc / modifier values used by linux-dmabuf (native endian in protocol
// tables; these constants are the packed little-endian codes).
const (
	drmFormatARGB8888 = uint32('A') | uint32('R')<<8 | uint32('2')<<16 | uint32('4')<<24
	drmFormatXRGB8888 = uint32('X') | uint32('R')<<8 | uint32('2')<<16 | uint32('4')<<24
	drmFormatABGR8888 = uint32('A') | uint32('B')<<8 | uint32('2')<<16 | uint32('4')<<24
	drmFormatXBGR8888 = uint32('X') | uint32('B')<<8 | uint32('2')<<16 | uint32('4')<<24

	drmModLinear  = uint64(0)
	drmModInvalid = uint64(0x00ffffffffffffff)
)

type dmaFmtMod struct {
	format   uint32
	modifier uint64
}

// WaylandPresentPref returns the UITK_WAYLAND_PRESENT choice (auto/shm/dmabuf).
func WaylandPresentPref() string {
	return parseWaylandPresent(os.Getenv(EnvWaylandPresent))
}

func parseWaylandPresent(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case WaylandPresentSHM:
		return WaylandPresentSHM
	case WaylandPresentDmabuf:
		return WaylandPresentDmabuf
	default:
		return WaylandPresentAuto
	}
}

func dmabufFormatUsable(fmt uint32) bool {
	switch fmt {
	case drmFormatARGB8888, drmFormatXRGB8888, drmFormatABGR8888, drmFormatXBGR8888:
		return true
	}
	return false
}

func dmabufNeedsSwizzle(fmt uint32) bool {
	return fmt == drmFormatARGB8888 || fmt == drmFormatXRGB8888
}

func modifierCPUFriendly(mod uint64) bool {
	return mod == drmModLinear || mod == drmModInvalid
}

func dmabufFormatOpaque(fmt uint32) bool {
	return fmt == drmFormatXRGB8888 || fmt == drmFormatXBGR8888
}

// waylandWantDmabuf is true only for the explicit opt-in. auto equals shm
// (v0.4.1): default/auto dmabuf presented a fully transparent window on
// real compositors while UITK_WAYLAND_PRESENT=shm painted correctly.
func waylandWantDmabuf() bool {
	return WaylandPresentPref() == WaylandPresentDmabuf
}

func pickDmabufFormat(pairs []dmaFmtMod) (format uint32, modifier uint64, ok bool) {
	if len(pairs) == 0 {
		return 0, 0, false
	}
	// Opaque fourccs first: ARGB with alpha=0 (empty GBM map or wrong
	// swizzle) composites as a fully transparent window.
	order := []uint32{drmFormatXRGB8888, drmFormatXBGR8888, drmFormatARGB8888, drmFormatABGR8888}
	for _, fmt := range order {
		var linear, invalid bool
		found := false
		for _, p := range pairs {
			if p.format != fmt {
				continue
			}
			found = true
			if p.modifier == drmModLinear {
				linear = true
			}
			if p.modifier == drmModInvalid {
				invalid = true
			}
		}
		if !found {
			continue
		}
		if linear {
			return fmt, drmModLinear, true
		}
		if invalid {
			return fmt, drmModInvalid, true
		}
		// Format advertised only with tiled modifiers — skip; we need CPU linear.
	}
	return 0, 0, false
}

// imageRaw is the paintengine2d CPU pixmap export used by Wayland present:
// premul 8-bit RGBA pointer, size, and row stride (v0.7.2 Image.Pix / RowStride).
func imageRaw(img *paintengine2d.Image) (pix []byte, w, h, stride int) {
	if img == nil {
		return nil, 0, 0, 0
	}
	return img.Pix, img.Width, img.Height, img.RowStride()
}

// copyImageRect copies a damage rect from a paintengine2d pixmap into a
// 32-bit destination. One pass only (no intermediate buffer, no second
// swizzle). swizzleRB true converts premul RGBA → BGRA (DRM / wl_shm
// XRGB8888 and ARGB8888). false is a row memcpy (XBGR8888 / ABGR8888).
// opaque true forces destination A/X to 0xFF so an opaque UI cannot
// present as a fully transparent ARGB window.
func copyImageRect(dst []byte, dstStride int, img *paintengine2d.Image, r paintengine2d.Rect, swizzleRB, opaque bool) {
	if img == nil || len(dst) == 0 {
		return
	}
	pix, w, h, srcStride := imageRaw(img)
	if len(pix) == 0 || w < 1 || h < 1 {
		return
	}
	x0, y0, x1, y1 := r.IntBounds()
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > w {
		x1 = w
	}
	if y1 > h {
		y1 = h
	}
	if x0 >= x1 || y0 >= y1 {
		return
	}
	if dstStride < w*4 {
		dstStride = w * 4
	}
	if srcStride < w*4 {
		srcStride = w * 4
	}
	rowBytes := (x1 - x0) * 4
	for y := y0; y < y1; y++ {
		si := y*srcStride + x0*4
		di := y*dstStride + x0*4
		if si+rowBytes > len(pix) || di+rowBytes > len(dst) {
			return
		}
		src := pix[si : si+rowBytes]
		out := dst[di : di+rowBytes]
		copyPresentRow(out, src, swizzleRB, opaque)
	}
}

// copyPresentRow is one pass over a tightly packed 32-bit row (no
// intermediate buffer). Little-endian uint32: RGBA 0xAABBGGRR → BGRA
// 0xAARRGGBB; opaque forces A/X to 0xFF.
func copyPresentRow(dst, src []byte, swizzleRB, opaque bool) {
	n := len(src) / 4
	if n == 0 || len(dst) < n*4 {
		return
	}
	if !swizzleRB && !opaque {
		copy(dst, src[:n*4])
		return
	}
	sp := unsafe.Slice((*uint32)(unsafe.Pointer(&src[0])), n)
	dp := unsafe.Slice((*uint32)(unsafe.Pointer(&dst[0])), n)
	if swizzleRB && opaque {
		for i, p := range sp {
			dp[i] = p&0x0000ff00 | (p&0x000000ff)<<16 | (p&0x00ff0000)>>16 | 0xff000000
		}
		return
	}
	if swizzleRB {
		for i, p := range sp {
			dp[i] = p&0xff00ff00 | (p&0x000000ff)<<16 | (p&0x00ff0000)>>16
		}
		return
	}
	// memcpy + force opaque X/A
	for i, p := range sp {
		dp[i] = p | 0xff000000
	}
}

func pixmapHasOpaque(img *paintengine2d.Image) bool {
	if img == nil {
		return false
	}
	pix := img.Pix
	if len(pix) >= 4 && pix[3] != 0 {
		return true
	}
	// Sample a handful of pixels — do not walk a 1080p frame on present.
	step := 64 * 4
	if step < 4 {
		step = 4
	}
	for i := 3; i < len(pix); i += step {
		if pix[i] != 0 {
			return true
		}
	}
	if len(pix) >= 4 && pix[len(pix)-1] != 0 {
		return true
	}
	return false
}

// destAlphaAllZero reports whether the first n bytes of a 32-bit buffer
// have a zero alpha/padding byte in every pixel (typical empty GBM map).
func destAlphaAllZero(dst []byte, n int) bool {
	if n > len(dst) {
		n = len(dst)
	}
	if n < 4 {
		return true
	}
	any := false
	for i := 3; i < n; i += 4 {
		any = true
		if dst[i] != 0 {
			return false
		}
	}
	return any
}
