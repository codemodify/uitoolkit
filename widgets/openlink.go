package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// openURI is the opener (a seam for tests, which must never start a
// browser).
var openURI = platform.OpenURI

// OpenLink opens uri — a web link, a mailto:, a file or a folder — with the
// desktop's application for it, as the window that hosts from asks
// (platform.OpenURI: the OpenURI portal, else xdg-open). It returns at
// once; the portal is talked to on a goroutine of its own. done, if not
// nil, hears how it went on the UI goroutine. Every link the toolkit
// shows opens through here.
func OpenLink(from widget.Component, uri string, done func(error)) {
	opts := platform.OpenURIOptions{}
	if from != nil {
		if p, ok := from.Host().(platform.PortalParenter); ok {
			// Named here, on the UI goroutine: a Wayland window is
			// exported for it (xdg-foreign) — the portal's "Open with…"
			// then opens as its child.
			opts.ParentWindow = p.PortalParent()
		}
	}
	var timers widget.Timers
	if from != nil {
		timers, _ = from.Host().(widget.Timers)
	}
	go func() {
		err := openURI(uri, opts)
		if done == nil {
			return
		}
		if timers == nil {
			done(err)
			return
		}
		timers.AfterFunc(0, func() { done(err) })
	}()
}

// LinkButton is a hyperlink: text in the look's link colour that opens a
// URI (GtkLinkButton, QLabel with openExternalLinks, KUrlLabel). Tab
// reaches it, Return or Space opens it, and its tooltip shows where it
// goes.
type LinkButton struct {
	widget.Base
	Text string
	URI  string
	// OnOpen, if set, runs instead of opening URI (an app that routes
	// its own links); OnError hears a link that could not be opened.
	OnOpen  func(uri string)
	OnError func(err error)
	// Visited draws the link in the muted colour once it has been opened.
	Visited bool

	hovered, pressed bool
}

// NewLinkButton is a link that reads text and opens uri.
func NewLinkButton(text, uri string) *LinkButton {
	l := &LinkButton{Text: text, URI: uri}
	l.Init(l)
	l.SetWantsFocus(true)
	l.SetFocusVisibleOnly(true)
	return l
}

// Open opens the link, as a click does.
func (l *LinkButton) Open() {
	l.Visited = true
	l.Invalidate()
	if l.OnOpen != nil {
		l.OnOpen(l.URI)
		return
	}
	OpenLink(l, l.URI, func(err error) {
		if err != nil && l.OnError != nil {
			l.OnError(err)
		}
	})
}

// Tooltip is where the link goes.
func (l *LinkButton) Tooltip() string { return l.URI }

func (l *LinkButton) Measure(c layout.Constraints) paintengine2d.Point {
	f := l.Look().Font()
	return c.Constrain(paintengine2d.Pt(f.Advance(l.Text)+4, f.Height()+4))
}

func (l *LinkButton) Arrange(r paintengine2d.Rect) { l.SetBounds(r) }

func (l *LinkButton) Paint(ctx *paintengine2d.Context) {
	lk := l.Look()
	pal := lk.Palette()
	f := lk.Font()
	b := l.LocalBounds()
	col := pal.Accent
	if l.Visited {
		col = pal.TextMuted
	}
	if !l.Enabled() {
		col = pal.TextMuted.WithAlpha(0.6)
	}
	text := l.Text
	if f.Advance(text) > b.Dx()-4 {
		text = f.Fit(text, b.Dx()-4)
	}
	y := b.Min.Y + (b.Dy()-f.Height())*0.5
	f.Draw(ctx, text, paintengine2d.Pt(b.Min.X+2, y), col)
	if l.hovered || l.pressed {
		// Underlined under the pointer, as links are on both desktops.
		w := f.Advance(text)
		ly := y + f.Height() - max(style.Dip(lk, 1), 1)*1.5
		ctx.DrawRect(paintengine2d.XYWH(b.Min.X+2, ly, w, max(style.Dip(lk, 1), 1)), paintengine2d.Fill(col))
	}
	if l.State()&style.StateFocused != 0 {
		lk.DrawFocusRing(ctx, b)
	}
}

func (l *LinkButton) MouseEnter() { l.hovered = true; l.Invalidate(); l.Base.MouseEnter() }
func (l *LinkButton) MouseExit() {
	l.hovered, l.pressed = false, false
	l.Invalidate()
	l.Base.MouseExit()
}

func (l *LinkButton) MousePress(e widget.MouseEvent) bool {
	if !l.Enabled() || e.Button != platform.ButtonLeft {
		return false
	}
	l.MarkPointerFocus()
	l.pressed = true
	l.Invalidate()
	return true
}

func (l *LinkButton) MouseRelease(e widget.MouseEvent) bool {
	was := l.pressed
	l.pressed = false
	l.Invalidate()
	if was && l.LocalBounds().Contains(e.Pos) && l.Enabled() {
		l.Open()
	}
	return true
}

func (l *LinkButton) KeyPress(e widget.KeyEvent) bool {
	if !l.Enabled() {
		return false
	}
	if e.Key == platform.KeyReturn || e.Key == platform.KeySpace {
		l.MarkKeyboardFocus()
		l.Open()
		return true
	}
	return false
}

func (l *LinkButton) Describe(n *a11y.Node) {
	n.Role = a11y.RoleLink
	nameOr(n, l.Text)
	n.Description = l.URI
}
