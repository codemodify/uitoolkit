//go:build linux

package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalNotification = "org.freedesktop.portal.Notification"
	// notifyKeep bounds the notifications a notifier remembers for their
	// clicks: the portal never says when one is gone.
	notifyKeep = 64
)

// Notifier sends desktop notifications and brings their clicks back into
// the app. A Linux desktop has two services for it, and the notifier takes
// whichever answers, in this order:
//
//   - In a Flatpak or Snap sandbox, org.freedesktop.portal.Notification —
//     the only way out of one.
//   - Otherwise the notification server itself, org.freedesktop.Notifications
//     (Plasma's, GNOME Shell's, dunst, mako), when one is running — which is
//     what GTK and KDE apps outside a sandbox use too: the portal names an
//     unsandboxed app by nothing at all, and GNOME's portal cannot show a
//     notification for an app it cannot name.
//   - The portal when no server is running, then the server.
//
// UITK_NOTIFY=portal or fdo picks one outright; off sends nothing. A
// service that is not running and cannot be started is passed over at
// once; one that D-Bus has to start is given a few seconds, never more.
//
// Send, Withdraw and Close may be called from any goroutine.
type Notifier struct {
	opts NotifierOptions

	mu   sync.Mutex
	conn *dbus.Conn
	// dial opens the bus (a seam for tests: a private bus); portal and
	// fdo are the services' bus names.
	dial   func() (*dbus.Conn, error)
	portal string
	fdo    string
	// backend is the service the last notification went through.
	backend string
	live    map[string]*notifyRecord
	order   []string
	byFdo   map[uint32]string
	seq     int
	ch      chan *dbus.Signal
	closed  bool
}

type notifyRecord struct {
	n   DesktopNotification
	fdo uint32
	via string
}

// NewNotifier makes a notifier. It connects to nothing until the first
// Send.
func NewNotifier(opts NotifierOptions) *Notifier {
	return newNotifier(opts, SharedSessionBus, portalDest, fdoNotify)
}

func newNotifier(opts NotifierOptions, dial func() (*dbus.Conn, error), portal, fdo string) *Notifier {
	if opts.AppName == "" {
		opts.AppName = "uitoolkit"
	}
	return &Notifier{opts: opts, dial: dial, portal: portal, fdo: fdo,
		live: map[string]*notifyRecord{}, byFdo: map[uint32]string{}}
}

// Backend is the service the last notification went through: "portal",
// "fdo", "stub" for a notifier that sends nothing, or "" before any was
// sent.
func (n *Notifier) Backend() string {
	if n == nil || n.opts.Stub {
		return "stub"
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.backend
}

// Send shows note, or replaces the one of the same ID, and returns its ID.
// It returns ErrNoNotifications, quickly, when neither service answers.
func (n *Notifier) Send(note DesktopNotification) (string, error) {
	if n == nil {
		return "", ErrNoNotifications
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if note.ID == "" {
		n.seq++
		note.ID = fmt.Sprintf("uitk-%d", n.seq)
	}
	if n.opts.Stub || n.closed {
		return note.ID, nil
	}
	want := notifyWant()
	if want == "off" || want == "none" {
		return note.ID, nil
	}
	conn, err := n.connLocked()
	if err != nil {
		return "", ErrNoNotifications
	}
	rec := n.live[note.ID]
	for _, via := range n.orderLocked(conn, want) {
		var err error
		switch via {
		case "portal":
			err = n.sendPortalLocked(conn, note)
		case "fdo":
			err = n.sendFdoLocked(conn, note, rec)
		}
		if err == nil {
			n.backend = via
			return note.ID, nil
		}
	}
	return "", ErrNoNotifications
}

// Withdraw takes the notification id down, if it is still up.
func (n *Notifier) Withdraw(id string) error {
	if n == nil || n.opts.Stub {
		return nil
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	rec := n.live[id]
	if rec == nil || n.conn == nil {
		return nil
	}
	n.forgetLocked(id)
	ctx, cancel := context.WithTimeout(context.Background(), busCallTimeout)
	defer cancel()
	switch rec.via {
	case "portal":
		return n.conn.Object(n.portal, portalPath).CallWithContext(ctx, portalNotification+".RemoveNotification", 0, id).Err
	case "fdo":
		return n.conn.Object(n.fdo, fdoNotifyPath).CallWithContext(ctx, fdoNotify+".CloseNotification", dbus.FlagNoAutoStart, rec.fdo).Err
	}
	return nil
}

// Close stops listening for clicks. Notifications already up stay up.
func (n *Notifier) Close() {
	if n == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closed = true
	if n.conn != nil && n.ch != nil {
		n.conn.RemoveSignal(n.ch)
		close(n.ch)
		n.ch = nil
	}
}

func (n *Notifier) connLocked() (*dbus.Conn, error) {
	if n.conn != nil {
		return n.conn, nil
	}
	conn, err := n.dial()
	if err != nil || conn == nil {
		return nil, ErrNoNotifications
	}
	n.conn = conn
	// The clicks, from either service.
	_ = conn.AddMatchSignal(dbus.WithMatchInterface(fdoNotify), dbus.WithMatchMember("ActionInvoked"))
	_ = conn.AddMatchSignal(dbus.WithMatchInterface(fdoNotify), dbus.WithMatchMember("NotificationClosed"))
	_ = conn.AddMatchSignal(dbus.WithMatchInterface(portalNotification), dbus.WithMatchMember("ActionInvoked"))
	n.ch = make(chan *dbus.Signal, 16)
	conn.Signal(n.ch)
	go n.listen(n.ch)
	return conn, nil
}

// orderLocked is the services to try, best first (see Notifier).
func (n *Notifier) orderLocked(conn *dbus.Conn, want string) []string {
	switch want {
	case "portal":
		return []string{"portal"}
	case "fdo":
		return []string{"fdo"}
	}
	if sandboxed() {
		return []string{"portal", "fdo"}
	}
	if busNameOwned(conn, n.fdo) {
		return []string{"fdo", "portal"}
	}
	return []string{"portal", "fdo"}
}

func (n *Notifier) sendPortalLocked(conn *dbus.Conn, note DesktopNotification) error {
	timeout, ok := busServiceTimeout(conn, n.portal)
	if !ok {
		return ErrNoNotifications
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	call := conn.Object(n.portal, portalPath).CallWithContext(ctx, portalNotification+".AddNotification", 0, note.ID, portalNotificationOf(note))
	if call.Err != nil {
		return call.Err
	}
	n.rememberLocked(note, 0, "portal")
	return nil
}

func (n *Notifier) sendFdoLocked(conn *dbus.Conn, note DesktopNotification, prev *notifyRecord) error {
	// The server is used only while it runs: on some desktops its name is
	// activatable by a helper that waits for the shell to appear.
	if !busNameOwned(conn, n.fdo) {
		return ErrNoNotifications
	}
	var replaces uint32
	if prev != nil && prev.via == "fdo" {
		replaces = prev.fdo
	}
	actions, hints := fdoNotificationOf(note, n.opts.DesktopEntry)
	ctx, cancel := context.WithTimeout(context.Background(), busCallTimeout)
	defer cancel()
	var id uint32
	err := conn.Object(n.fdo, fdoNotifyPath).CallWithContext(ctx, fdoNotify+".Notify", dbus.FlagNoAutoStart,
		n.opts.AppName, replaces, note.IconName, note.Title, note.Body, actions, hints, int32(-1)).Store(&id)
	if err != nil {
		return err
	}
	if prev != nil && prev.via == "fdo" && prev.fdo != id {
		delete(n.byFdo, prev.fdo)
	}
	n.rememberLocked(note, id, "fdo")
	return nil
}

func (n *Notifier) rememberLocked(note DesktopNotification, fdoID uint32, via string) {
	if _, ok := n.live[note.ID]; !ok {
		n.order = append(n.order, note.ID)
	}
	n.live[note.ID] = &notifyRecord{n: note, fdo: fdoID, via: via}
	if via == "fdo" {
		n.byFdo[fdoID] = note.ID
	}
	for len(n.order) > notifyKeep {
		n.forgetLocked(n.order[0])
	}
}

func (n *Notifier) forgetLocked(id string) {
	if rec := n.live[id]; rec != nil && rec.via == "fdo" {
		delete(n.byFdo, rec.fdo)
	}
	delete(n.live, id)
	for i, o := range n.order {
		if o == id {
			n.order = append(n.order[:i], n.order[i+1:]...)
			break
		}
	}
}

// listen hands the desktop's clicks to the notifications they are for.
// Signals from the shared bus arrive for every watcher on it — and a
// server's ActionInvoked for every app's notification — so each is matched
// to one of ours or dropped.
func (n *Notifier) listen(ch chan *dbus.Signal) {
	for sig := range ch {
		if sig == nil {
			continue
		}
		var fn func()
		n.mu.Lock()
		switch sig.Name {
		case portalNotification + ".ActionInvoked":
			if len(sig.Body) >= 2 {
				id, _ := sig.Body[0].(string)
				name, _ := sig.Body[1].(string)
				if rec := n.live[id]; rec != nil && rec.via == "portal" {
					fn = n.activation(rec, name)
					n.forgetLocked(id)
				}
			}
		case fdoNotify + ".ActionInvoked":
			if len(sig.Body) >= 2 {
				fid, _ := sig.Body[0].(uint32)
				name, _ := sig.Body[1].(string)
				if id, ok := n.byFdo[fid]; ok {
					fn = n.activation(n.live[id], name)
				}
			}
		case fdoNotify + ".NotificationClosed":
			if len(sig.Body) >= 1 {
				fid, _ := sig.Body[0].(uint32)
				if id, ok := n.byFdo[fid]; ok {
					n.forgetLocked(id)
				}
			}
		}
		dispatch := n.opts.Dispatch
		n.mu.Unlock()
		if fn != nil {
			invokeStatus(dispatch, fn)
		}
	}
}

func (n *Notifier) activation(rec *notifyRecord, name string) func() {
	if rec == nil || rec.n.OnActivate == nil {
		return nil
	}
	action, ok := actionOf(rec.n, name)
	if !ok {
		return nil
	}
	on := rec.n.OnActivate
	return func() { on(action) }
}

// portalIcon is a serialised GIcon, the portal's icon: ("themed", names)
// or ("bytes", PNG).
type portalIcon struct {
	Kind  string
	Value dbus.Variant
}

// portalNotificationOf is AddNotification's a{sv}.
func portalNotificationOf(note DesktopNotification) map[string]dbus.Variant {
	v := map[string]dbus.Variant{
		"title":          dbus.MakeVariant(note.Title),
		"body":           dbus.MakeVariant(note.Body),
		"priority":       dbus.MakeVariant(note.Priority.portalPriority()),
		"default-action": dbus.MakeVariant("default"),
	}
	if note.IconName != "" {
		v["icon"] = dbus.MakeVariant(portalIcon{"themed", dbus.MakeVariant([]string{note.IconName})})
	} else if png := notifyPNG(note.Icon); png != nil {
		v["icon"] = dbus.MakeVariant(portalIcon{"bytes", dbus.MakeVariant(png)})
	}
	if len(note.Actions) > 0 {
		buttons := make([]map[string]dbus.Variant, 0, len(note.Actions))
		for i, a := range note.Actions {
			buttons = append(buttons, map[string]dbus.Variant{
				"label":  dbus.MakeVariant(a.Label),
				"action": dbus.MakeVariant(notifyAction(i)),
			})
		}
		v["buttons"] = dbus.MakeVariant(buttons)
	}
	return v
}

// fdoNotificationOf is Notify's actions and hints.
func fdoNotificationOf(note DesktopNotification, desktopEntry string) ([]string, map[string]dbus.Variant) {
	actions := []string{"default", "Open"}
	for i, a := range note.Actions {
		actions = append(actions, notifyAction(i), a.Label)
	}
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(note.Priority.fdoUrgency()),
	}
	if desktopEntry != "" {
		hints["desktop-entry"] = dbus.MakeVariant(desktopEntry)
	}
	if note.IconName == "" {
		if img, ok := notifyImage(note.Icon); ok {
			hints["image-data"] = dbus.MakeVariant(img)
		}
	}
	return actions, hints
}

// busNameOwned reports whether a service is running on the bus now.
func busNameOwned(conn *dbus.Conn, name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), busCallTimeout)
	defer cancel()
	var owned bool
	if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.NameHasOwner", 0, name).Store(&owned); err != nil {
		return false
	}
	return owned
}

// busActivatable reports whether D-Bus can start a service by name.
func busActivatable(conn *dbus.Conn, name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), busCallTimeout)
	defer cancel()
	var names []string
	if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListActivatableNames", 0).Store(&names); err != nil {
		return false
	}
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

// busServiceTimeout is how long a call to a service may take: a running
// one gets busCallTimeout, one D-Bus would start busStartTimeout; ok is
// false for one that is neither, which is not worth a call.
func busServiceTimeout(conn *dbus.Conn, name string) (time.Duration, bool) {
	if busNameOwned(conn, name) {
		return busCallTimeout, true
	}
	if busActivatable(conn, name) {
		return busStartTimeout, true
	}
	return 0, false
}
