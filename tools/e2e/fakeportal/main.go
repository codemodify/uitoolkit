// Command fakeportal stands in for xdg-desktop-portal and the notification
// server on an e2e rig instance's own D-Bus session, so the toolkit's
// portal calls can be watched without the real portal — which, started by
// D-Bus activation on a nested session, would reach for the user's own
// desktop (its Wayland socket, its document-portal mount).
//
// It owns org.freedesktop.portal.Desktop (FileChooser, OpenURI,
// Notification) and org.freedesktop.Notifications, logs every call with
// its arguments, and plays the desktop's part: a file dialog is a real
// window made by the importer helper, the child of the window the call
// named (xdg-foreign's zxdg_importer_v2 on Wayland, WM_TRANSIENT_FOR on
// X11), and a notification can be clicked after a delay (-click).
//
//	fakeportal -bus "$(cat 30/bus.addr)" -importer ./importer -click button-0
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest = "org.freedesktop.portal.Desktop"
	portalPath = "/org/freedesktop/portal/desktop"
	fdoName    = "org.freedesktop.Notifications"
	fdoPath    = "/org/freedesktop/Notifications"
)

var (
	busAddr   = flag.String("bus", "", "the rig instance's D-Bus address (N/bus.addr)")
	importer  = flag.String("importer", "", "the importer helper that makes a file dialog's window")
	dialogFor = flag.Duration("dialog", 8*time.Second, "how long a file dialog stays up")
	click     = flag.String("click", "", `click each notification after -after: "default" or "button-N"`)
	after     = flag.Duration("after", 3*time.Second, "when to click")
)

var (
	conn *dbus.Conn
	mu   sync.Mutex
	next uint32
)

func logf(format string, args ...any) { log.Printf(format, args...) }

// variants prints an a{sv} in a stable order.
func variants(m map[string]dbus.Variant) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		v := m[k]
		s := v.String()
		if len(s) > 120 {
			s = s[:120] + "…"
		}
		fmt.Fprintf(&b, " %s=%s", k, s)
	}
	return b.String()
}

type fileChooser struct{}

func (fileChooser) OpenFile(sender dbus.Sender, parent, title string, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	return dialog("OpenFile", sender, parent, title, opts)
}

func (fileChooser) SaveFile(sender dbus.Sender, parent, title string, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	return dialog("SaveFile", sender, parent, title, opts)
}

// dialog logs a FileChooser call and shows the dialog's stand-in as the
// child of parent; when it is gone, the request answers "cancelled".
func dialog(method string, sender dbus.Sender, parent, title string, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	token, _ := opts["handle_token"].Value().(string)
	path := dbus.ObjectPath(portalPath + "/request/" + strings.ReplaceAll(strings.TrimPrefix(string(sender), ":"), ".", "_") + "/" + token)
	logf("FileChooser.%s parent=%q title=%q%s", method, parent, title, variants(opts))
	go func() {
		if *importer != "" {
			cmd := exec.Command(*importer, parent, title, fmt.Sprint(int(dialogFor.Seconds())))
			cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
			if err := cmd.Run(); err != nil {
				logf("importer: %v", err)
			}
		} else {
			time.Sleep(*dialogFor)
		}
		_ = conn.Emit(path, "org.freedesktop.portal.Request.Response", uint32(1), map[string]dbus.Variant{})
		logf("FileChooser.%s answered: cancelled", method)
	}()
	return path, nil
}

type openURI struct{}

func (openURI) OpenURI(parent, uri string, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	logf("OpenURI.OpenURI parent=%q uri=%q%s", parent, uri, variants(opts))
	return portalPath + "/request/x/openuri", nil
}

func fdPath(fd dbus.UnixFD) string {
	p, _ := os.Readlink(fmt.Sprintf("/proc/self/fd/%d", int(fd)))
	return p
}

func (openURI) OpenFile(parent string, fd dbus.UnixFD, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	logf("OpenURI.OpenFile parent=%q file=%q%s", parent, fdPath(fd), variants(opts))
	return portalPath + "/request/x/openfile", nil
}

func (openURI) OpenDirectory(parent string, fd dbus.UnixFD, opts map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	logf("OpenURI.OpenDirectory parent=%q dir=%q%s", parent, fdPath(fd), variants(opts))
	return portalPath + "/request/x/opendir", nil
}

type notifyPortal struct{}

func (notifyPortal) AddNotification(id string, n map[string]dbus.Variant) *dbus.Error {
	logf("Notification.AddNotification id=%q%s", id, variants(n))
	if *click != "" {
		go func() {
			time.Sleep(*after)
			_ = conn.Emit(portalPath, "org.freedesktop.portal.Notification.ActionInvoked", id, *click, []dbus.Variant{})
			logf("Notification.ActionInvoked id=%q action=%q", id, *click)
		}()
	}
	return nil
}

func (notifyPortal) RemoveNotification(id string) *dbus.Error {
	logf("Notification.RemoveNotification id=%q", id)
	return nil
}

type server struct{}

func (server) Notify(app string, replaces uint32, icon, summary, body string, actions []string, hints map[string]dbus.Variant, timeout int32) (uint32, *dbus.Error) {
	mu.Lock()
	id := replaces
	if id == 0 {
		next++
		id = next
	}
	mu.Unlock()
	logf("Notifications.Notify app=%q replaces=%d icon=%q summary=%q body=%q actions=%q timeout=%d%s -> %d",
		app, replaces, icon, summary, body, actions, timeout, variants(hints), id)
	if *click != "" {
		go func() {
			time.Sleep(*after)
			_ = conn.Emit(fdoPath, fdoName+".ActionInvoked", id, *click)
			logf("Notifications.ActionInvoked id=%d action=%q", id, *click)
		}()
	}
	return id, nil
}

func (server) CloseNotification(id uint32) *dbus.Error {
	logf("Notifications.CloseNotification id=%d", id)
	return nil
}

func (server) GetCapabilities() ([]string, *dbus.Error) {
	return []string{"actions", "body", "icon-static"}, nil
}

func (server) GetServerInformation() (string, string, string, string, *dbus.Error) {
	return "fakeportal", "uitoolkit e2e", "1", "1.2", nil
}

// props answers the portal interfaces' version property.
type props struct{}

func (props) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	if prop == "version" {
		return dbus.MakeVariant(uint32(3)), nil
	}
	return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("no property %s.%s", iface, prop))
}

func (props) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	return map[string]dbus.Variant{"version": dbus.MakeVariant(uint32(3))}, nil
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if *busAddr == "" {
		log.Fatal("fakeportal: -bus is required (the rig's N/bus.addr); it never uses the session bus")
	}
	// Never the user's own session bus.
	if strings.Contains(*busAddr, fmt.Sprintf("/run/user/%d/bus", os.Getuid())) {
		log.Fatal("fakeportal: that is the user's session bus")
	}
	var err error
	if conn, err = dbus.Connect(*busAddr); err != nil {
		log.Fatal(err)
	}
	for _, name := range []string{portalDest, fdoName} {
		if r, err := conn.RequestName(name, dbus.NameFlagDoNotQueue); err != nil || r != dbus.RequestNameReplyPrimaryOwner {
			log.Fatalf("fakeportal: cannot own %s (%v): something already does", name, err)
		}
	}
	must := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}
	must(conn.Export(fileChooser{}, portalPath, "org.freedesktop.portal.FileChooser"))
	must(conn.Export(openURI{}, portalPath, "org.freedesktop.portal.OpenURI"))
	must(conn.Export(notifyPortal{}, portalPath, "org.freedesktop.portal.Notification"))
	must(conn.Export(props{}, portalPath, "org.freedesktop.DBus.Properties"))
	must(conn.Export(server{}, fdoPath, fdoName))
	logf("fakeportal up on %s: %s and %s", *busAddr, portalDest, fdoName)
	select {}
}
