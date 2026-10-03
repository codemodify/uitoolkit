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

// Re-registering an id replaces what is painted, in looks made before
// and after.
//
// Two in-memory registrations of one id have the same name and no
// directory, so the decoded-asset cache keyed on those alone handed the
// old pixels back; and a look memoized its *Skin for its own lifetime,
// so it went on painting the old one however many times the id was
// re-registered. The pack reported the new skin the whole time, which is
// what makes it worth a test: every name said it had changed.
func TestReRegisteringASkinReplacesWhatIsPainted(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	InvalidateSkinCache()
	const id = "swapped"

	reg := func(cap string) {
		fsys := fstest.MapFS{
			SkinFile:         {Data: []byte(strings.Replace(shortCapDoc, "%CAP%", cap, 1))},
			"art/sheet.png":  {Data: skinTestPNG(t, 1)},
			"art/sheet2.png": {Data: skinTestPNG(t, 2)},
		}
		if err := RegisterSkinFS(fsys, id); err != nil {
			t.Fatalf("register: %v", err)
		}
	}

	reg("20")
	p, ok := LoadTheme(id)
	if !ok {
		t.Fatal("not registered")
	}
	look := p.Look() // a look made before the replacement
	first := skinFor(look)
	if first == nil {
		t.Fatal("the look resolved no skin")
	}

	// The same id, a different skin.
	reg("30")
	after := skinFor(look)
	if after == nil {
		t.Fatal("the look resolved no skin after the replacement")
	}
	if after == first {
		t.Error("a look made before the replacement still resolves the old skin")
	}
	if after.gen == first.gen {
		t.Error("the two registrations share an identity, so they share cached artwork")
	}
	// And a fresh look gets the new one too.
	p2, _ := LoadTheme(id)
	if fresh := skinFor(p2.Look()); fresh == nil || fresh.gen != after.gen {
		t.Error("a fresh look does not resolve the current registration")
	}
}
