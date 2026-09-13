package style

import (
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
)

// goldenParseName is not a builtin id, so parseThemeFile cannot quietly
// merge the Go era pack into the parsed document.
const goldenParseName = "zz-golden-check"

// TestShippedThemeJSONMatchesPacks keeps style/themes/*/theme.json in sync
// with the Go era packs of the same id. The files are dead data at runtime
// (era packs win in loadEmbedded) but they are the worked example users copy
// into ~/.config/uitoolkit/themes, so a drifted file is a documentation bug.
//
// Run with UITK_UPDATE_THEMES=1 to regenerate them.
func TestShippedThemeJSONMatchesPacks(t *testing.T) {
	update := os.Getenv("UITK_UPDATE_THEMES") == "1"
	seen := 0
	err := fs.WalkDir(starterFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path.Base(p) != "theme.json" {
			return err
		}
		name := path.Base(path.Dir(p))
		seen++
		pack, ok := builtinEraPack(name)
		if !ok {
			t.Errorf("%s: no Go era pack named %q; delete the file or add the pack", p, name)
			return nil
		}
		want, err := json.MarshalIndent(themeDoc(pack.Label, pack.Era, pack.Tokens), "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, '\n')
		got, err := starterFS.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if update {
			if err := os.WriteFile(filepath.FromSlash(p), want, 0o644); err != nil {
				t.Fatal(err)
			}
			return nil
		}
		if string(got) != string(want) {
			t.Errorf("%s is out of sync with pack %q; run UITK_UPDATE_THEMES=1 go test ./style/", p, name)
		}
		// The file must also parse back to the pack's tokens.
		parsed, err := parseThemeFile(goldenParseName, got, ThemeSourceBuiltin)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if parsed.Tokens.Bevel != pack.Tokens.Bevel {
			t.Errorf("%s bevel %q want %q", p, parsed.Tokens.Bevel, pack.Tokens.Bevel)
		}
		if parsed.Tokens.Family != pack.Tokens.Family {
			t.Errorf("%s family %q want %q", p, parsed.Tokens.Family, pack.Tokens.Family)
		}
		if parsed.Tokens.Metrics != pack.Tokens.Metrics {
			t.Errorf("%s metrics %+v want %+v", p, parsed.Tokens.Metrics, pack.Tokens.Metrics)
		}
		if a, b := tokensToColorMap(parsed.Tokens), tokensToColorMap(pack.Tokens); !reflect.DeepEqual(a, b) {
			for k, v := range b {
				if a[k] != v {
					t.Errorf("%s colour %q = %q want %q", p, k, a[k], v)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen == 0 {
		t.Fatal("no embedded theme.json found")
	}
}
