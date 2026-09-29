//go:build !linux && !darwin && !windows

package platform

func secureInputAvailable() bool      { return false }
func captureExclusionAvailable() bool { return false }
