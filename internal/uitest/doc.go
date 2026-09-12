// Package uitest is a headless widget harness: Measure/Arrange a tree,
// inject mouse/wheel/key, paint or record, and assert geometry.
//
// Inspired by Qt (QTest event injection + itemview scroll tests),
// GTK (fixtures + wait-for-draw + adjustment checks), and Avalonia
// (headless layout + input). Prefer paint-op / hit-test / rect
// assertions over screenshots.
//
// ChromeNorms / CompareReport map Button, ToolBar, Menu, ComboBox,
// ScrollView, Splitter, TableView, Switch, Slider, Tabs, and dialog
// chrome to Avalonia / Qt / GTK / Office XP (docs/compare.md).
// ChromeStrip + testdata/chrome-strip-{1,2}x.png guard full-frame paint.
package uitest
