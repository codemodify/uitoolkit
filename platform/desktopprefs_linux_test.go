//go:build linux

package platform

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

// The portal's Read wraps a value in one more variant than ReadOne.
func TestDesktopPrefsKeys(t *testing.T) {
	var p DesktopPrefs
	if !p.set("color-scheme", dbus.MakeVariant(dbus.MakeVariant(uint32(1)))) || p.ColorScheme != SchemeDark {
		t.Fatalf("color-scheme %+v", p)
	}
	if !p.set("color-scheme", dbus.MakeVariant(uint32(2))) || p.ColorScheme != SchemeLight {
		t.Fatalf("color-scheme light %+v", p)
	}
	if !p.set("color-scheme", dbus.MakeVariant(uint32(7))) || p.ColorScheme != SchemeNoPreference {
		t.Fatalf("color-scheme unknown %+v", p)
	}
	if p.set("color-scheme", dbus.MakeVariant("dark")) {
		t.Fatal("a string is not a scheme")
	}
	if !p.set("reduced-motion", dbus.MakeVariant(uint32(1))) || !p.ReducedMotion {
		t.Fatalf("reduced-motion %+v", p)
	}
	if !p.set("contrast", dbus.MakeVariant(uint32(1))) || !p.HighContrast {
		t.Fatalf("contrast %+v", p)
	}
	accent := dbus.MakeVariant([]interface{}{0.25, 0.5, 0.75})
	if !p.set("accent-color", accent) || !p.HasAccent || p.Accent != [3]float64{0.25, 0.5, 0.75} {
		t.Fatalf("accent %+v", p)
	}
	// Out of range: no accent set.
	p.set("accent-color", dbus.MakeVariant([]interface{}{-1.0, 2.0, 0.5}))
	if p.HasAccent {
		t.Fatal("an out-of-range accent is unset")
	}
	if p.set("cursor-theme", dbus.MakeVariant("x")) {
		t.Fatal("unknown key")
	}
}

func TestSessionBusAddressNeverLaunches(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	if a := sessionBusAddress(); a != "" {
		t.Fatalf("address %q without a bus", a)
	}
	if _, ok := ReadDesktopPrefs(0); ok {
		t.Fatal("no bus, no prefs")
	}
}

func TestDesktopPrefsTitleBarKeys(t *testing.T) {
	var p DesktopPrefs
	if !p.setNS(gnomeWMNS, "button-layout", dbus.MakeVariant(dbus.MakeVariant("appmenu:close"))) || p.ButtonLayout != "appmenu:close" {
		t.Fatalf("layout %+v", p)
	}
	if !p.setNS(gnomeWMNS, "action-double-click-titlebar", dbus.MakeVariant("minimize")) || p.TitlebarDoubleClick != "minimize" {
		t.Fatalf("double click %+v", p)
	}
	if !p.setNS(gnomeWMNS, "action-right-click-titlebar", dbus.MakeVariant("menu")) || p.TitlebarRightClick != "menu" {
		t.Fatalf("right click %+v", p)
	}
	if !p.setNS(gnomeMouseNS, "double-click", dbus.MakeVariant(int32(350))) || p.DoubleClickTime != 350 {
		t.Fatalf("double-click time %+v", p)
	}
	if !p.setNS(gnomeMouseNS, "drag-threshold", dbus.MakeVariant(int32(9))) || p.DragThreshold != 9 {
		t.Fatalf("drag threshold %+v", p)
	}
	if p.setNS(gnomeWMNS, "button-layout", dbus.MakeVariant(uint32(1))) {
		t.Fatal("a number is no layout")
	}
	if p.setNS(gnomeWMNS, "theme", dbus.MakeVariant("Adwaita")) {
		t.Fatal("an unrelated key")
	}
	if !p.setNS(appearanceNS, "color-scheme", dbus.MakeVariant(uint32(1))) || p.ColorScheme != SchemeDark {
		t.Fatal("appearance keys still arrive through setNS")
	}
}
