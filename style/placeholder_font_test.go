package style

import "testing"

// A placeholder is the field's own text, greyed. It is not a different size.
//
// mutedFor kept the supplied face only when its family was the look's
// monospace one, and answered with the look's cached muted face for anything
// else. So a field drawn with an explicit 11-point proportional face typed at
// 11 points and showed its placeholder at the look's 16: a placeholder that
// clips or sits wrong in a field sized for the smaller text, and that reads as
// a deliberate style rather than a dropped argument.

func TestMutingKeepsTheFacesMetrics(t *testing.T) {
	lk := DarkLook()
	for _, tc := range []struct {
		what   string
		family string
	}{
		{"a proportional face", lk.Font().Family},
		{"the monospace face", lk.MonoFamily()},
	} {
		small := BakeFamily(tc.family, WeightRegular, 11, lk.Palette().Text)
		muted := lk.mutedFor(small)
		if muted.Size != small.Size {
			t.Errorf("%s: muted at %.0f points, the face is %.0f", tc.what, muted.Size, small.Size)
		}
		if muted.Family != small.Family {
			t.Errorf("%s: muted in %q, the face is %q", tc.what, muted.Family, small.Family)
		}
		if muted.Height() != small.Height() {
			t.Errorf("%s: muted is %.2f tall, the face is %.2f", tc.what, muted.Height(), small.Height())
		}
		// And it is muted, which is the one thing it should change.
		if muted.Color == small.Color {
			t.Errorf("%s: muting did not change the colour", tc.what)
		}
		if muted.Color != lk.Palette().TextMuted {
			t.Errorf("%s: muted ink is %v, want the palette's", tc.what, muted.Color)
		}
	}
}

// A placeholder measures as the typed text does, which is what a field is
// sized against.
func TestAPlaceholderMeasuresLikeTheTypedText(t *testing.T) {
	lk := DarkLook()
	const word = "Search"
	small := BakeFamily(lk.Font().Family, WeightRegular, 11, lk.Palette().Text)
	typed := small.Advance(word)
	place := lk.mutedFor(small).Advance(word)
	if typed != place {
		t.Errorf("typed %q is %.1f wide and its placeholder %.1f: the field is sized for one of them", word, typed, place)
	}
	// The look's own default is a different size, so the test above is not
	// comparing a face with itself.
	if lk.mutedFor(nil).Size == small.Size {
		t.Skip("the look's muted face is already 11 points; the case cannot be shown")
	}
}

// No face at all is still the look's own muted face.
func TestMutingNothingIsTheLooksOwnMutedFace(t *testing.T) {
	lk := DarkLook()
	if got := lk.mutedFor(nil); got != lk.muted {
		t.Errorf("mutedFor(nil) = %v, want the look's cached muted face", got)
	}
}
