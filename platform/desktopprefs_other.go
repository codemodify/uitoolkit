//go:build !linux

package platform

import "time"

// ReadDesktopPrefs has no portal to ask outside Linux yet.
func ReadDesktopPrefs(timeout time.Duration) (DesktopPrefs, bool) { return DesktopPrefs{}, false }

// WatchDesktopPrefs has no portal to watch outside Linux yet.
func WatchDesktopPrefs(timeout time.Duration, fn func(DesktopPrefs)) (DesktopPrefs, bool, func()) {
	return DesktopPrefs{}, false, func() {}
}
