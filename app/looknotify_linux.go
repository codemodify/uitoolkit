//go:build linux

package app

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// lookNotify wakes the app when a watched directory changes (inotify), so
// an idle app does not re-stat look.json three times a second.
type lookNotify struct {
	fd  int
	mu  sync.Mutex
	wds map[string]bool
}

const lookNotifyMask = syscall.IN_CLOSE_WRITE | syscall.IN_MOVED_TO | syscall.IN_MOVED_FROM |
	syscall.IN_CREATE | syscall.IN_DELETE | syscall.IN_ATTRIB

// newLookNotify starts watching; onEvent runs on the watcher's goroutine.
// nil when inotify is not available (the run loop polls then).
func newLookNotify(onEvent func()) *lookNotify {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		return nil
	}
	n := &lookNotify{fd: fd, wds: map[string]bool{}}
	go func() {
		buf := make([]byte, 4096)
		for {
			k, err := syscall.Read(fd, buf)
			if err == syscall.EINTR {
				continue
			}
			if err != nil || k <= 0 {
				return
			}
			onEvent()
		}
	}()
	return n
}

// watchFile watches the directory holding path, or the nearest existing
// ancestor until it exists (a first save creates ~/.config/uitoolkit).
func (n *lookNotify) watchFile(path string) {
	if n == nil || path == "" {
		return
	}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			n.watch(dir)
			return
		}
		if parent := filepath.Dir(dir); parent == dir {
			return
		}
	}
}

func (n *lookNotify) watch(dir string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.wds[dir] {
		return
	}
	if _, err := syscall.InotifyAddWatch(n.fd, dir, lookNotifyMask); err == nil {
		n.wds[dir] = true
	}
}
