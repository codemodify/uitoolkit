package style

import (
	"sync"

	"github.com/codemodify/paintengine2d"
)

// Whether a pack marks a *latched* push button, measured rather than
// listed — the same question [PrimaryFaceOf] asks about a default one,
// and the same answer where a pack does not.
//
// A latched push button is a search that is on, a filter that is
// applied, a panel that is showing: the control stays down until it is
// pressed again. Most eras draw one, and 53 of the 135 shipped packs
// draw a checked button exactly as they draw an ordinary one — so a
// button that told the truth about its state on the other 82 said
// nothing at all on these.
//
// Where a pack does not mark it, the button is drawn **pressed**
// instead. That is not a stand-in invented here: a toggle that stays
// held down is how Windows 3.1, Motif, CDE and OPEN LOOK all drew one,
// and it is a state every engine paints because an ordinary press needs
// it. The alternative — an accent ring or a tint of our own — would put
// a shape into eras that never had one, which is what the Amiga packs
// taught us in 0.22.0.

var checkedFace struct {
	mu sync.Mutex
	m  map[string]bool
}

// CheckedFaceOf reports whether l's engine draws a checked push button
// differently from an ordinary one.
//
// A caller that draws a latched button uses it to decide whether to add
// StatePressed itself; [LatchedState] does that and is what widgets
// call.
func CheckedFaceOf(l LookAndFeel) bool {
	if l == nil {
		return true
	}
	// Keyed on the pack, not on Name — which is the palette family, so
	// every pack in a build would share one answer.
	key := l.Name()
	if c, ok := l.(*Classic); ok {
		key = c.Pack()
	}
	checkedFace.mu.Lock()
	if got, ok := checkedFace.m[key]; ok {
		checkedFace.mu.Unlock()
		return got
	}
	checkedFace.mu.Unlock()

	got := measureCheckedFace(l)

	checkedFace.mu.Lock()
	if checkedFace.m == nil {
		checkedFace.m = map[string]bool{}
	}
	checkedFace.m[key] = got
	checkedFace.mu.Unlock()
	return got
}

// LatchedState is st with whatever it takes for this look to show that
// the button is latched.
//
// Where the engine marks a checked button, that is st unchanged. Where
// it does not, the button is drawn pressed, which every engine paints
// and which is how the eras that do not mark Checked drew a toggle in
// the first place.
func LatchedState(l LookAndFeel, st ControlState) ControlState {
	if !st.Checked() || CheckedFaceOf(l) {
		return st
	}
	return st | StatePressed
}

// measureCheckedFace draws the same button twice and asks whether the
// engine drew two different things.
//
// It calls the engine rather than [Classic.DrawButton] for the same
// reason [PrimaryFaceOf] does: measuring through the wrapper would ask
// the question about its own answer.
func measureCheckedFace(l LookAndFeel) bool {
	c, ok := l.(*Classic)
	if !ok {
		return true
	}
	const w, h = 72, 26
	draw := func(st ControlState) *paintengine2d.Image {
		img := paintengine2d.NewImage(w, h)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(l.Palette().Background))
		c.eng().DrawButton(c, ctx, paintengine2d.XYWH(2, 2, w-4, h-4), st, ButtonDraw{Label: "OK"})
		img.Touch()
		return img
	}
	plain := draw(StateNone)
	checked := draw(StateChecked | StateToggle)
	if plain.Width != checked.Width || plain.Height != checked.Height {
		return true
	}
	return visiblyDifferent(plain, checked) >= primaryFaceMinPixels
}
