//go:build !linux

package platform

// FileChooserAvailable: no portal outside Linux yet.
func FileChooserAvailable() bool { return false }

// OpenFileChooser: no portal outside Linux yet; the caller shows the
// toolkit's dialog.
func OpenFileChooser(opts FileChooserOptions, done func(paths []string)) bool { return false }
