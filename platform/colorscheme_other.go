//go:build !linux

package platform

// DesktopColorScheme has no portal to read outside Linux yet.
func DesktopColorScheme() (ColorScheme, bool) { return SchemeNoPreference, false }

// WatchColorScheme is a no-op outside Linux yet.
func WatchColorScheme(fn func(ColorScheme)) (stop func()) { return func() {} }
