// inj — inject pointer/keyboard input into the nested KWin via org_kde_kwin_fake_input.
// usage: inj CMD... where CMD is one of
//   move X Y        absolute pointer motion (logical px)
//   click [BTN]     press+release (BTN: left|right|middle|back|forward, default left)
//   down [BTN] / up [BTN]
//   wheel DY        vertical axis (positive = down), in 15-unit clicks
//   key CODE        evdev keycode press+release; key+ CODE / key- CODE for hold
//   sleep MS
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <wayland-client.h>
#include "fake-input-client.h"

static struct org_kde_kwin_fake_input *fi;

static void reg_global(void *d, struct wl_registry *r, uint32_t name, const char *iface, uint32_t ver) {
	if (strcmp(iface, org_kde_kwin_fake_input_interface.name) == 0)
		fi = wl_registry_bind(r, name, &org_kde_kwin_fake_input_interface, ver < 4 ? ver : 4);
}
static void reg_remove(void *d, struct wl_registry *r, uint32_t name) {}
static const struct wl_registry_listener rl = {reg_global, reg_remove};

static uint32_t btn(const char *s) {
	if (!s || !strcmp(s, "left")) return 0x110;
	if (!strcmp(s, "right")) return 0x111;
	if (!strcmp(s, "middle")) return 0x112;
	if (!strcmp(s, "back")) return 0x113;    /* BTN_SIDE */
	if (!strcmp(s, "forward")) return 0x114; /* BTN_EXTRA */
	return 0x110;
}

int main(int argc, char **argv) {
	struct wl_display *dpy = wl_display_connect(NULL);
	if (!dpy) { fprintf(stderr, "inj: no wayland display\n"); return 1; }
	struct wl_registry *reg = wl_display_get_registry(dpy);
	wl_registry_add_listener(reg, &rl, NULL);
	wl_display_roundtrip(dpy);
	if (!fi) { fprintf(stderr, "inj: no fake_input global\n"); return 1; }
	org_kde_kwin_fake_input_authenticate(fi, "uitk-e2e", "end to end test");
	wl_display_roundtrip(dpy);
	for (int i = 1; i < argc; i++) {
		const char *c = argv[i];
		if (!strcmp(c, "move") && i + 2 < argc) {
			org_kde_kwin_fake_input_pointer_motion_absolute(fi, wl_fixed_from_double(atof(argv[i + 1])), wl_fixed_from_double(atof(argv[i + 2])));
			i += 2;
		} else if (!strcmp(c, "click") || !strcmp(c, "down") || !strcmp(c, "up")) {
			const char *b = NULL;
			if (i + 1 < argc && (!strcmp(argv[i + 1], "left") || !strcmp(argv[i + 1], "right") || !strcmp(argv[i + 1], "middle") ||
			                      !strcmp(argv[i + 1], "back") || !strcmp(argv[i + 1], "forward"))) b = argv[++i];
			if (strcmp(c, "up")) org_kde_kwin_fake_input_button(fi, btn(b), 1);
			if (!strcmp(c, "click")) { wl_display_roundtrip(dpy); usleep(30000); }
			if (strcmp(c, "down")) org_kde_kwin_fake_input_button(fi, btn(b), 0);
		} else if (!strcmp(c, "wheel") && i + 1 < argc) {
			org_kde_kwin_fake_input_axis(fi, 0, wl_fixed_from_double(atof(argv[++i]) * 15.0));
		} else if (!strcmp(c, "key") && i + 1 < argc) {
			uint32_t k = atoi(argv[++i]);
			org_kde_kwin_fake_input_keyboard_key(fi, k, 1);
			wl_display_roundtrip(dpy); usleep(20000);
			org_kde_kwin_fake_input_keyboard_key(fi, k, 0);
		} else if (!strcmp(c, "key+") && i + 1 < argc) {
			org_kde_kwin_fake_input_keyboard_key(fi, atoi(argv[++i]), 1);
		} else if (!strcmp(c, "key-") && i + 1 < argc) {
			org_kde_kwin_fake_input_keyboard_key(fi, atoi(argv[++i]), 0);
		} else if (!strcmp(c, "sleep") && i + 1 < argc) {
			wl_display_roundtrip(dpy);
			usleep(atoi(argv[++i]) * 1000);
		} else {
			fprintf(stderr, "inj: bad command %s\n", c);
			return 2;
		}
		wl_display_roundtrip(dpy);
	}
	wl_display_roundtrip(dpy);
	wl_display_disconnect(dpy);
	return 0;
}
