//go:build linux

package platform

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/godbus/dbus/v5"
)

const portalOpenURI = "org.freedesktop.portal.OpenURI"

// oPath is Linux's O_PATH (the generic value: x86, ARM, RISC-V), which
// package syscall leaves out.
const oPath = 0x200000

// OpenURI opens uri with the desktop's application for it — a web link in
// the browser, a mailto: in the mail client, a file or a folder (a path or
// a file:// URI) in whatever opens that — as QDesktopServices::openUrl and
// gtk_show_uri do.
//
// It goes through the OpenURI portal, which is what works in a sandbox and
// what lets the desktop ask "Open with…" (Ask) as the window's child
// (ParentWindow); a file goes to it as a descriptor, which is how the
// portal wants files. Without a portal, or when it refuses, xdg-open opens
// it instead. It returns once the request is on its way (the portal
// answers before the application has started), within a couple of seconds
// at worst, and ErrNoOpener when there was nothing to hand it to.
func OpenURI(uri string, opts OpenURIOptions) error {
	if !validOpenURI(uri) {
		return ErrNoOpener
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvOpenURI))) {
	case "off":
		return nil
	case "xdg-open":
		return openURIFallback(uri)
	}
	conn, err := SharedSessionBus()
	return openURIWith(conn, err, portalDest, uri, opts)
}

// openURIWith is OpenURI on a given bus (tests use a private one): the
// portal at dest, else xdg-open.
func openURIWith(conn *dbus.Conn, connErr error, dest, uri string, opts OpenURIOptions) error {
	if connErr == nil && conn != nil && openURIPortal(conn, dest, uri, opts) == nil {
		return nil
	}
	return openURIFallback(uri)
}

// openURIFallback opens uri with xdg-open (a seam for tests, which must
// never start a browser).
var openURIFallback = func(uri string) error {
	bin, err := exec.LookPath("xdg-open")
	if err != nil {
		return ErrNoOpener
	}
	if p, ok := localPath(uri); ok {
		uri = p
	}
	cmd := exec.Command(bin, uri)
	if err := cmd.Start(); err != nil {
		return ErrNoOpener
	}
	// Reaped in the background: xdg-open lives as long as its opener does
	// on some desktops.
	go cmd.Wait()
	return nil
}

// openURIPortal hands uri to the portal at dest.
func openURIPortal(conn *dbus.Conn, dest, uri string, opts OpenURIOptions) error {
	timeout, ok := busServiceTimeout(conn, dest)
	if !ok {
		return ErrNoOpener
	}
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(portalToken()),
	}
	if opts.Ask {
		options["ask"] = dbus.MakeVariant(true)
	}
	if opts.ActivationToken != "" {
		options["activation_token"] = dbus.MakeVariant(opts.ActivationToken)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	obj := conn.Object(dest, portalPath)
	path, local := localPath(uri)
	if !local {
		return obj.CallWithContext(ctx, portalOpenURI+".OpenURI", 0, opts.ParentWindow, uri, options).Err
	}
	// A file is passed by descriptor (O_PATH: the portal reads the path
	// from it, and nothing is opened for reading); a folder opens in the
	// file manager, which is OpenDirectory from version 3.
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	fd, err := syscall.Open(path, oPath|syscall.O_CLOEXEC, 0)
	if err != nil {
		// Read-only does as well where O_PATH is not this number.
		if fd, err = syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0); err != nil {
			return err
		}
	}
	defer syscall.Close(fd)
	method := ".OpenFile"
	if st.IsDir() {
		method = ".OpenDirectory"
	}
	return obj.CallWithContext(ctx, portalOpenURI+method, 0, opts.ParentWindow, dbus.UnixFD(fd), options).Err
}
