package style

import (
	"io/fs"
)

// RegisterSkin makes a skin this process parsed available to be painted,
// under its own name, as though it had been embedded in the toolkit.
//
// A binary can embed its own skins and parse them with [LoadSkinFS], but
// until now nothing could make the result paintable: painting and slot
// lookups resolve an id through [LoadSkin], which reads the user's skin
// directory and the toolkit's built-ins and knew nothing of a skin the
// application had in hand. Registering its [ThemePack] was not enough —
// that is the palette and the metrics, not the art. The way through was
// to write the skin into a temporary directory and put that on the search
// path, which means disk, a lifetime to manage, and files outside the
// application's control.
//
// A skin registered here is found by [LoadSkin] and by everything built
// on it, and its pack is registered too, so [LoadTheme] answers for it.
//
// **A skin of the user's own wins.** LoadSkin reads the user's directory
// first, so a person who has written a skin of that name still gets
// theirs: an application cannot take a name away from them by embedding
// one. That is the same rule the toolkit's own built-ins follow.
//
// Registering a name twice replaces it, which is what a reload wants.
//
// An application's skins are kept apart from the toolkit's own: "built
// in" means shipped here, and the checks that hold a shipped skin to the
// format's rules are about those. They are found after the user's and
// before the toolkit's, so an application may override a shipped skin by
// name and a person may still override the application.
func RegisterSkin(sk *Skin) error {
	if sk == nil {
		return &SkinError{Msg: "no skin"}
	}
	clean, err := SanitizeThemeName(sk.Name)
	if err != nil {
		return &SkinError{Skin: sk.Name, Msg: err.Error()}
	}
	pack := sk.Pack()
	skinRegistry.mu.Lock()
	if _, seen := skinRegistry.app[clean]; !seen {
		skinRegistry.appOrder = append(skinRegistry.appOrder, clean)
	}
	skinRegistry.app[clean] = sk
	skinRegistry.mu.Unlock()
	RegisterPack(pack)
	InvalidateSkinCache()
	return nil
}

// RegisterSkinFS parses a skin out of fsys under the given id and
// registers it ([LoadSkinFS] then [RegisterSkin]), which is the whole of
// what an application embedding a skin has to do:
//
//	//go:embed skins/nocturne
//	var art embed.FS
//	sub, _ := fs.Sub(art, "skins/nocturne")
//	if err := style.RegisterSkinFS(sub, "nocturne"); err != nil { … }
//
// The filesystem is kept: a skin reads its artwork lazily, so fsys has to
// go on answering for as long as the skin may be painted. An embed.FS
// does, which is the case this exists for.
func RegisterSkinFS(fsys fs.FS, id string) error {
	sk, err := LoadSkinFS(fsys, id)
	if err != nil {
		return err
	}
	return RegisterSkin(sk)
}
