package platform

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codemodify/paintengine2d"
)

// DesktopNotification is a notification the desktop shows — a toast that
// goes to its notification centre — with buttons, and a way back into the
// app when the user clicks it. It is GNotification's shape (GTK) and
// KNotification's (KDE), and what the portal's AddNotification takes.
type DesktopNotification struct {
	// ID names the notification within the app: sending another with the
	// same ID replaces it where it stands (a "3 new messages" that
	// becomes "4 new messages"), and [Notifier.Withdraw] takes it down.
	// Empty gives each notification an ID of its own.
	ID    string
	Title string
	Body  string
	// IconName is a freedesktop icon theme name ("mail-unread"); Icon is
	// pixels, used where there is no name.
	IconName string
	Icon     *paintengine2d.Image
	// Actions are the notification's buttons, in order. Desktops show two
	// or three at most.
	Actions []NotificationAction
	// Priority is how insistent it is; the zero value is normal.
	Priority NotificationPriority
	// OnActivate runs when the user clicks the notification (action "")
	// or one of its buttons (that action's ID) — on the UI goroutine, when
	// the notifier has a Dispatch. The desktop takes the notification
	// down itself.
	OnActivate func(action string)
}

// NotificationAction is one button on a notification.
type NotificationAction struct {
	ID    string
	Label string
}

// NotificationPriority is how insistent a notification is.
type NotificationPriority uint8

const (
	PriorityNormal NotificationPriority = iota
	// PriorityLow may be kept out of sight (no popup, only the centre).
	PriorityLow
	// PriorityHigh is shown even in "do not disturb" on some desktops.
	PriorityHigh
	// PriorityUrgent stays until dismissed.
	PriorityUrgent
)

// portalPriority is the portal's name for p.
func (p NotificationPriority) portalPriority() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	}
	return "normal"
}

// fdoUrgency is org.freedesktop.Notifications' urgency hint for p (0 low,
// 1 normal, 2 critical).
func (p NotificationPriority) fdoUrgency() byte {
	switch p {
	case PriorityLow:
		return 0
	case PriorityUrgent:
		return 2
	}
	return 1
}

// NotifierOptions configure [NewNotifier].
type NotifierOptions struct {
	// AppName is how the desktop names the sender ("Mail").
	AppName string
	// DesktopEntry is the app's .desktop file name without the suffix
	// ("mailclientui"): desktops take the app's icon and its
	// notification settings from it.
	DesktopEntry string
	// Dispatch runs OnActivate on the UI goroutine (the app package sets
	// it to Application.Post). Nil runs it on the D-Bus goroutine.
	Dispatch func(func())
	// Stub sends nothing (headless apps, tests).
	Stub bool
}

// ErrNoNotifications: neither the notification portal nor a notification
// server answered.
var ErrNoNotifications = errors.New("platform: no desktop notification service")

// EnvNotify picks the notification backend: "portal", "fdo" (the
// org.freedesktop.Notifications server), or "off". Unset lets the notifier
// choose (see [Notifier]).
const EnvNotify = "UITK_NOTIFY"

// notifyWant is UITK_NOTIFY, normalised.
func notifyWant() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv(EnvNotify)))
}

// sandboxed reports whether the process runs in a Flatpak or Snap
// sandbox, where the portal is the only way out.
func sandboxed() bool {
	if _, err := os.Stat("/.flatpak-info"); err == nil {
		return true
	}
	return os.Getenv("SNAP") != "" || os.Getenv("container") == "flatpak"
}

// notifyAction is the action name a notification's button i is sent as,
// and the default (a click on the body) is "default". Names the app picks
// are never put on the bus: the portal only takes valid GAction names, and
// the notification server treats "default" specially.
func notifyAction(i int) string {
	return "button-" + strconv.Itoa(i)
}

// actionOf maps an action name that came back from the desktop to the
// app's action ID ("" for the body).
func actionOf(n DesktopNotification, name string) (string, bool) {
	if name == "default" || name == "" {
		return "", true
	}
	for i, a := range n.Actions {
		if name == notifyAction(i) {
			return a.ID, true
		}
	}
	return "", false
}

// notifyPNG encodes a notification's pixels as a PNG (straight alpha), the
// portal's "bytes" icon.
func notifyPNG(img *paintengine2d.Image) []byte {
	if img == nil || img.Width < 1 || img.Height < 1 {
		return nil
	}
	out := image.NewNRGBA(image.Rect(0, 0, img.Width, img.Height))
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, a := img.PremulAt(x, y)
			out.SetNRGBA(x, y, color.NRGBA{unpremul(r, a), unpremul(g, a), unpremul(b, a), a})
		}
	}
	var buf bytes.Buffer
	if png.Encode(&buf, out) != nil {
		return nil
	}
	return buf.Bytes()
}

// notifyImageData is the notification server's "image-data" hint: width,
// height, rowstride, has alpha, bits per sample, channels, and straight
// RGBA bytes.
type notifyImageData struct {
	W, H, Stride int32
	Alpha        bool
	Bits, Chans  int32
	Data         []byte
}

func notifyImage(img *paintengine2d.Image) (notifyImageData, bool) {
	if img == nil || img.Width < 1 || img.Height < 1 {
		return notifyImageData{}, false
	}
	pix := make([]byte, img.Width*img.Height*4)
	i := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b, a := img.PremulAt(x, y)
			pix[i], pix[i+1], pix[i+2], pix[i+3] = unpremul(r, a), unpremul(g, a), unpremul(b, a), a
			i += 4
		}
	}
	return notifyImageData{int32(img.Width), int32(img.Height), int32(img.Width * 4), true, 8, 4, pix}, true
}

// Timeouts for the desktop's services: a call to one that is running
// answers in milliseconds; one that D-Bus has to start first gets longer.
// Neither may hold an app up for more.
const (
	busCallTimeout  = 2 * time.Second
	busStartTimeout = 5 * time.Second
)
