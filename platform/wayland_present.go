package platform

import (
	"os"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// UITK_WAYLAND_PRESENT selects the Wayland buffer import path.
//
//	auto   (default) — linux-dmabuf when the compositor advertises a usable
//	                   format and a local allocator works; otherwise wl_shm
//	dmabuf           — same preference as auto (still falls back to wl_shm)
//	shm              — force wl_shm
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

func pickDmabufFormat(pairs []dmaFmtMod) (format uint32, modifier uint64, ok bool) {
	if len(pairs) == 0 {
		return 0, 0, false
	}
	order := []uint32{drmFormatARGB8888, drmFormatXRGB8888, drmFormatABGR8888, drmFormatXBGR8888}
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
// 32-bit destination. swizzleRB true converts premul RGBA → BGRA (DRM
// ARGB8888 / XRGB8888). false is a row memcpy (ABGR8888 / XBGR8888).
func copyImageRect(dst []byte, dstStride int, img *paintengine2d.Image, r paintengine2d.Rect, swizzleRB bool) {
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
		if !swizzleRB {
			copy(dst[di:di+rowBytes], pix[si:si+rowBytes])
			continue
		}
		src := pix[si : si+rowBytes]
		out := dst[di : di+rowBytes]
		for i := 0; i < rowBytes; i += 4 {
			out[i+0] = src[i+2] // B
			out[i+1] = src[i+1] // G
			out[i+2] = src[i+0] // R
			out[i+3] = src[i+3] // A
		}
	}
}
