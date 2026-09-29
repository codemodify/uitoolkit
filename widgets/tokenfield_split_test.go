package widgets

import (
	"reflect"
	"testing"

	"github.com/codemodify/uitoolkit/layout"
)

// A comma inside quotes is part of the value, not the end of it — the one
// piece of syntax every address list shares. Splitting on every comma
// made `"Lovelace, Ada" <ada@x>` into two recipients, both nonsense.
func TestTokenFieldKeepsAQuotedComma(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want []string
		tail string
	}{
		{
			"a quoted display name",
			`"Lovelace, Ada" <ada@x>, grace@y`,
			[]string{`"Lovelace, Ada" <ada@x>`},
			"grace@y",
		},
		{
			"two of them",
			`"Lovelace, Ada" <ada@x>, "Hopper, Grace" <grace@y>, z@w`,
			[]string{`"Lovelace, Ada" <ada@x>`, `"Hopper, Grace" <grace@y>`},
			"z@w",
		},
		{
			"an escaped quote inside the name",
			`"Ada \"The Countess\", Lovelace" <ada@x>, z@w`,
			[]string{`"Ada \"The Countess\", Lovelace" <ada@x>`},
			"z@w",
		},
		{
			"a semicolon inside quotes",
			`"Lovelace; Ada" <ada@x>; z@w`,
			[]string{`"Lovelace; Ada" <ada@x>`},
			"z@w",
		},
		{
			"no quotes at all still splits",
			"ada@x, grace@y, z@w",
			[]string{"ada@x", "grace@y"},
			"z@w",
		},
		{
			// A field that stopped accepting separators the moment a
			// quote was opened would stop working after a stray
			// keystroke.
			"an unclosed quote does not swallow the rest",
			`"Ada, ada@x, grace@y`,
			[]string{`"Ada`, "ada@x"},
			"grace@y",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := NewTokenField("", nil)
			f.splitInput(tc.in)
			if got := f.Tokens(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tokens %q, want %q", got, tc.want)
			}
			if got := f.Pending(); got != tc.tail {
				t.Errorf("editor %q, want %q", got, tc.tail)
			}
		})
	}
}

// A chip field wraps, so the width it would take with every chip on one
// line is not a width it wants. Reporting that made a Form size its field
// column from it — a Grid measures its columns unbounded — and a Flex
// track never goes below its content, so three addresses pushed a Write
// window's form past the window's edge.
func TestTokenFieldNaturalWidthDoesNotGrowWithChips(t *testing.T) {
	unbounded := layout.Constraints{MaxW: -1, MaxH: -1}

	f := NewTokenField("", nil)
	f.SetTokens([]string{"ada@example.com"})
	one := f.Measure(unbounded)

	f.SetTokens([]string{
		"ada@example.com", "grace.hopper@navy.mil", "katherine@nasa.gov",
		"margaret@mit.edu", "radia@sun.com", "barbara@ibm.com",
	})
	many := f.Measure(unbounded)

	if many.X != one.X {
		t.Errorf("six chips ask for %v wide, one chip asked for %v", many.X, one.X)
	}
	if many.Y <= one.Y {
		t.Errorf("six chips are %v tall and one chip was %v — it should have folded", many.Y, one.Y)
	}
}

// A chip is never split, so a chip wider than the preferred width sets
// the floor rather than being clipped.
func TestTokenFieldNaturalWidthFitsItsWidestChip(t *testing.T) {
	unbounded := layout.Constraints{MaxW: -1, MaxH: -1}
	f := NewTokenField("", nil)
	f.PreferredWidth = 40
	f.SetTokens([]string{"a-very-long-address@a-very-long-domain.example.com"})
	got := f.Measure(unbounded)

	chip := f.Chips()[0].Measure(unbounded)
	if got.X < chip.X {
		t.Errorf("the field asks for %v, narrower than its %v-wide chip", got.X, chip.X)
	}
}

// PreferredWidth is where it folds, not a width it is held to: given a
// width, it uses that one.
func TestTokenFieldUsesTheWidthItIsGiven(t *testing.T) {
	f := NewTokenField("", nil)
	f.PreferredWidth = 200
	f.SetTokens([]string{"ada@example.com", "grace@navy.mil"})
	got := f.Measure(layout.Constraints{MaxW: 600, MaxH: -1})
	if got.X != 600 {
		t.Errorf("measured %v wide inside 600, want 600", got.X)
	}
}
