#include "kde_palette.h"

#include <stddef.h>

// The argument types of every request, in order, as libwayland wants them:
// the interface of each object or new_id argument, NULL for the rest.
static const struct wl_interface *uitk_kde_palette_types[] = {
	&uitk_kde_palette_interface, // create: id
	&wl_surface_interface,       // create: surface
	NULL,                        // set_palette: palette
};

static const struct wl_message uitk_kde_palette_manager_requests[] = {
	{ "create", "no", uitk_kde_palette_types + 0 },
};

static const struct wl_message uitk_kde_palette_requests[] = {
	{ "set_palette", "s", uitk_kde_palette_types + 2 },
	{ "release", "", uitk_kde_palette_types + 2 },
};

const struct wl_interface uitk_kde_palette_manager_interface = {
	"org_kde_kwin_server_decoration_palette_manager", 1,
	1, uitk_kde_palette_manager_requests,
	0, NULL,
};

const struct wl_interface uitk_kde_palette_interface = {
	"org_kde_kwin_server_decoration_palette", 1,
	2, uitk_kde_palette_requests,
	0, NULL,
};

struct wl_proxy *uitk_kde_palette_create(struct wl_proxy *manager, struct wl_surface *surface)
{
	if (!manager || !surface) {
		return NULL;
	}
	return wl_proxy_marshal_flags(manager, 0, &uitk_kde_palette_interface,
		wl_proxy_get_version(manager), 0, NULL, surface);
}

void uitk_kde_palette_set(struct wl_proxy *palette, const char *path)
{
	if (palette && path) {
		wl_proxy_marshal_flags(palette, 0, NULL, wl_proxy_get_version(palette), 0, path);
	}
}

void uitk_kde_palette_release(struct wl_proxy *palette)
{
	if (palette) {
		wl_proxy_marshal_flags(palette, 1, NULL, wl_proxy_get_version(palette), WL_MARSHAL_FLAG_DESTROY);
	}
}
