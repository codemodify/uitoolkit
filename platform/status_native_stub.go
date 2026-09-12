//go:build (!linux && !windows && !darwin) || (darwin && !cgo)

package platform

func newNativeStatusItem(opts StatusItemOptions) (StatusItem, error) {
	return newStubStatusItem(opts), nil
}

func nativeStatusItemAvailable() bool { return false }
