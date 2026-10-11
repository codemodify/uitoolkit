package style

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// caseSkinDoc binds the four parts whose label the engine paints to a text
// role, with "case" set or not.
func caseSkinDoc(upper bool) string {
	role := `{ "color": "#ffffff" }`
	if upper {
		role = `{ "color": "#ffffff", "case": "upper" }`
	}
	return `{
  "skin": 1, "label": "Case probe", "family": "dark", "base": "dark",
  "sheets": { "chrome": { "1x": "art/sheet.png", "2x": "art/sheet2.png" } },
  "sprites": { "face": { "sheet": "chrome", "at": [0, 0, 12, 12], "slice": [4, 4, 4, 4] } },
  "text": { "caps": ` + role + ` },
  "parts": {
    "button": { "states": { "normal": "face" }, "text": "caps" },
    "tool":   { "states": { "normal": "face" }, "text": "caps" },
    "tab":    { "states": { "normal": "face" }, "text": "caps" },
    "combo":  { "states": { "normal": "face" }, "text": "caps" },
    "field":  { "states": { "normal": "face" }, "text": "caps" },
    "row":    { "states": { "normal": "face" }, "text": "caps" }
  }
}`
}

func caseSkinLook(t *testing.T, name string, upper bool) *Classic {
	t.Helper()
	installSkin(t, name, caseSkinDoc(upper))
	pack, ok := LoadTheme(name)
	if !ok {
		t.Skipf("the %q pack is not in this build", name)
	}
	lk := pack.Look()
	if id := lk.Engine().ID(); id != skinEngineID {
		t.Fatalf("the look paints with %q, not the skin engine", id)
	}
	return lk
}

// **A skin's capitals reach the labels it paints.** A text role's
// "case": "upper" was applied where the frame paints the window's title
// and nowhere else, so a skin could ask for capitals and get them on one
// string in the whole window: a button, a tab or a tool item bound to the
// same role drew its label as written.
func TestASkinsCapitalsReachTheLabelsItPaints(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := caseSkinLook(t, "caseprobe", true)
	for _, role := range []Role{RoleButton, RoleTool, RoleTab, RoleCombo} {
		if got := lk.ControlLabel(role, "Save as"); got != "SAVE AS" {
			t.Errorf("role %v paints %q, want capitals", role, got)
		}
	}
	// What the control shows of the person's own content is theirs: a
	// field's text and a row's data stay as they are, and a skin does not
	// get to rewrite the document to suit its art.
	for _, role := range []Role{RoleField, RoleRow, RoleMenu} {
		if got := lk.ControlLabel(role, "Save as"); got != "Save as" {
			t.Errorf("role %v paints %q, want the words as written", role, got)
		}
	}
}

// Without the case the labels are as written, which is every skin that
// does not ask.
func TestASkinWithoutACasePaintsTheWordsAsWritten(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := caseSkinLook(t, "caseplain", false)
	for _, role := range []Role{RoleButton, RoleTool, RoleTab, RoleCombo} {
		if got := lk.ControlLabel(role, "Save as"); got != "Save as" {
			t.Errorf("role %v paints %q, want the words as written", role, got)
		}
	}
}

// And it is the painting that changes, in ink on the button: the capitals
// are in the picture, and the label the application set is untouched.
func TestTheCapitalsAreInThePicture(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	upper := caseSkinLook(t, "casepic", true)
	plain := caseSkinLook(t, "casepicplain", false)

	shot := func(lk *Classic, draw func(*paintengine2d.Context, *Classic)) *paintengine2d.Image {
		img := paintengine2d.NewImage(140, 32)
		ctx := paintengine2d.NewContext(img)
		ctx.Clear(paintengine2d.Color{})
		draw(ctx, lk)
		img.Touch()
		return img
	}
	box := paintengine2d.XYWH(2, 2, 136, 28)
	for _, tc := range []struct {
		name string
		draw func(*paintengine2d.Context, *Classic)
	}{
		{"a button", func(ctx *paintengine2d.Context, lk *Classic) {
			lk.DrawButton(ctx, box, StateNone, ButtonDraw{Label: "Save as"})
		}},
		{"a tool button", func(ctx *paintengine2d.Context, lk *Classic) {
			lk.DrawToolButton(ctx, box, StateNone, "Save as", IconNone)
		}},
		{"a tab", func(ctx *paintengine2d.Context, lk *Classic) {
			lk.DrawTab(ctx, box, StateNone, "Save as", true)
		}},
		{"a closed combo box", func(ctx *paintengine2d.Context, lk *Classic) {
			lk.DrawComboBox(ctx, box, StateNone, "Save as", false)
		}},
	} {
		if n := diffPixels(shot(upper, tc.draw), shot(plain, tc.draw)); n < 20 {
			t.Errorf("%s drew the same %d pixels either way: the case is not reaching it", tc.name, n)
		}
	}
	// A field's text is the person's, and it is identical.
	field := func(ctx *paintengine2d.Context, lk *Classic) {
		lk.DrawTextField(ctx, box, StateNone, "Save as", "", 0, 0, 0, false, 0, nil)
	}
	if n := diffPixels(shot(upper, field), shot(plain, field)); n != 0 {
		t.Errorf("a field's own text changed in %d pixels", n)
	}
}

// The width a widget reserves is the width of what will be painted.
// Measuring the words as written and painting them in capitals squeezes
// the label by the difference, which is how a "painting only" change
// shows up as clipped text.
func TestALabelInCapitalsIsMeasuredInCapitals(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	lk := caseSkinLook(t, "casemeasure", true)
	const label = "Save as"
	plain := ControlFontOf(lk, RoleButton).Advance(label)
	caps := ControlFontOf(lk, RoleButton).Advance(strings.ToUpper(label))
	if caps <= plain {
		t.Skip("capitals are not wider in this face")
	}
	if got := ControlAdvanceOf(lk, RoleButton, label); got != caps {
		t.Errorf("a button's label measures %g, want the %g its capitals need", got, caps)
	}
}
