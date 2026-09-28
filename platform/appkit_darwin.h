//go:build darwin && cgo

// The C face of the AppKit backend. Everything Objective-C lives behind
// it, so the Go side never has to know what a selector is.
#ifndef UITK_APPKIT_DARWIN_H
#define UITK_APPKIT_DARWIN_H

#include <stdint.h>

// Event kinds, matching uitkAkEvent's switch on the Go side. They are
// not platform.EventKind: the boundary between C and Go is a bad place
// to depend on the value of a Go constant.
enum {
	UITK_AK_CLOSE = 1,
	UITK_AK_RESIZE = 2,
	UITK_AK_EXPOSE = 3,
	UITK_AK_SCALE = 4,
	UITK_AK_FOCUS_IN = 5,
	UITK_AK_FOCUS_OUT = 6,
	UITK_AK_STATE = 7,
};

// Input kinds, for uitkAkInput. Separate from the list above because
// they carry a position, a button and modifiers and those would be dead
// weight on a resize.
enum {
	UITK_AK_MOUSE_DOWN = 1,
	UITK_AK_MOUSE_UP = 2,
	UITK_AK_MOUSE_MOVE = 3,
	UITK_AK_SCROLL = 4,
	UITK_AK_KEY_DOWN = 5,
	UITK_AK_KEY_UP = 6,
	UITK_AK_POINTER_LEAVE = 7,
};

// uitk_ak_init brings NSApplication up. Safe to call more than once, and
// it must be called on the process's first thread.
void uitk_ak_init(void);

// uitk_ak_window_new makes a window whose content area is w by h points,
// and returns the NSWindow. sid is the Go-side surface id handed back
// with every event from it.
void *uitk_ak_window_new(uintptr_t sid, const char *title, int w, int h);

void uitk_ak_window_show(void *win);
void uitk_ak_window_close(void *win);
void uitk_ak_set_title(void *win, const char *title);

// --- geometry -------------------------------------------------------
//
// Points, and y down from the top-left of the main screen, which is the
// toolkit's convention and not AppKit's: Cocoa measures y up from the
// bottom-left. The flip happens here rather than in Go so that only one
// file has to know which way is up.
void uitk_ak_move(void *win, int x, int y);
int uitk_ak_position(void *win, int *x, int *y);
void uitk_ak_order_front(void *win, int key);
void uitk_ak_order_out(void *win);
void uitk_ak_order_back(void *win);
int uitk_ak_visible(void *win);
// uitk_ak_set_limits pins the content size in points. A max of 0 means
// no limit; resizable adds or removes the resize control.
void uitk_ak_set_limits(void *win, int minW, int minH, int maxW, int maxH, int resizable);

// --- frame ----------------------------------------------------------
void uitk_ak_miniaturize(void *win);
int uitk_ak_set_zoomed(void *win, int on);
int uitk_ak_zoomed(void *win);
int uitk_ak_miniaturized(void *win);
int uitk_ak_fullscreen(void *win);
void uitk_ak_set_fullscreen(void *win, int on);
void uitk_ak_set_level(void *win, int above);
int uitk_ak_active(void *win);
// uitk_ak_set_decorations: 0 the desktop's frame, 1 none at all. A
// borderless window is also told to stop drawing its own shadow and
// background, because a frame the toolkit draws brings both.
void uitk_ak_set_decorations(void *win, int borderless);
// uitk_ak_start_move hands the press being handled to AppKit for an
// interactive drag; 0 when there is no press to hand over.
int uitk_ak_start_move(void *win);
// uitk_ak_set_app_icon sets the *application's* icon from w by h
// premultiplied RGBA pixels: macOS has no per-window icon.
void uitk_ak_set_app_icon(const unsigned char *pix, int w, int h);

// uitk_ak_set_cursor remembers the pointer shape the content area
// should have, so -[UitkView cursorUpdate:] can put it back every time
// AppKit asks. kind is what darwinCursorKind returns (cursor.go).
void uitk_ak_set_cursor(void *win, int kind);

// uitk_ak_present puts w by h premultiplied RGBA pixels on the window.
void uitk_ak_present(void *win, const unsigned char *pix, int w, int h);

// uitk_ak_pump runs the event queue dry, dispatching to AppKit. Nothing
// here calls back into the toolkit except through uitkAkEvent.
void uitk_ak_pump(void);

// uitk_ak_backing_scale is the window's device pixels per point.
double uitk_ak_backing_scale(void *win);

// uitk_ak_content_size is the content area in *device* pixels.
void uitk_ak_content_size(void *win, int *w, int *h);

// --- verification ---------------------------------------------------
//
// uitk_ak_readback renders the window's own layer tree into out, which
// must hold w*h premultiplied RGBA pixels, and returns 1 on success.
//
// It exists because on macOS there is no other way for a test to see
// what a window is showing. Reading the screen back wants Screen
// Recording permission, which a test run over ssh does not have and
// macOS 15 offers no unprivileged way around: CGWindowListCreateImage
// is obsoleted and ScreenCaptureKit wants the same grant. Worse,
// screencapture(1) does not fail without it — it returns a clean
// desktop with every window silently missing, which reads exactly like
// a window that never opened.
//
// A layer render needs no permission, and it tests what a present can
// actually get wrong: that the buffer reached the layer, with its
// channels in the right order and its geometry the right way up.
int uitk_ak_readback(void *win, unsigned char *out, int w, int h);

// uitk_ak_post_key and uitk_ak_post_mouse put a synthetic event in the
// application's own queue, for a test that needs to know input arrives.
// Posting to one's own queue needs no Accessibility grant, unlike
// driving the system pointer with CGEvent.
//
// x and y for the mouse are AppKit's own window coordinates: points,
// y *up* from the bottom-left, which is what an NSEvent carries. They
// are deliberately not the toolkit's convention — a poster that undid
// what -[UitkView where:] does would make a test of where a press lands
// a round trip through one function, which passes however wrong that
// function is. It did, and the window was upside down anyway.
void uitk_ak_post_key(void *win, const char *chars, const char *bare,
                      uint64_t mods, int down);
void uitk_ak_post_mouse(void *win, int kind, double x, double y,
                        int button, uint64_t mods);

#endif
