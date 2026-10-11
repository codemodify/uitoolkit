package platform

import (
	"errors"
	"testing"
)

type recordingCloser struct {
	on func()
}

func (c *recordingCloser) Close() error {
	if c.on != nil {
		c.on()
	}
	return errors.New("closed")
}

// **The application is told before the device closes, not after.**
//
// That order is the whole of this hook: libmpv's render API requires the
// context its renderer was created against to be current when the
// renderer is destroyed, and every backend closed the device and only
// then queued the change — so by the time anything heard, the context it
// needed was gone. Reported against media-player-video.
//
// Both backends go through closeGPUDevice so the order is written once;
// doing it by hand in each is how it came to be wrong, and a
// hand-written sequence cannot be tested without a GPU.
func TestTheApplicationIsToldBeforeTheGPUDeviceCloses(t *testing.T) {
	was := closingGPU
	t.Cleanup(func() { closingGPU = was })

	var order []string
	SetGPUClosing(func(Surface) { order = append(order, "told") })
	surf := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	closeGPUDevice(surf, &recordingCloser{on: func() { order = append(order, "closed") }})

	if len(order) != 2 || order[0] != "told" || order[1] != "closed" {
		t.Errorf("the order was %v, want told then closed", order)
	}
}

// The surface the hook is handed is the one whose device is closing, so
// the application can find the window it belongs to.
func TestTheClosingHookIsGivenItsOwnSurface(t *testing.T) {
	was := closingGPU
	t.Cleanup(func() { closingGPU = was })

	a := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	b := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	var got Surface
	SetGPUClosing(func(s Surface) { got = s })
	closeGPUDevice(b, &recordingCloser{})
	if got != Surface(b) {
		t.Errorf("the hook was given %p, want the surface whose device closed (%p, not %p)", got, b, a)
	}
}

// With no hook registered, closing still closes.
func TestClosingWithNoHookStillCloses(t *testing.T) {
	was := closingGPU
	t.Cleanup(func() { closingGPU = was })
	SetGPUClosing(nil)
	closed := false
	closeGPUDevice(NewOffscreen(WindowOptions{Width: 8, Height: 8}),
		&recordingCloser{on: func() { closed = true }})
	if !closed {
		t.Error("the device was not closed")
	}
}
