//go:build !linux || !cgo

package platform

func dmabufAllocatorOK() bool     { return false }
func dmabufAllocatorName() string { return "" }
func waylandUsingDmabuf() bool    { return false }
