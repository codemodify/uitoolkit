package minim

import (
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/players"
	"github.com/codemodify/uitoolkit/internal/players/minim/panel"
	"github.com/codemodify/uitoolkit/style"
)

// The faces.
//
// Minim wears three skins, and two of them are not dressings for its widgets
// but *panels*: pictures of the whole window with holes where the keys go
// (internal/players/minim/panel says where). A skin cannot lay out a panel —
// the format re-skins controls and says nothing about where they sit — so
// the player carries one layout per panel and picks it by the pack it is
// wearing. The widget tree is the same one in every face: switching skins
// moves and repaints the controls, hides the few one face has and another
// does not, and rebuilds nothing, so the focus, the tab order and the
// accessibility tree survive a switch the way they survive Lantern's.
//
// A pack that is not one of Minim's panels — the "minim" skin itself, and
// every other one of the hundred and twenty-odd — gets the widget face: the
// ordinary layout, painted by whatever the look paints.

// face is which of the layouts a look gets.
type face uint8

const (
	faceWidgets face = iota // the "minim" skin, and any pack that is not a panel
	faceClassic             // "minim-classic"
	faceSilver              // "minim-silver"
)

// faceOf is the face a look gets. It asks for the panel's own background as
// well as the pack's name, so a skin that has lost its art — or a stranger's
// skin that borrowed the name — is laid out as widgets rather than as a
// panel with nothing painted under it.
func faceOf(lk style.LookAndFeel) face {
	var f face
	switch style.LookAppearance(lk).Name {
	case SkinClassic:
		f = faceClassic
	case SkinSilver:
		f = faceSilver
	default:
		return faceWidgets
	}
	if _, _, ok := style.SkinSpriteSize(lk, "main.face"); !ok {
		return faceWidgets
	}
	return f
}

// geometry is a panel face's layout, nil for the widget face.
func (f face) geometry() *panel.Face {
	switch f {
	case faceClassic:
		return panel.Classic
	case faceSilver:
		return panel.Silver
	}
	return nil
}

// panelled reports whether the face is a panel.
func (f face) panelled() bool { return f != faceWidgets }

// ---- the ink a face sets text in -------------------------------------------------

// ink is the colours a panel face draws in where the app sets text or draws a
// line itself rather than painting a sprite: the playlist's rows, the
// equaliser's curve, the analyser's bars. They belong to the face — the art
// the rows sit on is the face's — and so they live beside its layout rather
// than in the look's palette, which a menu the player opens also reads.
type ink struct {
	text, row, current, selected, selectedText, info paintengine2d.Color
	curve, peak                                      paintengine2d.Color
	ramp                                             []paintengine2d.Color
}

func (f face) ink() *ink {
	switch f {
	case faceClassic:
		return &classicInk
	case faceSilver:
		return &silverInk
	}
	return nil
}

var classicInk = ink{
	text:         rgb(0x00, 0xe0, 0x00),
	row:          rgb(0x00, 0xd0, 0x00),
	current:      rgb(0xff, 0xff, 0xff),
	selected:     rgb(0x00, 0x00, 0xb8),
	selectedText: rgb(0xff, 0xff, 0xff),
	info:         rgb(0x00, 0xe0, 0x00),
	curve:        rgb(0xe4, 0xd1, 0x2f),
	peak:         rgb(0x9a, 0x9a, 0xa6),
	// Green at the floor, then yellow, then orange at the very top: the
	// climb an analyser of the era drew, one colour per row.
	ramp: []paintengine2d.Color{
		rgb(0x10, 0x9c, 0x10), rgb(0x14, 0xa8, 0x14), rgb(0x18, 0xb4, 0x18), rgb(0x1c, 0xc0, 0x1c),
		rgb(0x20, 0xcc, 0x20), rgb(0x28, 0xd4, 0x20), rgb(0x40, 0xdc, 0x20), rgb(0x60, 0xe0, 0x20),
		rgb(0x88, 0xe0, 0x20), rgb(0xb0, 0xdc, 0x20), rgb(0xd0, 0xd4, 0x20), rgb(0xe0, 0xc0, 0x20),
		rgb(0xe8, 0xa8, 0x20), rgb(0xec, 0x90, 0x20), rgb(0xee, 0x78, 0x1c), rgb(0xf0, 0x60, 0x18),
	},
}

var silverInk = ink{
	text:         rgb(0xe6, 0xee, 0xff),
	row:          rgb(0x9c, 0xb4, 0xe6),
	current:      rgb(0xff, 0xff, 0xff),
	selected:     rgb(0x66, 0x88, 0xcc),
	selectedText: rgb(0xff, 0xff, 0xff),
	info:         rgb(0xe6, 0xee, 0xff),
	curve:        rgb(0x6c, 0x8c, 0xc8),
	peak:         rgb(0xe6, 0xee, 0xff),
	// A dot-matrix bar: lit rows with a dark row between each, so a bar
	// reads as a stack of dots rather than as a solid block.
	ramp: func() []paintengine2d.Color {
		out := make([]paintengine2d.Color, 0, 16)
		for i := 0; i < 8; i++ {
			out = append(out, rgb(0xd8, 0xe4, 0xfa), rgb(0x2a, 0x48, 0x88))
		}
		return out
	}(),
}

func rgb(r, g, b uint8) paintengine2d.Color {
	return paintengine2d.RGB(float32(r)/255, float32(g)/255, float32(b)/255)
}

// ---- painting a panel --------------------------------------------------------------

// panelMargin is the empty design pixel round every panel sprite (see
// internal/skinart, pixMargin): a sprite is drawn into its box grown by it.
const panelMargin = 1

// box is a panel rect as a rect in a component's own coordinates at the
// look's scale, given the component's origin in the panel.
func box(lk style.LookAndFeel, at panel.R, origin paintengine2d.Point) paintengine2d.Rect {
	d := func(v int) float32 { return style.Dip(lk, float32(v)) }
	return paintengine2d.XYWH(d(at.X())-origin.X, d(at.Y())-origin.Y, d(at.W()), d(at.H()))
}

// panelSize is a panel face's content box for a window h design pixels
// tall: the whole of it, since the art is drawn for exactly that box.
func panelSize(lk style.LookAndFeel, g *panel.Face, h int) paintengine2d.Point {
	return paintengine2d.Pt(style.Dip(lk, float32(g.ContentW())), style.Dip(lk, float32(g.ContentH(h))))
}

// art paints a panel sprite into a rect of the component's own. It reports
// false when the look has no such sprite, so the caller can fall back.
func art(lk style.LookAndFeel, ctx *paintengine2d.Context, r paintengine2d.Rect, name string) bool {
	m := style.Dip(lk, panelMargin)
	return style.DrawSkinSprite(lk, ctx, r.Inset(-m), name, paintengine2d.Color{})
}

// keyArt is a key's sprite for a state: "key.play", "key.play.down",
// "key.eq.on", "key.eq.on.down" — falling back along the same order a skin's
// part states do, so a key drawn in two states still works in four.
func keyArt(lk style.LookAndFeel, ctx *paintengine2d.Context, r paintengine2d.Rect, name string, st style.ControlState) bool {
	var tries []string
	down := st.Pressed()
	switch {
	case st.Checked() && down:
		tries = []string{name + ".on.down", name + ".on", name + ".down", name}
	case st.Checked():
		tries = []string{name + ".on", name}
	case down:
		tries = []string{name + ".down", name}
	default:
		tries = []string{name}
	}
	for _, t := range tries {
		if art(lk, ctx, r, t) {
			return true
		}
	}
	return false
}

// paintAsKey makes a glyph button paint as the named key in a panel face,
// and as itself in the widget face.
func paintAsKey(b *players.GlyphButton, name string) {
	b.Painter = func(ctx *paintengine2d.Context, r paintengine2d.Rect, st style.ControlState) bool {
		lk := b.Look()
		if !faceOf(lk).panelled() {
			return false
		}
		return keyArt(lk, ctx, r, name, st)
	}
}

// paintAsThumb makes a fader paint as a thumb sprite riding a groove the
// face's background already has printed on it. The thumb is the sprite's own
// size, and the fader's Travel is set to half of it along its length, so the
// pointer and the picture agree about where the ends are.
func paintAsThumb(f *players.Fader, name string) {
	f.Painter = func(ctx *paintengine2d.Context, r paintengine2d.Rect, t float32, st style.ControlState) bool {
		lk := f.Look()
		sw, sh, ok := style.SkinSpriteSize(lk, name)
		if !faceOf(lk).panelled() || !ok {
			// Travel is read by the pointer, which always comes after a
			// paint, so the paint is where the face's answer is kept.
			f.Travel = 0
			return false
		}
		w, h := sw-2*panelMargin, sh-2*panelMargin
		tw, th := style.Dip(lk, w), style.Dip(lk, h)
		var thumb paintengine2d.Rect
		if f.Horizontal {
			f.Travel = w / 2
			x := r.Min.X + t*(r.Dx()-tw)
			thumb = paintengine2d.XYWH(snapDev(x), r.Min.Y+snapDev((r.Dy()-th)/2), tw, th)
		} else {
			f.Travel = h / 2
			y := r.Min.Y + (1-t)*(r.Dy()-th)
			thumb = paintengine2d.XYWH(r.Min.X+snapDev((r.Dx()-tw)/2), snapDev(y), tw, th)
		}
		sprite := name
		if st.Pressed() {
			if _, _, ok := style.SkinSpriteSize(lk, name+".down"); ok {
				sprite += ".down"
			}
		}
		return art(lk, ctx, thumb, sprite)
	}
}

// snapDev rounds to a whole device pixel, so a thumb that has moved by a
// fraction lands on the grid instead of between two pixels.
func snapDev(v float32) float32 {
	if v < 0 {
		return -snapDev(-v)
	}
	return float32(int(v + 0.5))
}

// ---- the display's own alphabet ------------------------------------------------------

// pixText sets a line in the skin's bitmap capitals ("font.<hex>" sprites)
// with its top left at (x, y), in ink, and returns how wide it was. A rune
// the face does not draw is set as a question mark, so a gap shows.
//
// The glyphs are the skin's and the line is the app's: this is the display
// printing the track, which a skin cannot know and an app cannot draw.
func pixText(lk style.LookAndFeel, ctx *paintengine2d.Context, x, y float32, s string, col paintengine2d.Color) float32 {
	d := func(v float32) float32 { return style.Dip(lk, v) }
	x0 := x
	for _, r := range strings.ToUpper(s) {
		name := glyphName(r)
		w, h, ok := style.SkinSpriteSize(lk, name)
		if !ok {
			name = glyphName('?')
			if w, h, ok = style.SkinSpriteSize(lk, name); !ok {
				continue
			}
		}
		if ctx != nil {
			m := d(panelMargin)
			style.DrawSkinSprite(lk, ctx, paintengine2d.XYWH(x-m, y-m, d(w), d(h)), name, col)
		}
		x += d(w - 2*panelMargin + 1)
	}
	return max(x-x0-d(1), 0)
}

// pixWidth is how wide pixText would set s.
func pixWidth(lk style.LookAndFeel, s string) float32 {
	return pixText(lk, nil, 0, 0, s, paintengine2d.Color{})
}

func glyphName(r rune) string {
	const digits = "0123456789abcdef"
	var b []byte
	for v := uint32(r); ; v >>= 4 {
		b = append([]byte{digits[v&15]}, b...)
		if v < 16 {
			break
		}
	}
	return "font." + string(b)
}

// pixScroll sets a line that is wider than its box sliding back and forth
// inside it, on the same rests and pace as the widget face's title
// (players.ScrollK), clipped to the box.
func pixScroll(lk style.LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, s string, col paintengine2d.Color, p *Player) {
	w := pixWidth(lk, s)
	over := w - b.Dx()
	x := b.Min.X
	if over > 0 {
		x -= snapDev(over * float32(players.ScrollK(lk, over, p.Transport.Pos)))
	}
	ctx.Save()
	ctx.ClipRect(b)
	pixText(lk, ctx, x, b.Min.Y, s, col)
	ctx.Restore()
}

// ledClock paints the clock in the skin's segmented digits.
func ledClock(lk style.LookAndFeel, ctx *paintengine2d.Context, m *panel.Main, origin paintengine2d.Point, clock string) {
	// "m:ss" or "mm:ss" to four digits: the era's clock showed minutes and
	// seconds in two digits each, with a leading zero.
	digits := strings.ReplaceAll(clock, ":", "")
	for len(digits) < 4 {
		digits = "0" + digits
	}
	digits = digits[len(digits)-4:]
	for i, r := range digits {
		art(lk, ctx, box(lk, m.Digits[i], origin), "led."+string(r))
	}
	art(lk, ctx, box(lk, m.Colon, origin), "led.colon")
}
