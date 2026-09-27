//go:build theme_engine_all || theme_engine_web

package style

// The editor palettes as web packs: each palette's colours placed in the UI
// roles its own style guide, docs, spec or reference port gives them
// (research §1.3), on a neutral web shape (6px controls, 8px popovers, a 2px
// focus band, underlined tabs, pill switches).

// paletteRoles are a palette's colours by UI role.

// surfaces
// buttons and other raised controls
// input borders, separators

// list selection, row hover, text selection

// paletteParams are the shared shapes of the palette packs.

// paletteMetrics are the palette packs' sizes at 14px text (room for the
// check boxes' focus band round the 16px box).

// paletteSpec is a palette pack from its roles.

// ---- Catppuccin ---------------------------------------------------------------------------------

// catppuccinFlavour is one flavour's 26 colours (catppuccin/palette 1.8).

// catppuccinSpecs follow the style guide: Base for panes, Mantle and Crust
// for secondary panes (sidebars, bars), Surface 0–2 for raised controls and
// popovers, Overlay 0 for inactive borders and Lavender for the active one,
// Text and Subtext for labels, Overlay 2 at 25% for selections, Blue as the
// accent with Base on it; Red, Yellow and Green for errors, warnings and
// success.

// ---- Nord ---------------------------------------------------------------------------------------

// nordSpecs follow Nord's docs: Polar Night nord0 for the dark background,
// nord1 for raised UI (bars, panels, popups, buttons, fields), nord2 for
// selections, nord3 for disabled UI, nord4 for text, the Frost nord8 as the
// accent; Snow Storm for the light scheme (nord6 background, nord4 raised UI
// and borders, nord5 selections, nord0 text) with the deeper Frost nord10 as
// its accent (nord8 is too pale there; an inference).

// ---- Dracula and Alucard ------------------------------------------------------------------------

// draculaSpecs follow the Dracula spec: Background for panes, Background
// Dark for sidebars and Darker for bars, the Floating colour for menus and
// popovers, Current Line's solid fallback for subtle borders, Selection,
// Comment for disabled text, Purple as the accent, and the spec's
// functional colours (the focus ring, error, warning, success, links).

// ---- Tokyo Night --------------------------------------------------------------------------------

// tokyoNightSpecs take the UI keys of the Tokyo Night VS Code theme (Night
// and Storm: editor and chrome colours, the #3d59a1 accent of buttons and the
// active tab, list selection and hover, input and focus borders) and, for
// Day, the palette of tokyonight.nvim's Day style.

// ---- Rosé Pine ----------------------------------------------------------------------------------

// rosePineSpecs follow rosepinetheme.com's roles: base for windows and side
// bars, surface for inputs, bars and popups, overlay for hovered and selected
// items and dialogs, highlight med for selections, highlight high for
// borders, text / subtle / muted for labels; love, gold, foam and pine for
// errors, warnings, information and success; iris (the links' colour) as the
// accent — a choice, the palette names none.
