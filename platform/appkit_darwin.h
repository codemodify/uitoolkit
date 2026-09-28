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

// uitk_ak_present puts w by h premultiplied RGBA pixels on the window.
void uitk_ak_present(void *win, const unsigned char *pix, int w, int h);

// uitk_ak_pump runs the event queue dry, dispatching to AppKit. Nothing
// here calls back into the toolkit except through uitkAkEvent.
void uitk_ak_pump(void);

// uitk_ak_backing_scale is the window's device pixels per point.
double uitk_ak_backing_scale(void *win);

// uitk_ak_content_size is the content area in *device* pixels.
void uitk_ak_content_size(void *win, int *w, int *h);

#endif
