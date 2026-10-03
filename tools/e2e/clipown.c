// clipown — another program's copy, for the toolkit's clipboard tests.
//
// It takes the X11 CLIPBOARD selection and serves TEXT on it until it is
// killed or SECONDS run out, which is the one thing a test cannot do from
// inside the process under test: a second owner has to be a second client.
// (xclip and xsel do this, and are not installed everywhere; this is 90
// lines and is built with the rest of the rig.)
//
// It prints "owned" and flushes as soon as the server has acknowledged the
// ownership, so the test can wait for that rather than sleep.
//
// usage: clipown TEXT SECONDS
#define _POSIX_C_SOURCE 199309L
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

int main(int argc, char **argv) {
	if (argc < 3) {
		fprintf(stderr, "usage: clipown TEXT SECONDS\n");
		return 2;
	}
	const char *text = argv[1];
	int len = (int)strlen(text);
	double secs = atof(argv[2]);

	Display *d = XOpenDisplay(NULL);
	if (!d) {
		fprintf(stderr, "clipown: no display\n");
		return 1;
	}
	Atom clipboard = XInternAtom(d, "CLIPBOARD", False);
	Atom utf8 = XInternAtom(d, "UTF8_STRING", False);
	Atom targets = XInternAtom(d, "TARGETS", False);
	Window w = XCreateSimpleWindow(d, DefaultRootWindow(d), 0, 0, 1, 1, 0, 0, 0);
	XSelectInput(d, w, PropertyChangeMask);
	XSetSelectionOwner(d, clipboard, w, CurrentTime);
	if (XGetSelectionOwner(d, clipboard) != w) {
		fprintf(stderr, "clipown: the server did not give us CLIPBOARD\n");
		return 1;
	}
	XFlush(d);
	printf("owned\n");
	fflush(stdout);

	// A deadline rather than a loop with no end: a test that fails must not
	// leave an owner of the desktop's clipboard behind.
	struct timespec t0;
	clock_gettime(CLOCK_MONOTONIC, &t0);
	for (;;) {
		struct timespec now;
		clock_gettime(CLOCK_MONOTONIC, &now);
		double el = (now.tv_sec - t0.tv_sec) + (now.tv_nsec - t0.tv_nsec) / 1e9;
		if (el >= secs) break;
		while (XPending(d)) {
			XEvent e;
			XNextEvent(d, &e);
			if (e.type == SelectionClear) {
				// Somebody else took it: our job is done.
				XCloseDisplay(d);
				return 0;
			}
			if (e.type != SelectionRequest) continue;
			XSelectionRequestEvent *r = &e.xselectionrequest;
			XSelectionEvent a;
			memset(&a, 0, sizeof a);
			a.type = SelectionNotify;
			a.display = r->display;
			a.requestor = r->requestor;
			a.selection = r->selection;
			a.target = r->target;
			a.time = r->time;
			a.property = None;
			if (r->target == targets) {
				Atom list[3] = {targets, utf8, XA_STRING};
				XChangeProperty(d, r->requestor, r->property, XA_ATOM, 32,
					PropModeReplace, (unsigned char *)list, 3);
				a.property = r->property;
			} else if (r->target == utf8 || r->target == XA_STRING) {
				XChangeProperty(d, r->requestor, r->property, r->target, 8,
					PropModeReplace, (const unsigned char *)text, len);
				a.property = r->property;
			}
			XSendEvent(d, r->requestor, False, 0, (XEvent *)&a);
			XFlush(d);
		}
		struct timespec nap = {0, 5 * 1000 * 1000}; // 5ms
		nanosleep(&nap, NULL);
	}
	XCloseDisplay(d);
	return 0;
}
