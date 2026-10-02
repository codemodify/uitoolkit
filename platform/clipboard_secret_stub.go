//go:build (!linux && !windows && !darwin) || (!cgo && !windows)

package platform

// Without a native clipboard there is nothing to hint to and nothing to
// clear: the secret lives in [SecretClip]'s own buffer, which is wiped
// on the timeout like everywhere else.
func clipboardNativeSetSecret([]byte) {}
func clipboardNativeClear()           {}

// clipboardNativeGetSecret: no bytes path here yet, so the caller falls
// back to the ordinary read. See the Linux file for what this is for.
func clipboardNativeGetSecret() ([]byte, bool) { return nil, false }
