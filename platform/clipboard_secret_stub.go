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

// clipboardSecretBytesPath reports whether this platform can read a secret
// from the clipboard without making a string of it. Where it can, a failed
// read is a failed read: falling back to the string reader would undo the
// whole point of the bytes path on the try that happens to succeed.
func clipboardSecretBytesPath() bool { return false }

// clipboardStillHoldsOurSecret: with no native clipboard the secret is only
// ever in [SecretClip]'s own buffer, which nothing else can take.
func clipboardStillHoldsOurSecret() bool { return true }
