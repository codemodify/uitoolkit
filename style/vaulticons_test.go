package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// vaultIcons are the seven a vault is made of: a padlock alone cannot
// say which of these a row is.
var vaultIcons = []ToolIcon{
	IconLockOpen, IconKey, IconPassword, IconTerminal, IconDrive, IconSafe, IconDice,
}

// **Every one of them draws, in both drawn sets.** A typed id that the
// drawn sets have no vector for comes out as the missing-icon mark —
// which is what the five PNG packs fall back to as well, through
// Drawable — so this is the test that keeps all four answers honest.
func TestTheVaultIconsAreDrawnInBothSets(t *testing.T) {
	for _, set := range []IconSetName{IconSetClassic, IconSetSharp} {
		for _, icon := range vaultIcons {
			img := paintedIcon(t, icon, set, 24)
			if !anyInk(img) {
				t.Errorf("%v in set %q drew nothing", ToolIconName(icon), set)
			}
		}
	}
}

// And each is its own mark. A set where two of them came out the same
// picture would be one where the missing-icon placeholder is standing in
// for both, which is exactly the failure this list was added to end.
func TestTheVaultIconsAreTheirOwnMarks(t *testing.T) {
	// The padlock is in the comparison: an open padlock that drew the
	// shut one would be the worst of these mistakes.
	icons := append([]ToolIcon{IconLock, IconMore}, vaultIcons...)
	for _, set := range []IconSetName{IconSetClassic, IconSetSharp} {
		shots := map[ToolIcon]*paintengine2d.Image{}
		for _, icon := range icons {
			shots[icon] = paintedIcon(t, icon, set, 24)
		}
		for i, a := range icons {
			for _, b := range icons[i+1:] {
				if diffPixels(shots[a], shots[b]) == 0 {
					t.Errorf("in set %q, %s and %s are the same picture",
						set, ToolIconName(a), ToolIconName(b))
				}
			}
		}
	}
}

// The open padlock carries the packs' own name for it, so their art
// answers for it rather than the drawn fallback.
func TestTheOpenPadlockUsesTheShippedStem(t *testing.T) {
	if got := StemOf(IconLockOpen); got != "unlock" {
		t.Errorf("IconLockOpen resolves to %q, want the packs' own \"unlock\"", got)
	}
	if got, ok := IconByStem("unlock"); !ok || got != IconLockOpen {
		t.Errorf("IconByStem(\"unlock\") = %v %v, want IconLockOpen", got, ok)
	}
	// And every one of the seven has a stem of its own, which is what a
	// person's copied icon set is searched by.
	seen := map[string]ToolIcon{}
	for _, icon := range vaultIcons {
		stem := StemOf(icon)
		if stem == "" {
			t.Errorf("%v has no stem", icon)
			continue
		}
		if prev, dup := seen[stem]; dup {
			t.Errorf("%v and %v share the stem %q", icon, prev, stem)
		}
		seen[stem] = icon
	}
}

// Each set draws them in its own hand. A drawn set with no vector of its
// own for an icon falls back to the Classic one — which is better than an
// empty box, and is why the test above cannot tell the two apart — so
// this is the one that says the square era got square marks.
func TestTheVaultIconsAreSquareInTheSharpSet(t *testing.T) {
	for _, icon := range vaultIcons {
		classic := paintedIcon(t, icon, IconSetClassic, 24)
		sharp := paintedIcon(t, icon, IconSetSharp, 24)
		if diffPixels(classic, sharp) == 0 {
			t.Errorf("%s is the same picture in both sets: the sharp set has no mark of its own and is falling back",
				ToolIconName(icon))
		}
	}
}
