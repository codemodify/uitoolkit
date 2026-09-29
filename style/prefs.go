package style

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const appearanceFile = "look.json"

// appearanceFileJSON is the on-disk XDG document. Other apps (Mail, gallery)
// read the same file via LoadAppearance / PreferredLook.
//
// Current format stores theme, corners, icons, and icon size independently:
//
//	{ "theme": "dark", "corners": "square", "icons": "lucide", "iconSize": "medium" }
//
// Compound v0.11–v0.12.1 theme ids (dark-round-classic, light-square-sharp,
// dark-round, …) migrate to palette + corners. Pack-level corners/icons
// are ignored.
type appearanceFileJSON struct {
	// Version 2 introduced corners "theme" (the theme's native shape) as
	// the default; older files carry "round" as the written-out default.
	Version  int    `json:"version,omitempty"`
	Theme    string `json:"theme"`
	Corners  string `json:"corners,omitempty"`
	Icons    string `json:"icons,omitempty"`
	IconSize string `json:"iconSize,omitempty"`
	// ReduceMotion turns animations off.
	ReduceMotion bool `json:"reduceMotion,omitempty"`
	// FollowDesktop swaps the theme for its light or dark sibling to match
	// the desktop.
	FollowDesktop bool `json:"followDesktop,omitempty"`
	// NativeDialogs uses the desktop's own file dialogs.
	NativeDialogs bool `json:"nativeDialogs,omitempty"`
	// ComboWheel lets the wheel over a closed combo box step its
	// selection. Left out unless it is on, like every other preference
	// here whose default is off.
	ComboWheel bool `json:"comboWheel,omitempty"`
	// Decorations: "auto" (omitted), "system" or "toolkit".
	Decorations string `json:"decorations,omitempty"`
	// CaptionButtons: "desktop" (omitted) or "theme".
	CaptionButtons string `json:"captionButtons,omitempty"`
	// Renderer: "auto" (omitted), "gpu" or "cpu" — which paint device a
	// new window's surface binds. UITK_PAINT overrides it and is never
	// written back here; see [RendererPref].
	Renderer string `json:"renderer,omitempty"`
	// FontUI and FontMono are the family names the user chose for the
	// two font roles, left out when the pack's own typefaces are wanted
	// (the default). They are stored as the family reads — "Liberation
	// Sans", not a slug — because that is what fontconfig is asked for
	// and what another application reading this file has to ask for
	// too. A family that is not installed on the machine that reads the
	// file is not an error: the pack's era fonts are still underneath
	// it. See [Appearance] and [withUserFonts].
	FontUI   string `json:"fontUI,omitempty"`
	FontMono string `json:"fontMono,omitempty"`
}

// ConfigDir is $XDG_CONFIG_HOME/uitoolkit (or ~/.config/uitoolkit).
func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit")
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, ".config", "uitoolkit")
	}
	return filepath.Join(os.TempDir(), "uitoolkit")
}

// CacheDir is $XDG_CACHE_HOME/uitoolkit (or the platform's own cache
// directory) — where the toolkit writes things it can regenerate, such
// as the colour-scheme files a desktop's own window frame reads.
//
// XDG_CACHE_HOME first on **every** platform, for the same reason
// [ConfigDir] reads XDG_CONFIG_HOME on every platform: the two have to
// follow one rule or a test that redirects one finds the other pointing
// at the real home directory. That is not hypothetical — this was
// os.UserCacheDir, which ignores XDG_CACHE_HOME on Windows and macOS,
// and a test that moved the cache aside on Linux quietly wrote into the
// developer's own %LocalAppData% on Windows and found somebody else's
// files there.
func CacheDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "uitoolkit")
	}
	if base, err := os.UserCacheDir(); err == nil && base != "" {
		return filepath.Join(base, "uitoolkit")
	}
	return filepath.Join(os.TempDir(), "uitoolkit")
}

// AppearancePath is $XDG_CONFIG_HOME/uitoolkit/look.json
// (or ~/.config/uitoolkit/look.json).
func AppearancePath() string {
	return filepath.Join(ConfigDir(), appearanceFile)
}

func writeJSONFile(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	// A temporary file of this writer's own, in the destination's
	// directory so the rename stays on one filesystem and is therefore
	// atomic.
	//
	// It used to be a fixed "<path>.tmp" shared by every writer, which
	// defeats the point of writing to a temporary file at all: two saves
	// truncate the same inode and interleave their bytes, one renames
	// the other's half-written file into place, and a failed save
	// removes a file its neighbour is about to publish. Two Settings
	// windows, or one application saving from two goroutines, could
	// leave invalid JSON where the preferences belong.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(b); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := replaceFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// replaceFile renames tmp over path, retrying briefly.
//
// The retry is Windows'. There, a rename over a file another process
// has open fails — Go opens files for reading without FILE_SHARE_DELETE
// — so publishing look.json fails exactly when another uitoolkit
// application happens to be reading it. That is not a rare race: every
// application *watches* that file (app/lookwatch.go), so a save races
// every other running app by design, and the failure mode is Apply
// reporting that it could not write the preferences.
//
// It is not only readers: two writers renaming over the same path at
// once collide there too, and a save is not the only thing saving —
// Settings can be open twice. So the wait backs off rather than
// spinning, out to about a quarter of a second in total, which is far
// longer than either holds the file and still imperceptible for
// something a user does by pressing Apply.
//
// On Unix the first attempt always succeeds and none of this runs.
func replaceFile(tmp, path string) error {
	var err error
	for i := 0; i < 60; i++ {
		if err = os.Rename(tmp, path); err == nil {
			return nil
		}
		// 1ms for the first few, growing to 16: quick enough that an
		// uncontended retry is over at once, patient enough that a
		// hundred writers all get through.
		time.Sleep(time.Duration(1+i/4) * time.Millisecond)
	}
	return err
}

func resolveAppearance(raw appearanceFileJSON) Appearance {
	a := DefaultAppearance()
	packName, cornersFromName, hasCorners := SplitLookThemeName(raw.Theme)
	if pack, ok := LoadTheme(strings.TrimSpace(raw.Theme)); ok {
		a.Name = pack.Name
		a.Theme = pack.Palette
	} else if pack, ok := LoadTheme(packName); ok {
		a.Name = pack.Name
		a.Theme = pack.Palette
	} else if packName == "light" {
		a.Name = StarterName(ThemeLight)
		a.Theme = ThemeLight
	} else if clean, err := SanitizeThemeName(packName); err == nil && clean != "" && clean != "dark" {
		// A pack this build cannot load keeps its name so the pref
		// survives — look.json is shared with other applications, and a
		// build without the engine must not throw the user's theme away
		// — but only in canonical form, because the name ends up in
		// file paths.
		//
		// Its family is **not** guessed from the name. That is what this
		// used to do, with ParseTheme, which answers the two words
		// "light" and "dark" and reads everything else as dark: every
		// pack a build was missing therefore came up dark and painted
		// the dark palette, so a light pack like metal-steel showed as
		// dark with nothing anywhere to say why. Theme stays the default
		// appearance's — the look this build would have shown if the
		// file had said nothing — and [Appearance.Missing] is how an
		// application finds out and says so.
		a.Name = clean
	}
	if strings.TrimSpace(raw.Corners) != "" {
		a.Corners = ParseCorners(raw.Corners)
		if raw.Version < 2 && a.Corners == CornersRound {
			// Pre-v2 files wrote "round" as the default, before themes had
			// shapes of their own: read it as "keep the theme's shape".
			a.Corners = CornersTheme
		}
	} else if hasCorners {
		a.Corners = cornersFromName
		if a.Corners == CornersRound {
			a.Corners = CornersTheme
		}
	}
	if strings.TrimSpace(raw.Icons) != "" {
		a.Icons = ParseIconSet(raw.Icons)
	}
	if strings.TrimSpace(raw.IconSize) != "" {
		a.IconSize = ParseIconSize(raw.IconSize)
	}
	a.FontUI = NormalizeFontChoice(raw.FontUI)
	a.FontMono = NormalizeFontChoice(raw.FontMono)
	a.ReduceMotion = raw.ReduceMotion
	a.FollowDesktop = raw.FollowDesktop
	a.NativeDialogs = raw.NativeDialogs
	a.ComboWheel = raw.ComboWheel
	a.Decorations = ParseDecorationsPref(raw.Decorations)
	a.CaptionButtons = ParseCaptionButtonsPref(raw.CaptionButtons)
	a.Renderer = ParseRendererPref(raw.Renderer)
	return a.Normalize()
}

// LoadAppearance reads XDG look.json. Missing or invalid files yield defaults.
// Compound theme ids and a legacy triad both resolve to independent
// theme / corners / icons fields.
//
// UITK_THEME=<pack> overrides the saved theme for this process, like
// GTK_THEME or QT_STYLE_OVERRIDE (UITK_THEME=win95 ./app); corners, icons
// and icon size still come from look.json. The override is never written
// back unless the user saves from Settings.
func LoadAppearance() Appearance {
	var raw appearanceFileJSON
	ok := false
	if b, err := os.ReadFile(AppearancePath()); err == nil && json.Unmarshal(b, &raw) == nil {
		ok = true
	} else {
		raw = appearanceFileJSON{}
	}
	if env := strings.TrimSpace(os.Getenv(ThemeEnv)); env != "" {
		// The pack asked for is the pack shown: no swapping it for its
		// light or dark sibling.
		raw.Theme = env
		raw.FollowDesktop = false
		ok = true
	}
	a := DefaultAppearance()
	if ok {
		a = resolveAppearance(raw)
	}
	// UITK_FONT / UITK_FONT_MONO override the saved typefaces for this
	// process alone, the way UITK_THEME overrides the pack. Setting one
	// to "theme" puts that role back on the pack's era fonts without the
	// file being touched, which is how a session is run at a pack's own
	// typography for a screenshot.
	//
	// They are applied to the appearance rather than to the file read
	// into it, because a typeface says nothing about the rest of it.
	// Folding them into the raw file made a machine with no look.json
	// resolve an empty file instead of the defaults, and an empty file
	// says FollowDesktop is off — so `UITK_FONT=theme ./app` stopped the
	// program following the desktop's dark mode, which is a setting it
	// was not asked about.
	for _, env := range []struct {
		name string
		into *string
	}{{FontEnv, &a.FontUI}, {MonoFontEnv, &a.FontMono}} {
		if v, set := os.LookupEnv(env.name); set {
			*env.into = NormalizeFontChoice(v)
		}
	}
	return a
}

// ThemeEnv names the environment variable that overrides the saved theme.
const ThemeEnv = "UITK_THEME"

// SaveAppearance writes look.json with theme, corners, icons, and iconSize (mode 0600).
//
// The renderer goes in as "renderer" and auto is left out, the way auto
// decorations and the desktop's caption buttons are. UITK_PAINT is never
// written here: it overrides the file in the process that has it set,
// and a Settings started under it must not bake it into everyone's
// preferences.
func SaveAppearance(a Appearance) error {
	a = a.Normalize()
	return writeJSONFile(AppearancePath(), appearanceFileJSON{
		Version:  2,
		Theme:    a.Name,
		Corners:  string(a.Corners),
		Icons:    string(a.Icons),
		IconSize: string(a.IconSize),

		FontUI:   a.FontUI,
		FontMono: a.FontMono,

		ReduceMotion:   a.ReduceMotion,
		FollowDesktop:  a.FollowDesktop,
		NativeDialogs:  a.NativeDialogs,
		ComboWheel:     a.ComboWheel,
		Decorations:    decorationsJSON(a.Decorations),
		CaptionButtons: string(ParseCaptionButtonsPref(string(a.CaptionButtons))),
		Renderer:       string(ParseRendererPref(string(a.Renderer))),
	})
}

// decorationsJSON is the look.json value of d (left out for auto, the
// default).
func decorationsJSON(d DecorationsPref) string {
	return string(ParseDecorationsPref(string(d)))
}
