//go:build darwin && cgo

package platform

// The tray menu's click callbacks.
//
// This file has its own, empty cgo preamble on purpose: a file carrying
// an //export must not define C functions, because cgo copies its
// preamble into the generated header and every definition would be
// duplicated in every translation unit that includes it. The menu
// building lives in status_darwin.go with the AppKit helpers it calls;
// what is here is the half that has to be exported.

import "C"

import "sync"

// An NSMenuItem carries its id in its tag and the action comes back
// through one exported function, so what is kept here is the other half:
// which closure an id belongs to.
//
// Ids are never reused. A menu replaced while its old one is open must
// not fire the new menu's handler for the row the user is pointing at,
// so the counter only goes up and the old generation's entries are
// dropped when the menu is.
var trayMenu struct {
	mu   sync.Mutex
	next uint64
	fns  map[uint64]func()
}

func trayMenuRegister(fn func()) uint64 {
	trayMenu.mu.Lock()
	defer trayMenu.mu.Unlock()
	trayMenu.next++
	if trayMenu.fns == nil {
		trayMenu.fns = map[uint64]func(){}
	}
	trayMenu.fns[trayMenu.next] = fn
	return trayMenu.next
}

func trayMenuForget(ids []uint64) {
	if len(ids) == 0 {
		return
	}
	trayMenu.mu.Lock()
	defer trayMenu.mu.Unlock()
	for _, id := range ids {
		delete(trayMenu.fns, id)
	}
}

//export uitkStatusMenuClick
func uitkStatusMenuClick(id uint64) {
	trayMenu.mu.Lock()
	fn := trayMenu.fns[id]
	trayMenu.mu.Unlock()
	if fn != nil {
		fn()
	}
}
