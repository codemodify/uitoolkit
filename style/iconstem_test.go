package style

import "testing"

// The shipped PNG vocabulary is wider than the typed ids, and it used to
// be unreachable: an application could see "calendar", "sync" and
// "users" sitting in all five premiere packs with no way to name one.
func TestIconByStemReachesTheWholeVocabulary(t *testing.T) {
	for _, stem := range ShippedIconStems() {
		icon, ok := IconByStem(stem)
		if !ok {
			t.Errorf("IconByStem(%q) found nothing, but the packs ship it", stem)
			continue
		}
		if icon == IconNone {
			t.Errorf("IconByStem(%q) gave IconNone", stem)
		}
		if got := StemOf(icon); got != stem {
			t.Errorf("IconByStem(%q) resolves back to %q", stem, got)
		}
	}
}

// A stem that has a typed id gives that id, so it behaves identically in
// every set rather than becoming a second, stem-only way to say the same
// thing.
func TestIconByStemPrefersTheTypedID(t *testing.T) {
	for _, tc := range []struct {
		stem string
		want ToolIcon
	}{
		{"save", IconSave},
		{"trash", IconTrash},
		{"print", IconPrint},
		{"bell-off", IconMute},
		{"external-link", IconExternalLink},
	} {
		if got, ok := IconByStem(tc.stem); !ok || got != tc.want {
			t.Errorf("IconByStem(%q) = %v %v, want %v", tc.stem, got, ok, tc.want)
		}
		if got := StemOf(tc.want); got != tc.stem {
			t.Errorf("StemOf(%v) = %q, want %q", tc.want, got, tc.stem)
		}
	}
}

// A stem nothing ships says so, rather than handing back a placeholder a
// caller cannot tell from a real answer.
func TestIconByStemRefusesWhatIsNotThere(t *testing.T) {
	for _, stem := range []string{"", "banana", "SAVE", "save.png", "no-such-thing"} {
		if icon, ok := IconByStem(stem); ok {
			t.Errorf("IconByStem(%q) = %v, want not found", stem, icon)
		}
	}
}

// A stem-only icon resolves to a file in a file set. It falls back to the
// missing-icon mark in a drawn set, which is the honest answer — but it
// must never resolve to nothing at all.
func TestStemOnlyIconsHaveAFileName(t *testing.T) {
	for _, stem := range []string{"calendar", "clock", "users", "sync", "inbox", "compose"} {
		icon, ok := IconByStem(stem)
		if !ok {
			t.Fatalf("%q is not shipped?", stem)
		}
		if _, typed := ToolIconByName(stem); typed {
			continue // this one has an id of its own
		}
		if got := StemOf(icon); got != stem {
			t.Errorf("stem-only %q resolves to %q", stem, got)
		}
	}
}

// The two geometric marks are drawn by the toolkit in every set,
// including the file sets, because none of the five packs ships either
// one and a filled star has no house style to match.
func TestFilledStarAndDotDrawInEverySet(t *testing.T) {
	for _, set := range []IconSetName{IconSetClassic, IconSetSharp, "lucide", "heroicons"} {
		for _, icon := range []ToolIcon{IconStarFilled, IconDot} {
			img := paintedIcon(t, icon, set, 24)
			if !anyInk(img) {
				t.Errorf("%v in set %q drew nothing", icon, set)
			}
		}
	}
}
