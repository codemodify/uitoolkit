package style

import (
	"strconv"

	"github.com/codemodify/paintengine2d"
)

// A small mark on the corner of an icon.
//
// An icon says what a control does; a badge says what this one does
// *differently*. A padlock with a plus is "unlock with something else",
// the vault's own mark with a plus is "a new one", and a count on an
// inbox is how many are waiting. A password manager drew all three
// itself, in a tool button, because the toolkit had no way to say it.

// BadgeMark is what a badge draws.
type BadgeMark uint8

const (
	// BadgeNone is no badge, and is the zero value: a plain icon.
	BadgeNone BadgeMark = iota
	// BadgePlus is the "and one more" of New, Add and Unlock with…
	BadgePlus
	// BadgeDot says only that there is something, without saying how
	// much: an unread mark, a change nobody has looked at.
	BadgeDot
	// BadgeCount is a number, from [IconBadge.Count].
	BadgeCount
)

// IconBadge is a mark to draw over the corner of an icon.
type IconBadge struct {
	Mark  BadgeMark
	Count int
}

// Badged is the badge for a count: none at all for zero, because a
// badge that says "0" is a badge that should not be there.
func Badged(n int) IconBadge {
	if n <= 0 {
		return IconBadge{}
	}
	return IconBadge{Mark: BadgeCount, Count: n}
}

// Empty reports a badge with nothing to draw.
func (b IconBadge) Empty() bool {
	return b.Mark == BadgeNone || (b.Mark == BadgeCount && b.Count <= 0)
}

// DrawIconBadge draws b's badge over the corner of box.
//
// The box is the caller's: a tool bar passes the button, an
// application drawing its own icon passes the icon. Which corner
// depends on what the badge says — a mark that modifies the action
// sits at the bottom, where it reads as part of it, and a count sits
// at the top right, where every notification badge has sat since the
// first one.
func DrawIconBadge(ctx *paintengine2d.Context, box paintengine2d.Rect, badge IconBadge, lk LookAndFeel) {
	if ctx == nil || lk == nil || badge.Empty() || box.Empty() {
		return
	}
	p := lk.Palette()
	side := min(box.Dx(), box.Dy())
	if side < 8 {
		return
	}
	d := max(side*0.46, 8)
	label := ""
	if badge.Mark == BadgeCount {
		label = strconv.Itoa(badge.Count)
		if badge.Count > 99 {
			label = "99+"
		}
	}

	// A count grows sideways and the others are round.
	w := d
	if n := len(label); n > 1 {
		w = d + float32(n-1)*d*0.42
	}
	x := box.Max.X - w
	y := box.Max.Y - d
	if badge.Mark == BadgeCount {
		y = box.Min.Y
	}
	r := paintengine2d.XYWH(x, y, w, d)

	// A ring in the surface's colour, so the badge reads as being on
	// top of the icon rather than part of it.
	ctx.DrawRoundRect(r.Inset(-1.5), d*0.5+1.5, d*0.5+1.5, paintengine2d.Fill(p.Surface))
	ctx.DrawRoundRect(r, d*0.5, d*0.5, paintengine2d.Fill(p.Accent))

	ink := p.TextOnAccent
	switch badge.Mark {
	case BadgePlus:
		arm := d * 0.28
		cx, cy := r.Min.X+w*0.5, r.Min.Y+d*0.5
		t := max(d*0.12, 1)
		ctx.DrawRect(paintengine2d.XYWH(cx-arm, cy-t*0.5, arm*2, t), paintengine2d.Fill(ink))
		ctx.DrawRect(paintengine2d.XYWH(cx-t*0.5, cy-arm, t, arm*2), paintengine2d.Fill(ink))
	case BadgeDot:
		// Nothing inside it: the badge is the dot.
	case BadgeCount:
		// The on-accent face, which is the one meant to be read
		// against this fill.
		f := lk.OnAccentFont()
		if f == nil {
			f = lk.Font()
		}
		if f != nil {
			tw := f.Advance(label)
			f.Draw(ctx, label,
				paintengine2d.Pt(r.Min.X+(w-tw)*0.5, r.Min.Y+(d-f.Height())*0.5), ink)
		}
	}
}
