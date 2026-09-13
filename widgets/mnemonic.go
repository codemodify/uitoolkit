package widgets

import "github.com/codemodify/uitoolkit/platform"

// ParseMnemonic strips the '&' marker from label text and reports the
// accelerator. "&File" → ("File", KeyF, 0). No marker → (s, KeyUnknown, -1).
// "&&" is an escaped literal ampersand: "Save && Exit" → ("Save & Exit",
// KeyUnknown, -1). index is a rune offset into the returned label, so an
// escaped ampersand before the marker does not shift the underline.
func ParseMnemonic(s string) (label string, key platform.Key, index int) {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	key = platform.KeyUnknown
	index = -1
	for i := 0; i < len(runes); i++ {
		if runes[i] != '&' {
			out = append(out, runes[i])
			continue
		}
		if i+1 >= len(runes) {
			// Trailing lone '&' is literal text, not a marker.
			out = append(out, '&')
			continue
		}
		if runes[i+1] == '&' {
			out = append(out, '&')
			i++
			continue
		}
		if index < 0 {
			index = len(out)
			key = platform.LetterKey(runes[i+1])
		}
		// Drop the marker; the next iteration appends the marked rune.
	}
	return string(out), key, index
}
