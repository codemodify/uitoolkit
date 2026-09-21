package skinart

import "github.com/codemodify/uitoolkit/internal/players/minim/panel"

// The fixed layouts Minim's two panel skins state.
//
// The rects are panel's (internal/players/minim/panel), the same numbers
// the faces were drawn round, so the slots the manifest states and the holes
// in the art are one set of numbers. The player reads nothing but the
// manifest: it binds its controls to the slot names and asks the skin where
// each one is.

// minimLayouts are a panel face's three layouts, with every key's art bound
// from the sprites the sheet actually has.
func minimLayouts(f *panel.Face, sh *Sheet) []LayoutSpec {
	have := map[string]bool{}
	for _, c := range sh.Cells {
		have[c.Name] = true
	}
	var out []LayoutSpec
	for _, lay := range f.Layouts() {
		ls := LayoutSpec{Name: lay.Name, W: lay.W, H: lay.H, Art: lay.Face}
		for _, sl := range lay.Slots {
			ss := SlotSpec{Name: sl.Name, At: [4]int(sl.R)}
			if sl.Key != "" {
				ss.Art = keyStates(sl.Key, have)
			}
			ls.Slots = append(ls.Slots, ss)
		}
		out = append(out, ls)
	}
	return out
}

// keyStates binds a key's sprites to the control states: "key.play" at
// rest, ".hover" under the pointer, ".down" held, ".on" switched on, and
// their pairs.
//
// A key drawn with no ".on" face is not a toggle, and its switched-on state
// is its resting face, stated outright — the part fallback would otherwise
// show a pause key that is on (the player is paused) as held down, which is
// not what the panel's art means by it.
func keyStates(key string, have map[string]bool) [][2]string {
	pick := func(names ...string) string {
		for _, n := range names {
			if have[n] {
				return n
			}
		}
		return ""
	}
	var out [][2]string
	add := func(state, sprite string) {
		if sprite != "" {
			out = append(out, [2]string{state, sprite})
		}
	}
	add("normal", pick(key))
	add("hover", pick(key+".hover"))
	add("pressed", pick(key+".down"))
	if have[key+".on"] {
		add("checked", key+".on")
		add("checkedHover", pick(key+".on.hover"))
		add("checkedPressed", pick(key+".on.down", key+".down"))
	} else {
		add("checked", key)
		add("checkedHover", pick(key+".hover"))
		add("checkedPressed", pick(key+".down"))
	}
	return out
}
