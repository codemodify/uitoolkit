//go:build (!linux && !windows && !darwin) || (!cgo && !windows)

package platform

func clipboardNativeGet() (string, bool)        { return "", false }
func clipboardNativePrimaryGet() (string, bool) { return "", false }
func clipboardNativeSet(string)                 {}
