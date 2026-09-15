//go:build !linux

package app

// lookNotify: outside Linux the run loop polls look.json.
type lookNotify struct{}

func newLookNotify(onEvent func()) *lookNotify { return nil }

func (n *lookNotify) watchFile(path string) {}
