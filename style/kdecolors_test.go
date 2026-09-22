package style

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// parseKDEScheme reads a colour-scheme file the way KConfig does: groups in
// brackets ("[Colors:Header][Inactive]" is the Inactive group nested in
// Colors:Header), key=value lines, blank lines between. It fails on
// anything else.
func parseKDEScheme(t *testing.T, data []byte) map[string]map[string]string {
	t.Helper()
	groupRe := regexp.MustCompile(`^(\[[^\[\]]+\])+$`)
	keyRe := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*=.*$`)
	out := map[string]map[string]string{}
	cur := ""
	sc := bufio.NewScanner(bytes.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		switch {
		case line == "":
		case groupRe.MatchString(line):
			cur = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if _, dup := out[cur]; dup {
				t.Fatalf("line %d: group %q twice", n, cur)
			}
			out[cur] = map[string]string{}
		case keyRe.MatchString(line):
			if cur == "" {
				t.Fatalf("line %d: key outside a group: %q", n, line)
			}
			k, v, _ := strings.Cut(line, "=")
			if _, dup := out[cur][k]; dup {
				t.Fatalf("line %d: %s twice in [%s]", n, k, cur)
			}
			out[cur][k] = v
		default:
			t.Fatalf("line %d is not KConfig: %q", n, line)
		}
	}
	return out
}

// rgbOf reads "r,g,b" with each channel 0 to 255, as a scheme states a
// colour.
func rgbOf(t *testing.T, where, v string) [3]int {
	t.Helper()
	parts := strings.Split(v, ",")
	if len(parts) != 3 {
		t.Fatalf("%s = %q: not r,g,b", where, v)
	}
	var c [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 255 {
			t.Fatalf("%s = %q: channel %q is not 0..255", where, v, p)
		}
		c[i] = n
	}
	return c
}

// The scheme is KDE's format: [General] names it, every colour set holds
// all twelve roles as r,g,b, the Header set has its Inactive state, and
// [WM] has the title-bar colours.
func TestKDEColorSchemeFormat(t *testing.T) {
	for _, pack := range []string{"luna", "aqua", "tahoe", "win95", "beos", "aero"} {
		t.Run(pack, func(t *testing.T) {
			lk := retroLook(t, pack, 1)
			g := parseKDEScheme(t, KDEColorScheme(lk, "uitoolkit "+pack))
			if gen := g["General"]; gen["ColorScheme"] != "uitoolkit "+pack || gen["Name"] == "" {
				t.Fatalf("[General] %v", gen)
			}
			for _, set := range append(KDEColorSchemeSets, "Header][Inactive") {
				grp, ok := g["Colors:"+set]
				if !ok {
					t.Fatalf("no [Colors:%s]", set)
				}
				for _, role := range KDEColorRoles {
					v, ok := grp[role]
					if !ok {
						t.Fatalf("[Colors:%s] has no %s", set, role)
					}
					rgbOf(t, set+"."+role, v)
				}
				if len(grp) != len(KDEColorRoles) {
					t.Fatalf("[Colors:%s] has %d keys, want the %d roles", set, len(grp), len(KDEColorRoles))
				}
			}
			for _, k := range KDEWMKeys {
				v, ok := g["WM"][k]
				if !ok {
					t.Fatalf("[WM] has no %s", k)
				}
				rgbOf(t, "WM."+k, v)
			}
			if g["ColorEffects:Inactive"]["Enable"] != "false" {
				t.Fatal("an inactive window's colours are stated, not an effect")
			}
		})
	}
}

// The title bar is the look's caption: Luna's blue with its white title,
// Aqua's and Tahoe's light grey with a dark one, the backdrop's its own.
func TestKDEColorSchemeTitleBarIsTheLooksCaption(t *testing.T) {
	type want struct {
		blue, light, whiteTitle bool
	}
	for pack, w := range map[string]want{
		"luna":  {blue: true, whiteTitle: true},
		"aqua":  {light: true},
		"tahoe": {light: true},
		"win95": {blue: true, whiteTitle: true},
	} {
		t.Run(pack, func(t *testing.T) {
			lk := retroLook(t, pack, 1)
			g := parseKDEScheme(t, KDEColorScheme(lk, pack))
			bar := rgbOf(t, "header", g["Colors:Header"]["BackgroundNormal"])
			title := rgbOf(t, "title", g["Colors:Header"]["ForegroundNormal"])
			if wm := rgbOf(t, "wm", g["WM"]["activeBackground"]); wm != bar {
				t.Errorf("[WM] activeBackground %v, the Header set's %v", wm, bar)
			}
			if w.blue && !(bar[2] > bar[0]+60 && bar[2] > 120) {
				t.Errorf("title bar %v is not the look's blue", bar)
			}
			if w.light && !(bar[0] > 170 && bar[1] > 170 && bar[2] > 170) {
				t.Errorf("title bar %v is not the look's light grey", bar)
			}
			lum := func(c [3]int) int { return (3*c[0] + 6*c[1] + c[2]) / 10 }
			if w.whiteTitle && lum(title) < 200 {
				t.Errorf("title %v on %v is not white", title, bar)
			}
			if !w.whiteTitle && lum(title) > 90 {
				t.Errorf("title %v on %v is not dark", title, bar)
			}
			if d := lum(title) - lum(bar); d < 80 && d > -80 {
				t.Errorf("title %v does not read on %v", title, bar)
			}
			off := rgbOf(t, "inactive", g["Colors:Header][Inactive"]["BackgroundNormal"])
			if pack == "luna" && off == bar {
				t.Errorf("Luna's backdrop caption %v is its active one", off)
			}
		})
	}
}

// The rest of the scheme is the look's palette.
func TestKDEColorSchemeFollowsThePalette(t *testing.T) {
	lk := retroLook(t, "win95", 1)
	g := parseKDEScheme(t, KDEColorScheme(lk, "w"))
	p := lk.Palette()
	if got, want := g["Colors:Window"]["BackgroundNormal"], kdeRGB(p.Background); got != want {
		t.Errorf("window background %s, the palette's %s", got, want)
	}
	if got, want := g["Colors:View"]["BackgroundNormal"], kdeRGB(over(p.Field, p.Background)); got != want {
		t.Errorf("view background %s, the palette's field %s", got, want)
	}
	if got, want := g["Colors:Window"]["ForegroundNormal"], kdeRGB(p.Text); got != want {
		t.Errorf("window text %s, the palette's %s", got, want)
	}
}
