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
	UITK_AK_MOVE = 8,
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
	// Drag and drop. x and y are the pointer in the window's content
	// area, device pixels y down; `button` carries the toolkit's drag
	// actions the source allows and `key` the one it prefers, or for
	// UITK_AK_DRAG_END the one that was performed.
	UITK_AK_DRAG_MOTION = 8,
	UITK_AK_DRAG_LEAVE = 9,
	UITK_AK_DROP = 10,
	UITK_AK_DRAG_END = 11,
	// Input methods. The text rides in `text`; for a preedit `key` is
	// the caret as a byte offset into it, and `button` the length in
	// bytes of the text replaced before the caret.
	UITK_AK_IME_PREEDIT = 12,
	UITK_AK_IME_COMMIT = 13,
	UITK_AK_IME_CANCEL = 14,
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
// uitk_ak_above is whether the window is at a level above the ordinary
// ones — the state a keep-above caption button draws itself from.
int uitk_ak_above(void *win);
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

// --- popups ----------------------------------------------------------
//
// A popup is a borderless, non-activating panel parented to the window
// it hangs from. Non-activating matters: a menu must not take key away
// from the window under it, and it does not need to — the toolkit's
// contract is that a popup hands every input event to its root, so the
// keyboard going straight there is the answer rather than a problem.
//
// Placement is the toolkit's, not AppKit's. macOS has no positioner
// like xdg_positioner, so SolvePopup does the flipping and sliding (the
// X11 backend's path too) and this is told where to put the window.
//
// x and y are the toolkit's screen coordinates: points, y down from the
// top-left of the main screen.
void *uitk_ak_popup_new(uintptr_t sid, void *parent, int x, int y, int w, int h, int tooltip);
void uitk_ak_popup_move(void *win, int x, int y, int w, int h);
void uitk_ak_popup_close(void *parent, void *win);

// uitk_ak_content_origin is where the window's *content* area begins, in
// the toolkit's screen coordinates — the origin a popup's placement is
// relative to.
void uitk_ak_content_origin(void *win, int *x, int *y);

// uitk_ak_work_area is the usable area of the screen the window is on,
// in the toolkit's screen coordinates: the screen less the menu bar and
// the Dock, which is where a popup may go.
void uitk_ak_work_area(void *win, int *x, int *y, int *w, int *h);

// --- drag and drop ---------------------------------------------------
//
// Types cross this boundary as MIME strings, which is the toolkit's
// vocabulary, and the Objective-C side maps the two macOS knows —
// text and file URLs — onto their pasteboard types. Anything else is
// put on the pasteboard under the MIME string itself: NSPasteboard
// takes an arbitrary type identifier, so a toolkit-specific type
// travels between two uitoolkit windows without a registry.
//
// Drag actions are the toolkit's bitset (DragCopy 1, DragMove 2,
// DragLink 4), not NSDragOperation, for the same reason the event
// kinds are not platform.EventKind: the boundary is a bad place to
// depend on the value of a constant from either side.

// uitk_ak_register_drops makes the window a drop target.
void uitk_ak_register_drops(void *win);

// uitk_ak_drop_types is the MIME types the drag now over the window
// offers, NUL-separated; the caller frees it. n is set to the total
// length, so the Go side can take the whole block in one piece and
// split it in Go, where the splitting can be tested. NULL when no drag
// is in hand.
char *uitk_ak_drop_types(void *win, int *n);

// uitk_ak_drop_data is one type's bytes from the drag now over the
// window; the caller frees it. n is set to the length, 0 for nothing.
void *uitk_ak_drop_data(void *win, const char *mime, int *n);

// uitk_ak_set_drop_answer records what the target would do with the
// drag, which -draggingUpdated: answers with.
void uitk_ak_set_drop_answer(void *win, int action);

// A drag's payload is built up and then started. uitk_ak_drag_new
// makes one, _add puts a type on it, _start begins the session from
// the press being handled and _free releases an unstarted one.
void *uitk_ak_drag_new(void);
void uitk_ak_drag_add(void *item, const char *mime, const void *bytes, int n);
int uitk_ak_drag_start(void *win, void *item, const unsigned char *icon, int iw, int ih,
                       double hotX, double hotY, int actions);
void uitk_ak_drag_free(void *item);

// uitk_ak_dragging reports whether a drag started from this window is
// still running.
int uitk_ak_dragging(void *win);

// --- input methods ---------------------------------------------------
//
// uitk_ak_set_ime_enabled activates or deactivates the view's input
// context, which is as near as macOS comes to the enable/disable
// zwp_text_input_v3 has: there is no per-window switch, only which
// responder the input context is serving.
void uitk_ak_set_ime_enabled(void *win, int on);

// uitk_ak_set_ime_cursor says where the text caret is, in device pixels
// of the content area — where the candidate window goes.
void uitk_ak_set_ime_cursor(void *win, int x, int y, int w, int h);

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

// uitk_ak_ime_simulate drives the view's NSTextInputClient directly:
// 1 marks text with a caret at a UTF-16 offset, 2 inserts it, 3
// unmarks. A real input method cannot be scripted, and this is the
// path it would take.
void uitk_ak_ime_simulate(void *win, int what, const char *text, int caret);

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
void uitk_ak_set_role(void *w, int dialog);
void uitk_ak_center(void *w);
int uitk_ak_activate(void *w);
