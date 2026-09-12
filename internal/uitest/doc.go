// Package uitest is a headless widget harness: Measure/Arrange a tree,
// inject mouse/wheel/key, paint or record, and assert geometry.
//
// Inspired by Qt (QTest event injection + itemview scroll tests),
// GTK (fixtures + wait-for-draw + adjustment checks), and Avalonia
// (headless layout + input). Prefer paint-op / hit-test / rect
// assertions over screenshots.
//
// ChromeNorms / CompareReport map Button, ToolBar, Menu, ComboBox, and
// related controls to Avalonia / Qt / GTK / Office XP menu chrome
// (docs/compare.md).
package uitest
