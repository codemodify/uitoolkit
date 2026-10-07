package style

import "github.com/codemodify/paintengine2d"

// A push button's words and its mark.
//
// [Engine.DrawButton] used to take the label alone, and that is why a button
// with an icon was about 64 device pixels wider than one without: the engine
// draws the label, every engine centres it in the box it is handed, and every
// engine decorates it its own way — Clearlooks embosses it, Win95 shadows it,
// Aqua draws gel text. A widget that wanted a mark at the leading edge could
// not move the label without taking the drawing away from the engine and
// losing all of that, so it reserved a strip at *each* end instead and let the
// centred label land between them. Half of that width bought nothing but
// symmetry.
//
// [Engine.DrawToolButton] had the answer all along: it is handed the icon and
// lays the two out itself, which is how a tool button can be tight. This is
// the same, for push buttons — and a struct rather than another parameter,
// because what goes on a button has grown before (a trailing chevron, a count)
// and will again. [MenuRow] is the same shape for a menu row.
type ButtonDraw struct {
	// Label is the button's words, "" for a button that is only a mark.
	Label string
	// Icon is a mark at the leading edge, [IconNone] for none.
	Icon ToolIcon
}

// ButtonIconWidth is what a leading mark adds to a push button's width: one
// strip and the gap after it, in host pixels at this look's scale.
//
// One strip, not two. A caller measuring a button asks this rather than
// working it out from [ToolButtonChromeFor], so that an era which wants a
// different strip — Win95's is 20 where Adwaita's is 24 — answers for itself.
func ButtonIconWidth(lk LookAndFeel, h float32) float32 {
	_, side, gap := ToolButtonChromeFor(lk, h)
	return side + gap
}

// ButtonIconBox is where a push button's leading mark goes inside b, or the
// empty rect when there is none.
func ButtonIconBox(lk LookAndFeel, b paintengine2d.Rect, d ButtonDraw) paintengine2d.Rect {
	if d.Icon == IconNone || b.Empty() {
		return paintengine2d.Rect{}
	}
	pad, side, _ := ToolButtonChromeFor(lk, b.Dy())
	if side > b.Dx() {
		return paintengine2d.Rect{}
	}
	if d.Label == "" {
		// Only a mark: it sits in the middle, as a tool button's does.
		return paintengine2d.XYWH(b.Min.X+(b.Dx()-side)*0.5, b.Min.Y+(b.Dy()-side)*0.5, side, side)
	}
	return paintengine2d.XYWH(b.Min.X+pad, b.Min.Y+(b.Dy()-side)*0.5, side, side)
}

// ButtonLabelBox is the box a push button's label goes in: the button, less the
// strip a leading mark takes at the start.
//
// An engine centres its label in *this* rather than in the whole button, which
// is the whole of the difference — the label stays centred in the space it has
// and the button is no wider than one strip, instead of being padded at both
// ends to keep it centred in the button. With no mark it is the button.
func ButtonLabelBox(lk LookAndFeel, b paintengine2d.Rect, d ButtonDraw) paintengine2d.Rect {
	ib := ButtonIconBox(lk, b, d)
	if ib.Empty() || d.Label == "" {
		return b
	}
	_, _, gap := ToolButtonChromeFor(lk, b.Dy())
	lb := b
	lb.Min.X = ib.Max.X + gap
	if lb.Min.X > lb.Max.X {
		lb.Min.X = lb.Max.X
	}
	return lb
}
