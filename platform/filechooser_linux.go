//go:build linux

package platform

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalFileChooser = "org.freedesktop.portal.FileChooser"
	portalRequest     = "org.freedesktop.portal.Request"
)

// FileChooserAvailable reports whether the desktop's file chooser portal
// answers (a quick check, no dialog).
func FileChooserAvailable() bool {
	conn, err := SharedSessionBus()
	if err != nil {
		return false
	}
	var version uint32
	v, err := conn.Object(portalDest, portalPath).GetProperty(portalFileChooser + ".version")
	if err != nil {
		return false
	}
	version, _ = v.Value().(uint32)
	return version >= 1
}

// OpenFileChooser shows the desktop's own file dialog (KDE's, GNOME's)
// through the XDG portal and calls done, on its own goroutine, with the
// chosen paths (nil when cancelled). It reports false, without calling
// done, when there is no portal: the caller shows the toolkit's dialog.
func OpenFileChooser(opts FileChooserOptions, done func(paths []string)) bool {
	conn, err := SharedSessionBus()
	if err != nil {
		return false
	}
	return openFileChooser(conn, portalDest, opts, done)
}

var chooserToken atomic.Uint64

func openFileChooser(conn *dbus.Conn, dest string, opts FileChooserOptions, done func(paths []string)) bool {
	token := fmt.Sprintf("uitk%d_%d", time.Now().UnixNano()%1e9, chooserToken.Add(1))
	// The Request object's path is known before the call: subscribe first,
	// so a quick answer is not missed.
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	reqPath := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)
	match := []dbus.MatchOption{
		dbus.WithMatchInterface(portalRequest),
		dbus.WithMatchMember("Response"),
		dbus.WithMatchObjectPath(reqPath),
	}
	if conn.AddMatchSignal(match...) != nil {
		return false
	}
	ch := make(chan *dbus.Signal, 4)
	conn.Signal(ch)
	cleanup := func() {
		conn.RemoveSignal(ch)
		_ = conn.RemoveMatchSignal(match...)
	}
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"modal":        dbus.MakeVariant(true),
	}
	if opts.Multiple && !opts.Save {
		options["multiple"] = dbus.MakeVariant(true)
	}
	if opts.Directory && !opts.Save {
		options["directory"] = dbus.MakeVariant(true)
	}
	if opts.AcceptLabel != "" {
		options["accept_label"] = dbus.MakeVariant(opts.AcceptLabel)
	}
	if len(opts.Filters) > 0 {
		type pattern struct {
			Kind uint32
			Glob string
		}
		type filter struct {
			Name     string
			Patterns []pattern
		}
		var fs []filter
		for _, f := range opts.Filters {
			var ps []pattern
			for _, g := range f.Patterns {
				ps = append(ps, pattern{0, g})
			}
			fs = append(fs, filter{f.Name, ps})
		}
		options["filters"] = dbus.MakeVariant(fs)
	}
	if opts.Folder != "" {
		// A byte string with its terminating NUL, as the portal wants.
		options["current_folder"] = dbus.MakeVariant(append([]byte(opts.Folder), 0))
	}
	method := portalFileChooser + ".OpenFile"
	if opts.Save {
		method = portalFileChooser + ".SaveFile"
		if opts.Name != "" {
			options["current_name"] = dbus.MakeVariant(opts.Name)
		}
	}
	var handle dbus.ObjectPath
	if err := conn.Object(dest, portalPath).Call(method, 0, opts.ParentWindow, opts.Title, options).Store(&handle); err != nil {
		cleanup()
		return false
	}
	go func() {
		defer cleanup()
		for sig := range ch {
			// Older portals answer on a path of their own choosing.
			if sig.Name != portalRequest+".Response" || (sig.Path != reqPath && sig.Path != handle) {
				continue
			}
			var paths []string
			if len(sig.Body) >= 2 {
				if code, _ := sig.Body[0].(uint32); code == 0 {
					if results, ok := sig.Body[1].(map[string]dbus.Variant); ok {
						if uris, ok := results["uris"].Value().([]string); ok {
							paths = pathsOfURIs(uris)
						}
					}
				}
			}
			done(paths)
			return
		}
	}()
	return true
}

// pathsOfURIs turns the portal's file:// URIs into paths.
func pathsOfURIs(uris []string) []string {
	var out []string
	for _, u := range uris {
		p, err := url.Parse(u)
		if err != nil || p.Scheme != "file" {
			continue
		}
		out = append(out, p.Path)
	}
	return out
}
