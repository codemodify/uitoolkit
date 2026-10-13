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

// bindingCloser is a device with a context to bind, as a real GPU
// device has.
type bindingCloser struct {
	recordingCloser
	bind func()
	err  error
}

func (c *bindingCloser) MakeCurrent() error {
	if c.bind != nil {
		c.bind()
	}
	return c.err
}

// **The closing device's context is made current before the hook runs.**
//
// The hook's whole promise is "your device, your context, now" — and in
// a process with two GPU windows what was actually current was whatever
// the last thing painted left there. media-player-video reproduced the
// callback running with the *other* window's context, which is exactly
// what libmpv's render API refuses: a renderer must be destroyed against
// the context it was made against.
func TestTheClosingDeviceIsBoundBeforeTheHookRuns(t *testing.T) {
	was := closingGPU
	t.Cleanup(func() { closingGPU = was })

	var order []string
	SetGPUClosing(func(Surface) { order = append(order, "told") })
	surf := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	dev := &bindingCloser{
		recordingCloser: recordingCloser{on: func() { order = append(order, "closed") }},
		bind:            func() { order = append(order, "bound") },
	}
	closeGPUDevice(surf, dev)

	want := []string{"bound", "told", "closed"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Errorf("the order was %v, want %v", order, want)
	}
}

// A device whose context has already gone is a device that cannot be
// made current, and the application is still told: the notice is what
// lets it drop a renderer it can no longer use.
func TestAHookStillRunsForADeviceThatCannotBeBound(t *testing.T) {
	was := closingGPU
	t.Cleanup(func() { closingGPU = was })

	told := false
	SetGPUClosing(func(Surface) { told = true })
	surf := NewOffscreen(WindowOptions{Width: 10, Height: 10})
	closeGPUDevice(surf, &bindingCloser{err: errors.New("no context")})
	if !told {
		t.Error("a device that refused to bind swallowed the application's notice")
	}
}

// GPUClosing reads back what is installed, so a program that installs a
// hook of its own can chain to what was there.
func TestTheInstalledHookCanBeReadBack(t *testing.T) {
	was := closingGPU
	t.Cleanup(func() { closingGPU = was })

	SetGPUClosing(nil)
	if GPUClosing() != nil {
		t.Error("a cleared hook reads back as something")
	}
	ran := false
	SetGPUClosing(func(Surface) { ran = true })
	got := GPUClosing()
	if got == nil {
		t.Fatal("the installed hook reads back as nothing")
	}
	got(nil)
	if !ran {
		t.Error("what was read back is not what was installed")
	}
}
