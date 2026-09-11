#ifndef UITK_WAYLAND_SYNC_H
#define UITK_WAYLAND_SYNC_H

#include <stdint.h>
#include <wayland-client.h>
#include "linux-explicit-synchronization-unstable-v1-client-protocol.h"
#include "linux-drm-syncobj-v1-client-protocol.h"

#ifdef __cplusplus
extern "C" {
#endif

const struct wl_interface *ui_wl_explicit_sync_iface(void);
const struct wl_interface *ui_wl_drm_syncobj_iface(void);

struct zwp_linux_surface_synchronization_v1 *ui_wl_explicit_get_sync(
	struct zwp_linux_explicit_synchronization_v1 *m, struct wl_surface *s);
void ui_wl_explicit_sync_destroy(struct zwp_linux_surface_synchronization_v1 *sync);
void ui_wl_explicit_set_acquire(struct zwp_linux_surface_synchronization_v1 *sync, int fd);
void ui_wl_explicit_get_release(struct zwp_linux_surface_synchronization_v1 *sync, uintptr_t sid, int slot);
void ui_wl_explicit_destroy(struct zwp_linux_explicit_synchronization_v1 *m);

struct ui_drm_timeline;

struct ui_drm_timeline *ui_drm_timeline_setup(
	struct wp_linux_drm_syncobj_manager_v1 *man, struct wl_surface *s, int drm_fd);
void ui_drm_timeline_destroy(struct ui_drm_timeline *t);
int ui_drm_timeline_wait_slot(struct ui_drm_timeline *t, int slot);
int ui_drm_timeline_set_points(struct ui_drm_timeline *t, int slot);
void ui_wl_drm_syncobj_destroy(struct wp_linux_drm_syncobj_manager_v1 *m);

#ifdef __cplusplus
}
#endif

#endif
