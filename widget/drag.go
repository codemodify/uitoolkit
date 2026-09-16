package widget

import (
	"net/url"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// Drag is something being dragged out of a component: the types it
// offers, the bytes for one of them, the picture that follows the
// pointer, and what may be done with it. It is the mirror of
// [DropEvent] — one application's Drag is the next one's drop.
//
// The same Drag drives a drag to another application and a drag that
// never leaves this one: Payload rides along inside the process, so a
// list that reorders itself needs no encoding at all.
type Drag struct {
	// Types are the types offered, best first. A target takes the first
	// of them it knows ([PickDropMime]), so put the richest first and
	// text/plain last.
	Types []string
	// Data reads one of Types. It is called on the event-loop
	// goroutine when the target asks, which may be long after the drop,
	// so it must not block — read the bytes at the press if they are
	// not already in hand.
	Data func(mime string) ([]byte, bool)
	// Image follows the pointer; Hotspot is the point in it that sits
	// under the pointer. Without one the drag shows only a cursor.
	Image   *paintengine2d.Image
	Hotspot paintengine2d.Point
	// Actions are what the source allows — copying alone when zero.
	// Preferred is the one it would rather the target took.
	Actions   platform.DragAction
	Preferred platform.DragAction
	// Payload rides along inside the process: a drop on this
	// application's own windows gets it in [DropEvent.Payload],
	// untouched. Source is the component the drag started from, so a
	// target can tell "dropped on itself" from a real move.
	Payload any
	Source  Component
	// Local keeps the drag inside the process: the toolkit moves it
	// itself instead of handing it to the desktop. Reordering a list
	// wants this; dragging a file out to a file manager does not.
	Local bool
	// Done is told how the drag ended: the action the target performed,
	// or [platform.DragNone] when nothing took it — a cancelled drag
	// and a refused one look the same, as they should. A source whose
	// move was taken removes the original here.
	Done func(platform.DragAction)
}

// DragSource is implemented by components that start drags. DragAt is
// asked, once a press has moved past the drag threshold, what a drag from
// that point carries; nil means the press drags nothing (it was not on a
// row, or on a selection).
//
// A component that would rather decide for itself can call
// [StartDrag] from its own MouseMove instead.
type DragSource interface {
	DragAt(local paintengine2d.Point) *Drag
}

// DropActions is implemented by drop targets that do more than copy:
// DropActionFor is what this target would do with a drag whose source
// allows offered, and is what the source is shown while the drag is over
// it. A target that does not implement it copies.
type DropActions interface {
	DropActionFor(offered platform.DragAction) platform.DragAction
}

// DragHost is implemented by app.Window: start a drag from the press
// being handled now.
type DragHost interface {
	StartDrag(d *Drag) bool
}

// StartDrag begins a drag from c's window. It reports false when the
// window cannot start one (no press to start it from, or a drag already
// running).
func StartDrag(c Component, d *Drag) bool {
	if c == nil || d == nil {
		return false
	}
	if d.Source == nil {
		d.Source = c
	}
	h, ok := c.Host().(DragHost)
	if !ok {
		return false
	}
	return h.StartDrag(d)
}

// Allowed is what the drag lets a target do, which is copying when it
// says nothing.
func (d *Drag) Allowed() platform.DragAction {
	if d == nil || d.Actions == platform.DragNone {
		return platform.DragCopy
	}
	return d.Actions
}

// DropEvent is the drop this drag makes on a target in this same
// application: the decoded data, plus the payload and the source that
// only an in-process drop can carry.
func (d *Drag) DropEvent(pos paintengine2d.Point, mime string, data []byte, action platform.DragAction) DropEvent {
	e := NewDropEvent(pos, mime, data)
	e.Action = action
	if d != nil {
		e.Payload, e.Source = d.Payload, d.Source
	}
	return e
}

// Read is the drag's data as mime, "" for no such type.
func (d *Drag) Read(mime string) ([]byte, bool) {
	if d == nil || d.Data == nil || mime == "" {
		return nil, false
	}
	return d.Data(mime)
}

// Offers reports whether mime is one of the drag's types.
func (d *Drag) Offers(mime string) bool {
	if d == nil {
		return false
	}
	for _, t := range d.Types {
		if t == mime {
			return true
		}
	}
	return false
}

// NewDrag is a drag of data already in hand: types best first, and the
// bytes for each of them.
func NewDrag(types []string, data map[string][]byte) *Drag {
	return &Drag{
		Types: types,
		Data: func(mime string) ([]byte, bool) {
			b, ok := data[mime]
			return b, ok
		},
	}
}

// DragFiles is a drag of local files. It offers them as a uri-list —
// what every file manager and mail client takes — and as plain text, so
// a terminal or an editor gets the paths.
func DragFiles(paths ...string) *Drag {
	if len(paths) == 0 {
		return nil
	}
	uris := FileURIList(paths)
	text := strings.Join(paths, "\n")
	types := append([]string{"text/uri-list"}, textMimes...)
	data := map[string][]byte{"text/uri-list": []byte(uris)}
	for _, m := range textMimes {
		data[m] = []byte(text)
	}
	d := NewDrag(types, data)
	// Files are copied by default; a file manager asks for a move when
	// the user holds Shift, and offering it lets that work.
	d.Actions = platform.DragCopy | platform.DragMove | platform.DragLink
	d.Preferred = platform.DragCopy
	return d
}

// DragText is a drag of text, offered under every name text goes by so
// that old X11 clients (STRING, TEXT) take it too.
func DragText(s string) *Drag {
	if s == "" {
		return nil
	}
	data := map[string][]byte{}
	for _, m := range textMimes {
		data[m] = []byte(s)
	}
	d := NewDrag(append([]string(nil), textMimes...), data)
	d.Actions = platform.DragCopy | platform.DragMove
	d.Preferred = platform.DragCopy
	return d
}

// FileURIList encodes local paths as a text/uri-list body: file:// URIs,
// CRLF separated, as the format's RFC 2483 registration requires. A
// receiver that splits on newlines copes either way, but one that does
// not would otherwise keep a stray carriage return in the path.
func FileURIList(paths []string) string {
	var b strings.Builder
	for _, p := range paths {
		if p == "" {
			continue
		}
		u := url.URL{Scheme: "file", Path: p}
		b.WriteString(u.String())
		b.WriteString("\r\n")
	}
	return b.String()
}

// DragSnapshot paints c as it looks now into an image, to follow the
// pointer as the drag's picture. It returns nil for a component with no
// size yet.
func DragSnapshot(c Component) *paintengine2d.Image {
	if c == nil {
		return nil
	}
	b := c.Bounds()
	w, h := int(b.Dx()), int(b.Dy())
	if w < 1 || h < 1 {
		return nil
	}
	// A whole window's worth of pixels under the pointer hides what it
	// is being dropped on; clamp to something a user can see past.
	const maxSide = 512
	if w > maxSide {
		w = maxSide
	}
	if h > maxSide {
		h = maxSide
	}
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return nil
	}
	ctx.Clear(paintengine2d.Transparent)
	ctx.Translate(-b.Min.X, -b.Min.Y)
	PaintTree(c, ctx, nil)
	return img
}

// DragLabel draws a drag's picture for something with no natural
// snapshot: a rounded chip of accent-tinted text ("3 files"), at the
// window's scale. Its hotspot is a little inside the top-left corner, so
// the chip sits under the pointer without covering what is beneath it.
//
// scale is the display scale the picture is wanted at. A look is usually
// already baked at its window's scale, and its metrics are then in device
// pixels; only what the look does not already carry is applied here, so
// passing a window's own look and its own scale draws the chip once, and
// passing an unscaled look with a scale of 2 draws it twice as large.
func DragLabel(lk style.LookAndFeel, text string, scale float32) (*paintengine2d.Image, paintengine2d.Point) {
	if lk == nil || text == "" {
		return nil, paintengine2d.Point{}
	}
	if scale < 1 {
		scale = 1
	}
	k := scale / style.LookScale(lk)
	if k < 0.01 {
		k = 1
	}
	f := lk.Font()
	pad := style.Dip(lk, 8)
	w := f.Advance(text) + 2*pad
	h := f.Height() + style.Dip(lk, 6)
	iw, ih := int(w*k+0.5), int(h*k+0.5)
	if iw < 1 || ih < 1 {
		return nil, paintengine2d.Point{}
	}
	img := paintengine2d.NewImage(iw, ih)
	ctx := paintengine2d.NewContext(img)
	if ctx == nil {
		return nil, paintengine2d.Point{}
	}
	ctx.Clear(paintengine2d.Transparent)
	ctx.Scale(k, k)
	pal := lk.Palette()
	r := lk.Metrics().Radius
	box := paintengine2d.XYWH(0, 0, w, h)
	ctx.DrawRoundRect(box, r, r, paintengine2d.Fill(pal.Accent.WithAlpha(0.92)))
	ctx.DrawRoundRect(box.Inset(0.5), r, r, paintengine2d.StrokePaint(pal.TextOnAccent.WithAlpha(0.35), 1))
	f.Draw(ctx, text, paintengine2d.Pt(pad, (h-f.Height())*0.5), pal.TextOnAccent)
	// The hotspot is in the picture's own pixels, as every backend takes
	// it, so it follows the same k the picture was drawn at.
	hot := style.Dip(lk, 6) * k
	return img, paintengine2d.Pt(hot, hot)
}
