#ifndef UITK_KDE_PALETTE_H
#define UITK_KDE_PALETTE_H

#include <wayland-client.h>

/*
 * KWin's server-decoration palette, spoken by hand. The protocol's XML is
 * KDE's and LGPL, so nothing here is generated from it; these are the
 * protocol's facts, which a client needs to speak it at all:
 *
 *   org_kde_kwin_server_decoration_palette_manager, version 1
 *     request 0  create(id: new_id org_kde_kwin_server_decoration_palette,
 *                       surface: object wl_surface)
 *   org_kde_kwin_server_decoration_palette, version 1
 *     request 0  set_palette(palette: string)
 *     request 1  release()                       (destructor)
 *
 * Neither interface has events. The palette string names a KDE colour
 * scheme, which KWin opens as a colour-scheme file.
 */

extern const struct wl_interface uitk_kde_palette_manager_interface;
extern const struct wl_interface uitk_kde_palette_interface;

struct wl_proxy *uitk_kde_palette_create(struct wl_proxy *manager, struct wl_surface *surface);
void uitk_kde_palette_set(struct wl_proxy *palette, const char *path);
void uitk_kde_palette_release(struct wl_proxy *palette);

#endif
