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
	for _, key := range []string{"UITK_SCALE", "GDK_SCALE", "QT_SCALE_FACTOR"} {
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
