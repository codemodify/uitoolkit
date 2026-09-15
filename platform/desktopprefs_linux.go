//go:build linux

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest     = "org.freedesktop.portal.Desktop"
	portalPath     = "/org/freedesktop/portal/desktop"
	portalSettings = "org.freedesktop.portal.Settings"
	appearanceNS   = "org.freedesktop.appearance"
)

// A portal that D-Bus has to start can take seconds; the first read waits
// for the caller's timeout, a second try in the background this long.
const prefsRetryTimeout = 5 * time.Second

var errNoSessionBus = errors.New("platform: no session bus")

// sessionBusAddress finds the session bus without starting one: godbus's
// ConnectSessionBus runs dbus-launch when there is none (ssh, containers).
func sessionBusAddress() string {
	if a := os.Getenv("DBUS_SESSION_BUS_ADDRESS"); a != "" && a != "autolaunch:" {
		return a
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		p := filepath.Join(dir, "bus")
		if st, err := os.Stat(p); err == nil && st.Mode()&os.ModeSocket != 0 {
			return "unix:path=" + dbus.EscapeBusAddressValue(p)
		}
	}
	return ""
}

// DialSessionBus connects to the session bus without starting one (see
// sessionBusAddress); the caller closes the connection.
func DialSessionBus() (*dbus.Conn, error) { return connectSessionBus() }

var sharedBus struct {
	once sync.Once
	conn *dbus.Conn
	err  error
}

// SharedSessionBus is the process's one session-bus connection for
// watchers (the desktop's preferences, the accessibility status): opened
// on first use, without starting a bus, and never closed. Watchers share
// its signals, so each filters by name and path.
func SharedSessionBus() (*dbus.Conn, error) {
	sharedBus.once.Do(func() { sharedBus.conn, sharedBus.err = connectSessionBus() })
	return sharedBus.conn, sharedBus.err
}

func connectSessionBus() (*dbus.Conn, error) {
	addr := sessionBusAddress()
	if addr == "" {
		return nil, errNoSessionBus
	}
	return dbus.Connect(addr)
}

// set records one org.freedesktop.appearance key; it reports whether the
// key is one DesktopPrefs carries.
func (p *DesktopPrefs) set(key string, v dbus.Variant) bool {
	switch key {
	case "color-scheme":
		u, ok := uintOf(v)
		p.ColorScheme = parseScheme(u)
		return ok
	case "reduced-motion":
		u, ok := uintOf(v)
		p.ReducedMotion = u == 1
		return ok
	case "contrast":
		u, ok := uintOf(v)
		p.HighContrast = u == 1
		return ok
	case "accent-color":
		p.Accent, p.HasAccent = accentOf(v)
		return true
	}
	return false
}

// uintOf unwraps the portal's (possibly doubly wrapped) uint32: Read
// wraps its value in one more variant than ReadOne and ReadAll.
func uintOf(v dbus.Variant) (uint32, bool) {
	for i := 0; i < 3; i++ {
		switch x := v.Value().(type) {
		case uint32:
			return x, true
		case dbus.Variant:
			v = x
		default:
			return 0, false
		}
	}
	return 0, false
}

// accentOf reads the (ddd) accent; a channel outside 0..1 means unset.
func accentOf(v dbus.Variant) ([3]float64, bool) {
	var c [3]float64
	for i := 0; i < 3; i++ {
		if w, ok := v.Value().(dbus.Variant); ok {
			v = w
			continue
		}
		break
	}
	fs, ok := v.Value().([]interface{})
	if !ok || len(fs) != 3 {
		return c, false
	}
	for i, f := range fs {
		x, ok := f.(float64)
		if !ok || x < 0 || x > 1 {
			return [3]float64{}, false
		}
		c[i] = x
	}
	return c, true
}

func readPrefs(conn *dbus.Conn, timeout time.Duration) (DesktopPrefs, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var all map[string]map[string]dbus.Variant
	call := conn.Object(portalDest, portalPath).CallWithContext(ctx, portalSettings+".ReadAll", 0, []string{appearanceNS})
	if call.Err != nil || call.Store(&all) != nil {
		return DesktopPrefs{}, false
	}
	var p DesktopPrefs
	for k, v := range all[appearanceNS] {
		p.set(k, v)
	}
	return p, true
}

// ReadDesktopPrefs asks the desktop portal for the appearance preferences,
// waiting at most timeout; ok is false without a session bus or portal.
func ReadDesktopPrefs(timeout time.Duration) (DesktopPrefs, bool) {
	p, ok, stop := WatchDesktopPrefs(timeout, nil)
	stop()
	return p, ok
}

// WatchDesktopPrefs reads the desktop's appearance preferences (waiting at
// most timeout) and then calls fn, on its own goroutine, with the whole set
// whenever the desktop changes one. When the first read times out (a
// portal still starting), the set arrives through fn once it is known. ok
// is false without a session bus or portal; stop is always safe to call.
func WatchDesktopPrefs(timeout time.Duration, fn func(DesktopPrefs)) (prefs DesktopPrefs, ok bool, stop func()) {
	if fn == nil {
		conn, err := connectSessionBus()
		if err != nil {
			return DesktopPrefs{}, false, func() {}
		}
		prefs, ok = readPrefs(conn, timeout)
		conn.Close()
		return prefs, ok, func() {}
	}
	conn, err := SharedSessionBus()
	if err != nil {
		return DesktopPrefs{}, false, func() {}
	}
	// Subscribe before reading, so no change falls between the two.
	match := []dbus.MatchOption{
		dbus.WithMatchInterface(portalSettings),
		dbus.WithMatchMember("SettingChanged"),
		dbus.WithMatchObjectPath(portalPath),
	}
	if err := conn.AddMatchSignal(match...); err != nil {
		return DesktopPrefs{}, false, func() {}
	}
	ch := make(chan *dbus.Signal, 8)
	conn.Signal(ch)
	prefs, ok = readPrefs(conn, timeout)
	done := make(chan struct{})
	go func(cur DesktopPrefs, known bool) {
		if !known {
			if p, ok := readPrefs(conn, prefsRetryTimeout); ok {
				cur = p
				fn(cur)
			}
		}
		for {
			select {
			case <-done:
				return
			case sig, open := <-ch:
				if !open {
					return
				}
				if sig.Name != portalSettings+".SettingChanged" || sig.Path != portalPath || len(sig.Body) < 3 {
					continue
				}
				ns, _ := sig.Body[0].(string)
				key, _ := sig.Body[1].(string)
				v, isVar := sig.Body[2].(dbus.Variant)
				if ns != appearanceNS || !isVar {
					continue
				}
				if cur.set(key, v) {
					fn(cur)
				}
			}
		}
	}(prefs, ok)
	var once sync.Once
	return prefs, ok, func() {
		once.Do(func() {
			close(done)
			conn.RemoveSignal(ch)
			_ = conn.RemoveMatchSignal(match...)
		})
	}
}
