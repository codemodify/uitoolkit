package style

import (
	"strings"
	"testing"
	"testing/fstest"
)

// A skin a binary embeds can be made paintable without touching the disk.
//
// LoadSkinFS parsed one, but nothing could register the result: painting
// and slot lookups resolve an id through LoadSkin, which read the user's
// directory and the toolkit's built-ins and knew nothing of a skin the
// application was holding. Registering its ThemePack was not enough —
// that is the palette and the metrics, not the art. The only way through
// was a temporary directory on the search path.
func TestAnEmbeddedSkinCanBeRegistered(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	InvalidateSkinCache()

	fsys := fstest.MapFS{
		SkinFile:         {Data: []byte(strings.Replace(shortCapDoc, "%CAP%", "20", 1))},
		"art/sheet.png":  {Data: skinTestPNG(t, 1)},
		"art/sheet2.png": {Data: skinTestPNG(t, 2)},
	}
	const id = "embedded"
	if _, ok := LoadSkin(id); ok {
		t.Fatal("the id was already taken")
	}
	if err := RegisterSkinFS(fsys, id); err != nil {
		t.Fatalf("register: %v", err)
	}

	sk, ok := LoadSkin(id)
	if !ok {
		t.Fatal("LoadSkin does not find a registered skin, so nothing can paint it")
	}
	if sk.Name != id {
		t.Errorf("registered as %q, want %q", sk.Name, id)
	}
	// And its pack resolves, which is what a look is built from.
	if _, ok := LoadTheme(id); !ok {
		t.Error("LoadTheme does not know the registered skin's pack")
	}
	if !IsSkin(id) {
		t.Error("the registered id is not reported as a skin")
	}

	// It is the application's, not the toolkit's: the checks that hold a
	// shipped skin to the format's rules are about what ships here, and
	// registering put this one among them until they were kept apart.
	skinRegistry.mu.RLock()
	_, inBuiltin := skinRegistry.builtin[id]
	_, inApp := skinRegistry.app[id]
	skinRegistry.mu.RUnlock()
	if inBuiltin {
		t.Error("an application's skin was registered as one of the toolkit's")
	}
	if !inApp {
		t.Error("the skin is not in the application's own registry")
	}
}
