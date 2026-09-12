// Package uitoolkit is a retained-mode desktop UI toolkit that paints
// exclusively with github.com/codemodify/paintengine2d.
//
// Layers:
//
//	platform  — window, event pump, present, OS clipboard, IME (X11 and Wayland on Linux; stubs elsewhere)
//	app       — event-driven run loop, retained scene (UITK_SCENE), windows, DPI, input routing
//	widget    — Component tree, focus, hit-test, Invalidate / Damage
//	layout    — row, column, stack, flex
//	widgets   — Button, Label, TextField, TextArea, TextView, NumberField, Checkbox,
//	            Switch, Slider, ScrollView, MenuBar, TabView, TreeView,
//	            TableView, CardList, StatusBar, ToolBar, ComboBox, ProgressBar,
//	            RadioGroup, MessageBox, TitleBar, ListView, FileDialog,
//	            Tooltip, Accordion, Expander, Separator, Spacer, …
//	style     — LookAndFeel themes (Titillium Web + JetBrains Mono OpenType atlases),
//	            named theme packs (embedded starters + XDG themes/<name>/theme.json;
//	            look.json stores the pack name; live watch)
//
// All pixels go through paintengine2d.Context (CPU scanline or Linux
// EGL/GLES2 GPUDevice). There is no second rasterizer.
package uitoolkit
