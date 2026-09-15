//go:build linux

package platform

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

// The portal's Read wraps the value in one more variant than ReadOne.
func TestSchemeOfUnwrapsVariants(t *testing.T) {
	for _, c := range []struct {
		v    dbus.Variant
		want ColorScheme
		ok   bool
	}{
		{dbus.MakeVariant(uint32(1)), SchemeDark, true},
		{dbus.MakeVariant(dbus.MakeVariant(uint32(2))), SchemeLight, true},
		{dbus.MakeVariant(uint32(0)), SchemeNoPreference, true},
		{dbus.MakeVariant(uint32(7)), SchemeNoPreference, true},
		{dbus.MakeVariant("dark"), SchemeNoPreference, false},
	} {
		got, ok := schemeOf(c.v)
		if got != c.want || ok != c.ok {
			t.Errorf("schemeOf(%v) = %v %v, want %v %v", c.v, got, ok, c.want, c.ok)
		}
	}
}
