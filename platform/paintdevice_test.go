package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The notice is the rule all three backends share, so it is tested once here
// rather than three times behind a display.

func TestTheFirstPaintDeviceIsNotNews(t *testing.T) {
	var n paintDeviceNotice
	dev := paintengine2d.NewCPUDevice(paintengine2d.NewImage(4, 4))
	if ev := n.paintDeviceEvent(dev); len(ev) != 0 {
		t.Errorf("the first device queued %d events, want none — an application that asks at start-up already has it", len(ev))
	}
	if ev := n.paintDeviceEvent(dev); len(ev) != 0 {
		t.Errorf("the same device again queued %d events", len(ev))
	}
}

func TestAReplacedPaintDeviceIsNews(t *testing.T) {
	var n paintDeviceNotice
	first := paintengine2d.NewCPUDevice(paintengine2d.NewImage(4, 4))
	second := paintengine2d.NewCPUDevice(paintengine2d.NewImage(4, 4))
	n.paintDeviceEvent(first)
	ev := n.paintDeviceEvent(second)
	if len(ev) != 1 || ev[0].Kind != EventPaintDevice {
		t.Fatalf("a replaced device queued %v, want one EventPaintDevice", ev)
	}
	if ev := n.paintDeviceEvent(second); len(ev) != 0 {
		t.Errorf("the new device again queued %d events", len(ev))
	}
}

// Losing it is the change that matters most: a window that fell back to the
// CPU has no GPU device, and an application holding a texture on the old one
// needs to hear that before it draws with a name that means nothing.
func TestLosingThePaintDeviceIsNews(t *testing.T) {
	var n paintDeviceNotice
	dev := paintengine2d.NewCPUDevice(paintengine2d.NewImage(4, 4))
	n.paintDeviceEvent(dev)
	ev := n.paintDeviceEvent(nil)
	if len(ev) != 1 || ev[0].Kind != EventPaintDevice {
		t.Fatalf("losing the device queued %v, want one EventPaintDevice", ev)
	}
	if ev := n.paintDeviceEvent(nil); len(ev) != 0 {
		t.Errorf("still having no device queued %d events", len(ev))
	}
	// And getting one back is news again.
	if ev := n.paintDeviceEvent(dev); len(ev) != 1 {
		t.Errorf("a device arriving after none queued %d events, want 1", len(ev))
	}
}

// A surface that never had one and never gets one says nothing at all.
func TestNoPaintDeviceEverSaysNothing(t *testing.T) {
	var n paintDeviceNotice
	for i := 0; i < 3; i++ {
		if ev := n.paintDeviceEvent(nil); len(ev) != 0 {
			t.Fatalf("a surface with no device queued %d events", len(ev))
		}
	}
}
