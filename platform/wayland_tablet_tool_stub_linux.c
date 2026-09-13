/* Satisfies cursor-shape-v1's get_tablet_tool_v2 type slot.
 * uitoolkit does not bind tablet-v2; the symbol is required to link. */

#include "wayland-util.h"

const struct wl_interface zwp_tablet_tool_v2_interface = {
	"zwp_tablet_tool_v2",
	1,
	0,
	NULL,
	0,
	NULL,
};
