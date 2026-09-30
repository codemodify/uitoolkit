package style

import "testing"

// The four runes the documentation says are safe as characters are the
// four the fallback draws, and no others.
//
// docs/widgets.md tells applications that a mark is an icon and that
// these four are the exception. A fifth added here without the
// documentation, or one removed from here while the documentation still
// promises it, is a rune an application would type and a user would see
// as a box.
func TestDocumentedSymbolFallbacks(t *testing.T) {
	for _, r := range []rune{'★', '📎', '●', '🔇'} {
		if p, adv := symbolFallback(r, 16); p == nil || adv <= 0 {
			t.Errorf("%q has no fallback, but the documentation says it is safe", r)
		}
	}
	// Everything else is a box, which is the rule the icons exist for.
	for _, r := range []rune{'✓', '✗', '→', '↳', '⊘', '└', '☆', '🔔'} {
		if p, _ := symbolFallback(r, 16); p != nil {
			t.Errorf("%q is drawn, so the documentation's list of four is wrong", r)
		}
	}
}

// Every typed icon is named, labelled and reachable by its name, so the
// table in docs/widgets.md can be trusted to be the whole list.
func TestEveryToolIconIsNamedAndReachable(t *testing.T) {
	ids := AllToolIcons()
	if len(ids) != 42 {
		t.Fatalf("%d typed icons; docs/widgets.md lists 42", len(ids))
	}
	for _, i := range ids {
		name := ToolIconName(i)
		if name == "" || i.Label() == "" {
			t.Errorf("%v: name %q label %q", i, name, i.Label())
		}
		if got, ok := ToolIconByName(name); !ok || got != i {
			t.Errorf("ToolIconByName(%q) = %v %v", name, got, ok)
		}
		if len(ToolIconThemeNames(i)) == 0 {
			t.Errorf("%s has no freedesktop name, so an installed icon theme cannot answer for it", name)
		}
	}
}

// The wider stem vocabulary the documentation points applications at is
// there, and the names it gives as examples resolve.
func TestDocumentedStemsExist(t *testing.T) {
	stems := map[string]bool{}
	for _, s := range ShippedIconStems() {
		stems[s] = true
	}
	if len(stems) < 60 {
		t.Fatalf("%d stems; docs/widgets.md says 79", len(stems))
	}
	for _, s := range []string{
		"trash", "archive", "send", "sync", "lock", "eye", "folder",
		"chevron-up", "chevron-down", "chevron-left", "chevron-right",
		"arrow-up", "arrow-down", "arrow-left", "arrow-right",
	} {
		if !stems[s] {
			t.Errorf("docs/widgets.md names %q as a shipped stem and it is not one", s)
		}
	}
}
