//go:build !darwin

package platform

// Everywhere but macOS, a shortcut is held with Control. See [AccelMods].
const primaryModifier = ModCtrl
