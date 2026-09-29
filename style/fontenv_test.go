package style

import (
	"os"
	"testing"
)

// A typeface override says which face draws and nothing about the rest
// of the appearance. It used to be folded into the file read from disk,
// which made a machine with no look.json resolve an *empty* file instead
// of the defaults — and an empty file says FollowDesktop is off. So
// `UITK_FONT=theme ./app` quietly stopped the program following the
// desktop's dark mode, a setting it was never asked about.
func TestFontEnvKeepsTheOtherDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := os.Stat(AppearancePath()); err == nil {
		t.Fatal("the temporary config dir already has a look.json")
	}
	base := LoadAppearance()
	if !base.FollowDesktop {
		t.Fatal("a machine with no look.json should follow the desktop")
	}

	for _, env := range []string{FontEnv, MonoFontEnv} {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "theme")
			a := LoadAppearance()
			if !a.FollowDesktop {
				t.Error("FollowDesktop went off with only a font named")
			}
			if a.Name != base.Name {
				t.Errorf("pack changed: %q, want %q", a.Name, base.Name)
			}
		})
	}

	// And the override still takes effect.
	t.Setenv(FontEnv, "Liberation Sans")
	if a := LoadAppearance(); a.FontUI != "Liberation Sans" {
		t.Errorf("FontUI %q, want the environment's", a.FontUI)
	}
	// "theme" means the pack's own, which is the empty choice.
	t.Setenv(FontEnv, "theme")
	if a := LoadAppearance(); a.FontUI != "" {
		t.Errorf(`FontUI %q after UITK_FONT=theme, want ""`, a.FontUI)
	}
}
