//go:build linux

package platform

import (
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest     = "org.freedesktop.portal.Desktop"
	portalPath     = "/org/freedesktop/portal/desktop"
	portalSettings = "org.freedesktop.portal.Settings"
	appearanceNS   = "org.freedesktop.appearance"
	colorSchemeKey = "color-scheme"
)

// DesktopColorScheme reads the desktop's light / dark preference from the
// XDG desktop portal; ok is false when there is no portal (no session bus,
// no desktop).
func DesktopColorScheme() (ColorScheme, bool) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return SchemeNoPreference, false
	}
	defer conn.Close()
	return readScheme(conn)
}

func readScheme(conn *dbus.Conn) (ColorScheme, bool) {
	obj := conn.Object(portalDest, portalPath)
	var v dbus.Variant
	// ReadOne (portal 2) returns the value; Read (portal 1) wraps it in one
	// more variant.
	if err := obj.Call(portalSettings+".ReadOne", 0, appearanceNS, colorSchemeKey).Store(&v); err != nil {
		if err := obj.Call(portalSettings+".Read", 0, appearanceNS, colorSchemeKey).Store(&v); err != nil {
			return SchemeNoPreference, false
		}
	}
	return schemeOf(v)
}

// schemeOf unwraps the portal's (possibly doubly wrapped) uint32.
func schemeOf(v dbus.Variant) (ColorScheme, bool) {
	for i := 0; i < 3; i++ {
		switch x := v.Value().(type) {
		case uint32:
			return parseScheme(x), true
		case dbus.Variant:
			v = x
		default:
			return SchemeNoPreference, false
		}
	}
	return SchemeNoPreference, false
}

// WatchColorScheme calls fn, on its own goroutine, whenever the desktop's
// light / dark preference changes. stop ends the watch; it is a no-op when
// there is no session bus.
func WatchColorScheme(fn func(ColorScheme)) (stop func()) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil || fn == nil {
		return func() {}
	}
	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface(portalSettings),
		dbus.WithMatchMember("SettingChanged"),
		dbus.WithMatchObjectPath(portalPath),
	); err != nil {
		conn.Close()
		return func() {}
	}
	ch := make(chan *dbus.Signal, 8)
	conn.Signal(ch)
	var once sync.Once
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case sig, ok := <-ch:
				if !ok {
					return
				}
				if len(sig.Body) < 3 {
					continue
				}
				ns, _ := sig.Body[0].(string)
				key, _ := sig.Body[1].(string)
				if ns != appearanceNS || key != colorSchemeKey {
					continue
				}
				if v, ok := sig.Body[2].(dbus.Variant); ok {
					if s, ok := schemeOf(v); ok {
						fn(s)
					}
				}
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(done)
			conn.RemoveSignal(ch)
			conn.Close()
		})
	}
}
