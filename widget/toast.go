package widget

// ToastHost is implemented by app.Window: a layer of passing notes over
// the window's content, under its popups.
//
// A note is not a dialog, a popup or a tooltip. A dialog waits to be
// answered; a popup is a surface of its own with the keyboard in it; a
// tooltip belongs to the pointer and goes as soon as it moves. A note
// says what just happened — Copied, Unlocked, what went wrong — for a few
// seconds, over whatever the window was showing, and nothing has to be
// clicked. It is the window's only other place for that, and the other
// one (a status bar) stays for ever and takes a row of the layout.
type ToastHost interface {
	Host
	// SetToastLayer puts c on the layer, or clears it with nil.
	SetToastLayer(c Component)
	// ToastLayer is what is on it, or nil.
	ToastLayer() Component
}

// ToastLayerOf is the host's toast layer, or nil where the host has none
// (a widget not in a window, a host that is not a window).
func ToastLayerOf(from Component) Component {
	if h, ok := hostOf(from).(ToastHost); ok {
		return h.ToastLayer()
	}
	return nil
}

// ShowToastLayer puts layer on the host's toast layer and reports whether
// there was a host to put it on.
func ShowToastLayer(from, layer Component) bool {
	h, ok := hostOf(from).(ToastHost)
	if !ok || layer == nil {
		return false
	}
	layer.SetHost(h)
	h.SetToastLayer(layer)
	return true
}

// DismissToastLayer clears the host's toast layer.
func DismissToastLayer(from Component) {
	if h, ok := hostOf(from).(ToastHost); ok {
		h.SetToastLayer(nil)
	}
}

// Announcer is implemented by app.Window: something worth saying that no
// control holds and no focus move would speak.
//
// Assistive technology hears about what the keyboard is on, and about
// what changes under it. A note that appears in the corner is neither:
// nothing moved the focus, so a screen reader is told nothing at all
// unless the window says it. This is that — the toolkit's word for
// ARIA's polite live region, which is how the same thing is said on the
// web.
type Announcer interface {
	// Announce says text to assistive technology, politely: after
	// whatever is being spoken now, never cutting it off.
	Announce(text string)
}

// Announce says text through from's host, if the host can say anything.
func Announce(from Component, text string) {
	if text == "" {
		return
	}
	if a, ok := hostOf(from).(Announcer); ok {
		a.Announce(text)
	}
}

// hostOf is from's host, nil-safe.
func hostOf(c Component) Host {
	if c == nil {
		return nil
	}
	return c.Host()
}
