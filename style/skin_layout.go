package style

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/codemodify/paintengine2d"
)

// Fixed layouts: the half of a skin that says where things go.
//
// A skin re-skins ordinary widgets at ordinary layout, and for a form or an
// editor that is the whole story. A player is not a form. WinAmp's 275×116
// and VLC's <Layout> were panels — a picture with holes in it, and a control
// in each hole — and a panel's proportions are its design: a flex box would
// give them away. This is that half of the format, with the rule that makes
// it safe kept intact:
//
//	A skin places. It never adds.
//
// A layout is a design size and a table of named slots, each a rect in the
// layout's design pixels. The *app* binds its own components to slot names
// (widget.Slots); the skin says where each slot is and, if it likes, what
// the control in it looks like in each state. A slot nobody binds is a
// place with nothing in it. A control whose slot the skin leaves out is the
// app's to place — it is reported back, never hidden by the skin, because a
// skin that could take a control out of the tab order would be deciding
// what the app can do. The tab order stays component order, the
// accessibility tree stays the app's tree, and every control in a slot is
// the same widget it was before it was placed.
//
// Slot rects use the silhouette's vocabulary (SkinShapeRect): a rect pinned
// to the top left, pinned to the right or the bottom (fromRight,
// fromBottom), or stretching with the box (stretchX, stretchY), so a layout
// can be drawn for a panel that is a fixed size — Minim's — or one that
// grows.
//
//	"layouts": {
//	  "minim.strip": {
//	    "size": [273, 101],
//	    "art": "main.face",
//	    "slots": {
//	      "play":    { "at": [38, 70, 21, 21],
//	                   "art": { "normal": "key.play", "pressed": "key.play.down" } },
//	      "display": { "at": [4, 3, 265, 54] }
//	    }
//	  }
//	}

// SkinLayout is one fixed panel: its size and its slots.
type SkinLayout struct {
	Name string
	// W and H are the size the layout was drawn for, in design pixels.
	W, H float32
	// Art is painted behind the slots, when the layout has any.
	Art *SkinPart
	// Slots are the named places, and order their names sorted.
	Slots map[string]*SkinSlot
	order []string
}

// SkinSlot is one named place in a layout.
type SkinSlot struct {
	Name string
	// Rect is where the slot is, in the layout's design pixels.
	Rect SkinShapeRect
	// Art is what the control in the slot looks like, per state, resolved
	// along the same fallback chain a part's states are; nil for a slot
	// whose control paints itself.
	Art *SkinPart
}

// SlotNames lists the layout's slots, sorted.
func (lay *SkinLayout) SlotNames() []string {
	if lay == nil {
		return nil
	}
	return append([]string(nil), lay.order...)
}

type skinLayoutJSON struct {
	Size  []float32                  `json:"size"`
	Art   json.RawMessage            `json:"art,omitempty"`
	Slots map[string]json.RawMessage `json:"slots,omitempty"`
}

type skinSlotJSON struct {
	At         []float32       `json:"at"`
	StretchX   bool            `json:"stretchX,omitempty"`
	StretchY   bool            `json:"stretchY,omitempty"`
	FromRight  bool            `json:"fromRight,omitempty"`
	FromBottom bool            `json:"fromBottom,omitempty"`
	Art        json.RawMessage `json:"art,omitempty"`
}

func (sk *Skin) loadLayouts(m map[string]json.RawMessage) error {
	for name, raw := range m {
		key := joinKey("layouts", name)
		if strings.TrimSpace(name) == "" {
			return skinErr(key, "a layout needs a name: the one an app binds its controls under")
		}
		var doc skinLayoutJSON
		if err := decodeSkinJSON(key, raw, &doc); err != nil {
			return err
		}
		if len(doc.Size) != 2 || doc.Size[0] <= 0 || doc.Size[1] <= 0 {
			return skinErr(joinKey(key, "size"), "needs 2 positive numbers: [width, height] in design pixels")
		}
		lay := &SkinLayout{Name: name, W: doc.Size[0], H: doc.Size[1], Slots: map[string]*SkinSlot{}}
		if len(doc.Art) > 0 {
			art, err := sk.slotArt(joinKey(key, "art"), doc.Art, name)
			if err != nil {
				return err
			}
			lay.Art = art
		}
		for sn, sraw := range doc.Slots {
			skey := joinKey(joinKey(key, "slots"), sn)
			if strings.TrimSpace(sn) == "" {
				return skinErr(skey, "a slot needs a name")
			}
			var sd skinSlotJSON
			if err := decodeSkinJSON(skey, sraw, &sd); err != nil {
				return err
			}
			r, err := parseSkinRect(skey, skinShapeRectJSON{
				At: sd.At, StretchX: sd.StretchX, StretchY: sd.StretchY,
				FromRight: sd.FromRight, FromBottom: sd.FromBottom,
			})
			if err != nil {
				return err
			}
			slot := &SkinSlot{Name: sn, Rect: r}
			if len(sd.Art) > 0 {
				art, err := sk.slotArt(joinKey(skey, "art"), sd.Art, name+"."+sn)
				if err != nil {
					return err
				}
				slot.Art = art
			}
			lay.Slots[sn] = slot
			lay.order = append(lay.order, sn)
		}
		sort.Strings(lay.order)
		sk.Layouts[name] = lay
	}
	return nil
}

// slotArt reads a slot's (or a layout's) art: one sprite name for every
// state, or an object of state names to sprite names that resolves along
// the part fallback chain — so a key drawn at rest and held down still has
// a face when it is hovered or switched on.
func (sk *Skin) slotArt(key string, raw json.RawMessage, name string) (*SkinPart, error) {
	p := &SkinPart{Name: name, States: map[string]*SkinSprite{}}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		sp, ok := sk.Sprites[one]
		if !ok {
			return nil, skinErr(key, "no sprite %q", one)
		}
		p.States["normal"] = sp
		return p, nil
	}
	var states map[string]string
	if err := decodeSkinJSON(key, raw, &states); err != nil {
		return nil, skinErr(key, "must be a sprite name, or an object of states to sprite names")
	}
	for st, spName := range states {
		sk2 := joinKey(key, st)
		if !skinStateSet[st] {
			return nil, skinErr(sk2, "%q is not a control state (one of %s)", st, strings.Join(skinStateNames, ", "))
		}
		sp, ok := sk.Sprites[spName]
		if !ok {
			return nil, skinErr(sk2, "no sprite %q", spName)
		}
		p.States[st] = sp
	}
	if p.States["normal"] == nil {
		return nil, skinErr(key, `needs "normal": every other state falls back to it`)
	}
	return p, nil
}

// ---- what an app asks ------------------------------------------------------

// skinLayoutOf is the look's skin and the named layout, or nil for a look
// that is not a skin or a skin without that layout.
func skinLayoutOf(lk LookAndFeel, name string) (*Classic, *Skin, *SkinLayout) {
	l, sk := lookSkin(lk)
	if sk == nil {
		return nil, nil, nil
	}
	lay := sk.Layouts[name]
	if lay == nil {
		return nil, nil, nil
	}
	return l, sk, lay
}

// SkinLayoutOf is the look's skin's layout by that name, and whether it has
// one. An app asks this to know whether to lay a panel out by the skin's
// slots or its own way — the answer for every look that is not a skin, and
// for a skin drawn for some other app, is false.
func SkinLayoutOf(lk LookAndFeel, name string) (*SkinLayout, bool) {
	_, _, lay := skinLayoutOf(lk, name)
	return lay, lay != nil
}

// SkinLayoutSize is the size the named layout was drawn for, in device
// pixels at the look's scale — what a panel laid out by it measures.
func SkinLayoutSize(lk LookAndFeel, name string) (paintengine2d.Point, bool) {
	l, sk, lay := skinLayoutOf(lk, name)
	if lay == nil {
		return paintengine2d.Point{}, false
	}
	s := l.Scale() / sk.Design.Scale
	return paintengine2d.Pt(lay.W*s, lay.H*s), true
}

// SkinSlotRect is where a slot of the named layout is inside box, in the
// same coordinates as box, at the look's scale. False when the look's skin
// has no such layout or no such slot.
//
// The rect is design pixels times the scale and is not rounded: a panel's
// pieces are drawn on one grid, and rounding each slot on its own would
// move neighbours by different fractions.
func SkinSlotRect(lk LookAndFeel, layout, slot string, box paintengine2d.Rect) (paintengine2d.Rect, bool) {
	l, sk, lay := skinLayoutOf(lk, layout)
	if lay == nil {
		return paintengine2d.Rect{}, false
	}
	sl := lay.Slots[slot]
	if sl == nil {
		return paintengine2d.Rect{}, false
	}
	return sl.Rect.resolve(box, l.Scale()/sk.Design.Scale), true
}

// DrawSkinLayout paints the named layout's own art into box — the picture
// the slots are holes in. False when there is none.
//
// A box within a pixel of the layout's own size is taken to be that size:
// a panel's box is whole pixels, its design size times 1.75 is not, and
// the art is drawn at its design size so every slot lands on its hole.
func DrawSkinLayout(lk LookAndFeel, ctx *paintengine2d.Context, box paintengine2d.Rect, layout string) bool {
	l, sk, lay := skinLayoutOf(lk, layout)
	if lay == nil || ctx == nil {
		return false
	}
	s := l.Scale() / sk.Design.Scale
	w, h := lay.W*s, lay.H*s
	if dw, dh := box.Dx()-w, box.Dy()-h; dw > -1 && dw < 1 && dh > -1 && dh < 1 {
		box = paintengine2d.XYWH(box.Min.X, box.Min.Y, w, h)
	}
	return sk.panelDraw(ctx, box, lay.Art.art("normal"), l.Scale(), paintengine2d.Color{})
}

// DrawSkinSlot paints the art the skin gives a slot's control in state st,
// into b — the control's own box. False when the slot has no art, so the
// control paints itself.
func DrawSkinSlot(lk LookAndFeel, ctx *paintengine2d.Context, b paintengine2d.Rect, layout, slot string, st ControlState) bool {
	l, sk, lay := skinLayoutOf(lk, layout)
	if lay == nil || ctx == nil {
		return false
	}
	sl := lay.Slots[slot]
	if sl == nil {
		return false
	}
	return sk.panelDraw(ctx, b, sl.Art.art(skinStateName(st)), l.Scale(), paintengine2d.Color{})
}

// SkinSlotShape is the silhouette of a slot's resting art drawn at at in a
// control whose box is size (device pixels, origin at the box's top left),
// or nil for the whole box: the slot has no art, the art fills its box, or
// it could not be drawn. It is what makes a round key painted from a slot
// take the pointer only on its ink.
func SkinSlotShape(lk LookAndFeel, size paintengine2d.Point, at paintengine2d.Rect, layout, slot string) *Silhouette {
	l, sk, lay := skinLayoutOf(lk, layout)
	if lay == nil {
		return nil
	}
	sl := lay.Slots[slot]
	if sl == nil {
		return nil
	}
	sp := sl.Art.art("normal")
	if sp == nil {
		return nil
	}
	return sk.shapeOf(l, sp, size, at, true)
}
