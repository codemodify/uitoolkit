package platform

import (
	"math"
	"os"
	"strconv"
	"strings"
)

// DetectScale returns the display scale for LookAndFeel metrics.
// Order: UITK_SCALE / GDK_SCALE / QT_SCALE_FACTOR, then the native
// backend (Xft.dpi or screen DPI on X11), else 1.
func DetectScale() float32 {
	if s := ScaleFromEnv(); s > 0 {
		return s
	}
	if s := nativeDetectScale(); s > 0 {
		return s
	}
	return 1
}

// ScaleFromEnv reads UITK_SCALE, then GDK_SCALE, then QT_SCALE_FACTOR.
// Returns 0 when none are set or parseable.
func ScaleFromEnv() float32 {
	for _, key := range []string{"UITK_SCALE", "GDK_SCALE", "QT_SCALE_FACTOR", "GDK_DPI_SCALE"} {
		if s := parseScale(os.Getenv(key)); s > 0 {
			return s
		}
	}
	return 0
}

func parseScale(s string) float32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil || v <= 0 {
		return 0
	}
	return clampScale(float32(v))
}

// Window geometry is stated in **logical pixels** throughout this package:
// [WindowOptions]'s Width, Height and the size limits, [Surface.Resize],
// and the width and height of an [EventResize]. A window asked for as 275
// by 116 is that many logical pixels at any display scale — 275 by 116
// device pixels at 1, 481 by 203 at 1.75 — so an application states its
// design once and the same numbers give the same window on every backend.
//
// Everything else stays device pixels: [Surface.Size] and [Surface.Buffer]
// are the pixmap, event positions are in it, and so is every rectangle a
// widget is laid out in.
//
// Wayland speaks logical pixels already (xdg_surface.set_window_geometry,
// and the compositor multiplies by the scale to get the buffer). X11 has
// no notion of a display scale at all — an X window's size *is* pixels —
// so the X11 backend converts with these two, at the one boundary where
// the numbers cross into the protocol.

// DevicePixels is a logical-pixel window measurement in device pixels.
func DevicePixels(v int, scale float32) int {
	if v == 0 {
		return 0
	}
	if scale <= 0 {
		scale = 1
	}
	return max(int(math.Round(float64(float32(v)*scale))), 1)
}

// LogicalPixels is DevicePixels' inverse: a device-pixel window
// measurement back in the logical pixels the toolkit's window geometry
// speaks. Rounding is symmetric, so a size that made the round trip out
// comes back unchanged.
func LogicalPixels(v int, scale float32) int {
	if v == 0 {
		return 0
	}
	if scale <= 0 {
		scale = 1
	}
	return max(int(math.Round(float64(float32(v)/scale))), 1)
}

func scaleFromDPI(dpi float32) float32 {
	if dpi < 48 {
		return 0
	}
	return clampScale(dpi / 96)
}

func clampScale(s float32) float32 {
	if s < 0.75 {
		s = 0.75
	}
	if s > 4 {
		s = 4
	}
	if s > 0.92 && s < 1.08 {
		return 1
	}
	return s
}

// fitLogicalSize maps a resize event to logical surface units.
// Wayland configure is logical; older present/configure paths echoed
// buffer pixels (logical*scale). Treating those as a new logical size
// re-applies scale and blows up layout on every resize.
func fitLogicalSize(inW, inH, logicalW, logicalH int, scale float32) (lw, lh int) {
	if inW < 1 {
		inW = 1
	}
	if inH < 1 {
		inH = 1
	}
	if scale <= 0 {
		scale = 1
	}
	if logicalW > 0 && logicalH > 0 && scale > 1.01 {
		bw := int(float32(logicalW)*scale + 0.999)
		bh := int(float32(logicalH)*scale + 0.999)
		if inW == bw && inH == bh {
			return logicalW, logicalH
		}
	}
	return inW, inH
}
