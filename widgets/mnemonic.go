package widgets

import "github.com/codemodify/uitoolkit/platform"

// ParseMnemonic strips a leading '&' marker from label text.
// "&File" → ("File", KeyF, 0). No marker → (s, KeyUnknown, -1).
func ParseMnemonic(s string) (label string, key platform.Key, index int) {
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '&' && i+1 < len(runes) && runes[i+1] != '&' {
			k := platform.LetterKey(runes[i+1])
			plain := string(append(append([]rune{}, runes[:i]...), runes[i+1:]...))
			return plain, k, i
		}
	}
	return s, platform.KeyUnknown, -1
}
