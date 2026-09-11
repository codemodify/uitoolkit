package mail

import (
	"strings"

	"github.com/codemodify/paintengine2d"
)

// DefaultTags are Thunderbird-like colored keywords.
func DefaultTags() []Tag {
	return []Tag{
		{Name: "Important", Color: "#c0392b"},
		{Name: "Work", Color: "#d35400"},
		{Name: "Personal", Color: "#27ae60"},
		{Name: "To Do", Color: "#2980b9"},
		{Name: "Later", Color: "#8e44ad"},
	}
}

func cloneTags(in []Tag) []Tag {
	out := make([]Tag, len(in))
	copy(out, in)
	return out
}

func upsertTag(list []Tag, t Tag) []Tag {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return list
	}
	if t.Color == "" {
		t.Color = "#7f8c8d"
	}
	for i, x := range list {
		if strings.EqualFold(x.Name, t.Name) {
			list[i] = t
			return list
		}
	}
	return append(list, t)
}

func tagByName(list []Tag, name string) (Tag, bool) {
	for _, t := range list {
		if strings.EqualFold(t.Name, name) {
			return t, true
		}
	}
	return Tag{}, false
}

// ParseHexColor accepts #rgb or #rrggbb.
func ParseHexColor(s string) paintengine2d.Color {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return paintengine2d.RGB(0.5, 0.5, 0.55)
	}
	n := func(a, b byte) float32 {
		return float32(unhex(a)*16+unhex(b)) / 255
	}
	return paintengine2d.RGB(n(s[0], s[1]), n(s[2], s[3]), n(s[4], s[5]))
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c - 'a' + 10)
	case c >= 'A' && c <= 'F':
		return int(c - 'A' + 10)
	}
	return 0
}
