//go:build (!linux && !windows && !darwin) || (!cgo && !windows)

package platform

// Without a native clipboard there is nothing to hint to and nothing to
// clear: the secret lives in [SecretClip]'s own buffer, which is wiped
// on the timeout like everywhere else.
func clipboardNativeSetSecret([]byte) {}
func clipboardNativeClear()           {}
