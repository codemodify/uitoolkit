package style

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// What a skin author wants to know about a skin that loads.
//
// The loader refuses what is wrong, by the key it came from (SkinError).
// This is the other half: a skin that loads and paints, but that its author
// would want to hear about before shipping — a control that has no art for
// being pressed, a sprite nothing uses, a 2× sheet that is not really twice
// the size. None of these is an error, because a terse skin is supported:
// art for "normal" alone gives a control every state. They are what
// `uitk-skin lint` prints.

// SkinWarning is one thing worth knowing about a skin, keyed like a
// SkinError by the manifest key it is about.
type SkinWarning struct {
	Skin string
	Key  string
	Msg  string
}

func (w SkinWarning) String() string {
	var b strings.Builder
	if w.Skin != "" {
		b.WriteString(w.Skin)
		b.WriteString(": ")
	}
	b.WriteString(SkinFile)
	b.WriteString(": ")
	if w.Key != "" {
		b.WriteString(w.Key)
		b.WriteString(": ")
	}
	b.WriteString("warning: ")
	b.WriteString(w.Msg)
	return b.String()
}

// skinLintStates are the states worth drawing for each part a person
// interacts with: without them the control still works, but it does not
// show that it was pressed, switched on, or out of reach.
var skinLintStates = map[string][]string{
	"button":         {"hover", "pressed", "disabled"},
	"tool":           {"hover", "pressed", "checked"},
	"field":          {"focus", "disabled"},
	"check":          {"checked", "disabled"},
	"tab":            {"checked", "hover"},
	"combo":          {"hover", "pressed", "disabled"},
	"thumb":          {"hover", "pressed"},
	"slider.thumb":   {"hover", "pressed"},
	"switch.track":   {"checked"},
	"caption":        {"inactive"},
	"caption.button": {"hover", "pressed"},
}

// LintSkin is everything worth knowing about a skin that loaded, sorted by
// key: states a control falls back from, sprites nothing binds, and art too
// small for 2×. A skin that says nothing worth a warning gives none.
func LintSkin(sk *Skin) []SkinWarning {
	if sk == nil {
		return nil
	}
	var out []SkinWarning
	warn := func(key, format string, args ...any) {
		out = append(out, SkinWarning{Skin: sk.Name, Key: key, Msg: fmt.Sprintf(format, args...)})
	}

	// States a person would notice missing.
	for _, name := range sortedKeys(sk.Parts) {
		lintStates(warn, "parts."+name+".states", sk.Parts[name], skinLintStates[name])
	}
	if w := sk.Window; w != nil {
		for _, role := range sortedKeys(w.Variants) {
			v := w.Variants[role]
			for _, name := range sortedKeys(v.Parts) {
				lintStates(warn, "window.variants."+role+".parts."+name+".states", v.Parts[name], skinLintStates[name])
			}
		}
	}
	for _, ln := range sortedKeys(sk.Layouts) {
		lay := sk.Layouts[ln]
		for _, sn := range lay.order {
			// A slot's art is a key's when it has more than one state; one
			// sprite for every state is a lamp or a picture, and asks for
			// nothing more.
			if a := lay.Slots[sn].Art; a != nil && len(a.States) > 1 && a.States["pressed"] == nil {
				warn("layouts."+ln+".slots."+sn+".art", `no "pressed" art: a press shows the resting face`)
			}
		}
	}

	// Sprites nothing in the manifest binds. They are allowed — an app
	// paints its own by name (DrawSkinSprite) — so they are reported as one
	// line per family rather than one per sprite.
	used := sk.usedSprites()
	families := map[string]int{}
	for name := range sk.Sprites {
		if used[sk.Sprites[name]] {
			continue
		}
		fam := name
		if i := strings.IndexByte(name, '.'); i > 0 {
			fam = name[:i]
		}
		families[fam+"*"]++
	}
	if len(families) > 0 {
		var parts []string
		total := 0
		for _, f := range sortedKeys(families) {
			parts = append(parts, fmt.Sprintf("%s (%d)", f, families[f]))
			total += families[f]
		}
		warn("sprites", "%d that no part, window or layout binds, which only an app that names them will paint: %s",
			total, strings.Join(parts, ", "))
	}

	// Art too small for 2×.
	for _, name := range sortedKeys(sk.Sheets) {
		sk.lintSheet(warn, "sheets."+name, sk.Sheets[name])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// lintStates warns about each of want a part draws no art of its own for.
func lintStates(warn func(string, string, ...any), key string, p *SkinPart, want []string) {
	if p == nil || p.States["normal"] == nil {
		return
	}
	for _, st := range want {
		if p.States[st] != nil {
			continue
		}
		from := "normal"
		for _, alt := range skinStateFallback[st] {
			if p.States[alt] != nil {
				from = alt
				break
			}
		}
		warn(key, "no %q art: it shows the %q face", st, from)
	}
}

// usedSprites is every sprite something in the manifest binds.
func (sk *Skin) usedSprites() map[*SkinSprite]bool {
	used := map[*SkinSprite]bool{}
	part := func(p *SkinPart) {
		if p == nil {
			return
		}
		for _, sp := range p.States {
			used[sp] = true
		}
	}
	for _, p := range sk.Parts {
		part(p)
	}
	if w := sk.Window; w != nil {
		used[w.ShapeArt] = true
		for _, p := range w.Parts {
			part(p)
		}
		for _, v := range w.Variants {
			used[v.ShapeArt] = true
			for _, p := range v.Parts {
				part(p)
			}
		}
	}
	for _, lay := range sk.Layouts {
		part(lay.Art)
		for _, sl := range lay.Slots {
			part(sl.Art)
		}
	}
	return used
}

// lintSheet checks a sheet has art at 2× and that it is really twice the
// size of its 1× art — and, for a pixelated sheet, that it is the 1× art
// with every pixel doubled, since that is what nearest sampling at 2
// assumes.
func (sk *Skin) lintSheet(warn func(string, string, ...any), key string, sh *SkinSheet) {
	if !sh.Covers(2) {
		warn(key, "no art at 2x or above: on a 2x display every sprite on it is drawn larger than it was made, and soft")
		return
	}
	one, ok1 := sh.Files[1]
	two, ok2 := sh.Files[2]
	if !ok1 || !ok2 {
		return
	}
	a, b := sk.sheetImage(one), sk.sheetImage(two)
	if a == nil || b == nil {
		return
	}
	if b.Width < 2*a.Width || b.Height < 2*a.Height {
		warn(joinKey(key, "2x"), "%s is %d×%d, smaller than twice %s's %d×%d: sprites cut from it at 2x fall short",
			filepath.Base(two), b.Width, b.Height, filepath.Base(one), a.Width, a.Height)
		return
	}
	if !sh.Pixelated {
		return
	}
	for y := 0; y < a.Height; y++ {
		for x := 0; x < a.Width; x++ {
			r, g, bl, al := a.PremulAt(x, y)
			for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				r2, g2, b2, a2 := b.PremulAt(2*x+d[0], 2*y+d[1])
				if r2 != r || g2 != g || b2 != bl || a2 != al {
					warn(joinKey(key, "2x"), "a pixelated sheet's 2x art should be its 1x art with every pixel doubled; (%d, %d) is not", 2*x+d[0], 2*y+d[1])
					return
				}
			}
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LoadSkinFile reads a skin from a directory holding skin.json or from a
// .uskin archive, named after the directory or the archive — what the lint
// command is pointed at.
func LoadSkinFile(path string) (*Skin, error) {
	base := filepath.Base(filepath.Clean(path))
	archive := strings.EqualFold(filepath.Ext(base), SkinArchiveExt)
	name := base
	if archive {
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	clean, err := SanitizeThemeName(name)
	if err != nil {
		return nil, &SkinError{Skin: name, Msg: err.Error()}
	}
	return loadSkinPath(path, clean, archive)
}
