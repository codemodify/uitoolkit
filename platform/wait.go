package platform

import "time"

// DisplayWaiter can block until the connection has events or timeout elapses.
// Timeout < 0 waits until an event; 0 is a non-blocking poll.
type DisplayWaiter interface {
	Wait(timeout time.Duration) bool
}

// WakeScheduler reports the next time the run loop must wake even if the
// display fd is idle (Wayland key repeat). Zero means no extra wake.
type WakeScheduler interface {
	WakeAt() time.Time
}

// WaitDisplay blocks on s when it implements [DisplayWaiter]. Otherwise it
// sleeps timeout (capped) so headless [Application.Run] can still honor
// caret / tooltip deadlines without spinning at 60 Hz.
func WaitDisplay(s Surface, timeout time.Duration) bool {
	if w, ok := s.(DisplayWaiter); ok {
		return w.Wait(timeout)
	}
	return waitOrSleep(timeout)
}

// DisplayWaker can interrupt a blocking [DisplayWaiter.Wait] so queued
// Application.Post work (SNI ContextMenu / dbusmenu Event) runs without
// waiting out the tray 100ms cap. Wayland polls an eventfd with
// wl_display; X11 uses the helper property.
type DisplayWaker interface {
	Wake()
}

// WakeSurface pokes s so the UI loop leaves Wait. Also signals the
// process-wide [WakeLoop] fd so a tray-only wait (no surface) wakes.
func WakeSurface(s Surface) {
	WakeLoop()
	if s == nil {
		return
	}
	if w, ok := s.(DisplayWaker); ok {
		w.Wake()
	}
}

// SurfaceWakeAt is the next extra wake from s, or zero.
func SurfaceWakeAt(s Surface) time.Time {
	if w, ok := s.(WakeScheduler); ok {
		return w.WakeAt()
	}
	return time.Time{}
}

func waitMillis(d time.Duration) int {
	if d < 0 {
		return -1
	}
	if d == 0 {
		return 0
	}
	ms := int(d / time.Millisecond)
	if ms < 1 {
		return 1
	}
	return ms
}
