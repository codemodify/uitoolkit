package platform

import (
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

func scaleFromDPI(dpi float32) float32 {
	if dpi < 48 {
		return 0
	}
	return clampScale(dpi / 96)
}

// SurfaceScale is s.Scale(), or 1 when missing.
func SurfaceScale(s Surface) float32 {
	if s == nil {
		return 1
	}
	if sc := s.Scale(); sc > 0 {
		return sc
	}
	return 1
}

// AdoptDisplayScale returns the scale to use for look metrics. A higher
// native/surface factor replaces a 1× guess from New() before the display
// connection existed. It never shrinks an explicit UITK_SCALE / env scale.
func AdoptDisplayScale(current, surface float32) float32 {
	if current <= 0 {
		current = 1
	}
	if surface > current+0.01 && surface > 1.01 {
		return clampScale(surface)
	}
	return current
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
