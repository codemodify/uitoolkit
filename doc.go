// Package uitoolkit is a retained-mode desktop UI toolkit that paints
// exclusively with github.com/codemodify/paintengine2d.
//
// Layers:
//
//	platform  — window, event pump, present (X11 on Linux; stubs elsewhere)
//	app       — run loop, windows, DPI, input routing
//	widget    — Component tree, focus, hit-test, Invalidate / Damage
//	layout    — row, column, stack, flex
//	widgets   — Button, Label, TextField, Checkbox, Slider, ScrollView, …
//	style     — LookAndFeel themes (colors, radii, baked glyph atlases)
//
// All pixels go through paintengine2d.Context. There is no second rasterizer.
package uitoolkit
