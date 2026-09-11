//go:build linux && cgo

package platform

/*
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>

extern void uitkXIMStart(uintptr_t win);
extern void uitkXIMDone(uintptr_t win);
extern void uitkXIMDraw(uintptr_t win, int first, int length, char *utf8, int caret);
extern void uitkXIMCaret(uintptr_t win, int caret);

typedef struct {
	XIMCallback start, done, draw, caret;
} uitk_xim_cbs;

static void uitk_utf8_from_xim(XIMText *t, char **out) {
	*out = NULL;
	if (!t) return;
	if (t->encoding_is_wchar && t->string.wide_char) {
		int n = (int)t->length;
		char *buf = (char*)malloc((size_t)n * 4 + 1);
		if (!buf) return;
		int o = 0;
		for (int i = 0; i < n; i++) {
			unsigned int cp = (unsigned int)t->string.wide_char[i];
			if (cp < 0x80) {
				buf[o++] = (char)cp;
			} else if (cp < 0x800) {
				buf[o++] = (char)(0xC0 | (cp >> 6));
				buf[o++] = (char)(0x80 | (cp & 0x3F));
			} else if (cp < 0x10000) {
				buf[o++] = (char)(0xE0 | (cp >> 12));
				buf[o++] = (char)(0x80 | ((cp >> 6) & 0x3F));
				buf[o++] = (char)(0x80 | (cp & 0x3F));
			} else {
				buf[o++] = (char)(0xF0 | (cp >> 18));
				buf[o++] = (char)(0x80 | ((cp >> 12) & 0x3F));
				buf[o++] = (char)(0x80 | ((cp >> 6) & 0x3F));
				buf[o++] = (char)(0x80 | (cp & 0x3F));
			}
		}
		buf[o] = 0;
		*out = buf;
		return;
	}
	if (t->string.multi_byte) {
		*out = strdup(t->string.multi_byte);
	}
}

static int uitk_xim_start(XIC ic, XPointer client, XPointer call) {
	(void)ic; (void)call;
	uitkXIMStart((uintptr_t)client);
	return 0;
}
static void uitk_xim_done(XIC ic, XPointer client, XPointer call) {
	(void)ic; (void)call;
	uitkXIMDone((uintptr_t)client);
}
static void uitk_xim_draw(XIC ic, XPointer client, XPointer call) {
	(void)ic;
	XIMPreeditDrawCallbackStruct *d = (XIMPreeditDrawCallbackStruct*)call;
	char *utf = NULL;
	int caret = 0, first = 0, length = 0;
	if (d) {
		caret = d->caret;
		first = d->chg_first;
		length = d->chg_length;
		if (d->text) uitk_utf8_from_xim(d->text, &utf);
	}
	uitkXIMDraw((uintptr_t)client, first, length, utf ? utf : "", caret);
	free(utf);
}
static void uitk_xim_caret(XIC ic, XPointer client, XPointer call) {
	(void)ic;
	XIMPreeditCaretCallbackStruct *d = (XIMPreeditCaretCallbackStruct*)call;
	int caret = 0;
	if (d) caret = d->position;
	uitkXIMCaret((uintptr_t)client, caret);
}

static XIC ui_create_ic_preedit(XIM im, Window w, void **cbs_out) {
	if (!im || !w) return NULL;
	uitk_xim_cbs *cb = (uitk_xim_cbs*)calloc(1, sizeof(uitk_xim_cbs));
	if (!cb) return NULL;
	cb->start.client_data = (XPointer)(uintptr_t)w;
	cb->start.callback = (XIMProc)uitk_xim_start;
	cb->done.client_data = (XPointer)(uintptr_t)w;
	cb->done.callback = (XIMProc)uitk_xim_done;
	cb->draw.client_data = (XPointer)(uintptr_t)w;
	cb->draw.callback = (XIMProc)uitk_xim_draw;
	cb->caret.client_data = (XPointer)(uintptr_t)w;
	cb->caret.callback = (XIMProc)uitk_xim_caret;
	XVaNestedList preedit = XVaCreateNestedList(0,
		XNPreeditStartCallback, &cb->start,
		XNPreeditDoneCallback, &cb->done,
		XNPreeditDrawCallback, &cb->draw,
		XNPreeditCaretCallback, &cb->caret,
		NULL);
	XIC ic = XCreateIC(im, XNInputStyle, XIMPreeditCallbacks | XIMStatusNothing,
		XNClientWindow, w, XNFocusWindow, w, XNPreeditAttributes, preedit, NULL);
	if (preedit) XFree(preedit);
	if (!ic) {
		free(cb);
		*cbs_out = NULL;
		return XCreateIC(im, XNInputStyle, XIMPreeditNothing | XIMStatusNothing,
			XNClientWindow, w, XNFocusWindow, w, NULL);
	}
	*cbs_out = cb;
	return ic;
}

static void ui_free_xim_cbs(void *p) { free(p); }
*/
import "C"

import "unsafe"

//export uitkXIMStart
func uitkXIMStart(win C.uintptr_t) {
	if x11c == nil {
		return
	}
	s := x11c.surfaces[C.Window(win)]
	if s == nil {
		return
	}
	s.preeditBuf = ""
	x11c.queues[s.win] = append(x11c.queues[s.win], Event{Kind: EventIMEPreedit})
}

//export uitkXIMDone
func uitkXIMDone(win C.uintptr_t) {
	if x11c == nil {
		return
	}
	s := x11c.surfaces[C.Window(win)]
	if s == nil {
		return
	}
	s.preeditBuf = ""
	x11c.queues[s.win] = append(x11c.queues[s.win], Event{Kind: EventIMEPreedit, Text: ""})
}

//export uitkXIMDraw
func uitkXIMDraw(win C.uintptr_t, first, length C.int, utf8 *C.char, caret C.int) {
	if x11c == nil {
		return
	}
	s := x11c.surfaces[C.Window(win)]
	if s == nil {
		return
	}
	ins := ""
	if utf8 != nil {
		ins = C.GoString(utf8)
	}
	s.preeditBuf = ApplyPreeditDraw(s.preeditBuf, int(first), int(length), ins)
	x11c.queues[s.win] = append(x11c.queues[s.win], Event{
		Kind:     EventIMEPreedit,
		Text:     s.preeditBuf,
		IMECaret: int(caret),
	})
}

//export uitkXIMCaret
func uitkXIMCaret(win C.uintptr_t, caret C.int) {
	if x11c == nil {
		return
	}
	s := x11c.surfaces[C.Window(win)]
	if s == nil {
		return
	}
	x11c.queues[s.win] = append(x11c.queues[s.win], Event{
		Kind:     EventIMEPreedit,
		Text:     s.preeditBuf,
		IMECaret: int(caret),
	})
}

func x11CreateIC(im C.XIM, win C.Window) (C.XIC, unsafe.Pointer) {
	var cbs unsafe.Pointer
	ic := C.ui_create_ic_preedit(im, win, &cbs)
	return ic, cbs
}

func x11FreeCbs(p unsafe.Pointer) {
	if p != nil {
		C.ui_free_xim_cbs(p)
	}
}
