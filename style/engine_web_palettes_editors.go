//go:build theme_engine_all || theme_engine_web

package style

// The older editor palettes as web packs (research §1.3), each palette's
// colours in the UI roles its own scheme gives them where it gives any:
// Gruvbox and Solarized on the palettes' neutral web shape, Atom's One
// themes on Atom's 3px corners with its boxed editor tabs.

// pmenuParams are the palettes' shape with the menu's current item in a
// colour of its own (Vim's PmenuSel), its shortcut in the item's text.

// ---- Gruvbox ------------------------------------------------------------------------------------

// gruvboxSpecs follow gruvbox.vim's own groups (morhetz/gruvbox): bg0 for
// panes, bg1 for raised controls and bars, bg2 for popup menus (Pmenu),
// bg3 for splits, separators and the visual selection, fg1 for text and
// fg4 for secondary text; the menu's current item in blue (PmenuSel), blue
// the accent; the bright accents on the dark mode, the faded ones on the
// light. Fields sit in the hard-contrast bg0 (an inference).

// PmenuSel: the current menu item on blue.

// ---- Solarized ----------------------------------------------------------------------------------

// solarizedSpecs follow Solarized's own roles (altercation/solarized):
// base03 (base3) backgrounds, base02 (base2) background highlights for
// raised surfaces and selections, base1 (base01) emphasised content as the
// UI's text for its contrast, base0 (base00) body text as the secondary
// text, base01 (base1) comments for disabled text and, at half strength,
// separators; blue as the accent. The menu's current item is base2 on
// base01 (base02 on base1), as solarized.vim's PmenuSel shows it.

// ---- One Dark and One Light ---------------------------------------------------------------------

// oneParams are Atom's UI metrics on the palettes' shape: 3px corners,
// boxed editor tabs, the tree's full-width selection.

// oneSpecs are Atom's One Dark and One Light UI themes (atom/atom, the
// themes' LESS in HSL computed to hex): the tree, tab bar and status bar on
// the app colour round the pane's, raised buttons, the accent for focus and
// the default button, list selections in the button's hover colour with the
// highlight text; secondary and disabled text and the status colours are
// the syntax themes' greys and hues.
