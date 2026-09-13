#define _GNU_SOURCE
#include "wayland_sync.h"

#include <drm/drm.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <unistd.h>

extern void uitkWlExplicitRelease(uintptr_t sid, int slot);

const struct wl_interface *ui_wl_explicit_sync_iface(void) {
	return &zwp_linux_explicit_synchronization_v1_interface;
}

const struct wl_interface *ui_wl_drm_syncobj_iface(void) {
	return &wp_linux_drm_syncobj_manager_v1_interface;
}

struct zwp_linux_surface_synchronization_v1 *ui_wl_explicit_get_sync(
	struct zwp_linux_explicit_synchronization_v1 *m, struct wl_surface *s) {
	if (!m || !s) return NULL;
	return zwp_linux_explicit_synchronization_v1_get_synchronization(m, s);
}

void ui_wl_explicit_sync_destroy(struct zwp_linux_surface_synchronization_v1 *sync) {
	if (sync) zwp_linux_surface_synchronization_v1_destroy(sync);
}

void ui_wl_explicit_set_acquire(struct zwp_linux_surface_synchronization_v1 *sync, int fd) {
	if (!sync || fd < 0) return;
	zwp_linux_surface_synchronization_v1_set_acquire_fence(sync, fd);
}

static void uitk_rel_fenced(void *data, struct zwp_linux_buffer_release_v1 *rel, int32_t fence) {
	uintptr_t packed = (uintptr_t)data;
	int slot = (int)(packed & 0xff);
	uintptr_t sid = packed >> 8;
	if (fence >= 0) close(fence);
	(void)rel;
	uitkWlExplicitRelease(sid, slot);
}

static void uitk_rel_immediate(void *data, struct zwp_linux_buffer_release_v1 *rel) {
	uintptr_t packed = (uintptr_t)data;
	int slot = (int)(packed & 0xff);
	uintptr_t sid = packed >> 8;
	(void)rel;
	uitkWlExplicitRelease(sid, slot);
}

static const struct zwp_linux_buffer_release_v1_listener uitk_rel_listener = {
	.fenced_release = uitk_rel_fenced,
	.immediate_release = uitk_rel_immediate,
};

void ui_wl_explicit_get_release(struct zwp_linux_surface_synchronization_v1 *sync, uintptr_t sid, int slot) {
	if (!sync) return;
	struct zwp_linux_buffer_release_v1 *rel = zwp_linux_surface_synchronization_v1_get_release(sync);
	if (!rel) return;
	uintptr_t packed = (sid << 8) | (uintptr_t)(slot & 0xff);
	zwp_linux_buffer_release_v1_add_listener(rel, &uitk_rel_listener, (void *)packed);
}

void ui_wl_explicit_destroy(struct zwp_linux_explicit_synchronization_v1 *m) {
	if (m) zwp_linux_explicit_synchronization_v1_destroy(m);
}

/* UI_DMA_SLOTS must match the Go wlSurface.slots array. */
#define UI_DMA_SLOTS 4

struct ui_drm_timeline {
	int drm_fd;
	uint32_t handle;
	uint64_t point;
	uint64_t slot_rel[UI_DMA_SLOTS];
	struct wp_linux_drm_syncobj_timeline_v1 *tl;
	struct wp_linux_drm_syncobj_surface_v1 *surf;
};

static int syncobj_create(int drm_fd, uint32_t *handle) {
	struct drm_syncobj_create args = {0};
	if (ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_CREATE, &args) != 0) return -1;
	*handle = args.handle;
	return 0;
}

static int syncobj_to_fd(int drm_fd, uint32_t handle) {
	struct drm_syncobj_handle args = { .handle = handle };
	if (ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_HANDLE_TO_FD, &args) != 0) return -1;
	return args.fd;
}

static int syncobj_signal(int drm_fd, uint32_t handle, uint64_t point) {
	uint32_t h = handle;
	uint64_t p = point;
	struct drm_syncobj_timeline_array args = {
		.handles = (uintptr_t)&h,
		.points = (uintptr_t)&p,
		.count_handles = 1,
	};
	return ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_TIMELINE_SIGNAL, &args);
}

static int syncobj_wait(int drm_fd, uint32_t handle, uint64_t point) {
	uint32_t h = handle;
	uint64_t p = point;
	struct drm_syncobj_timeline_wait args = {
		.handles = (uintptr_t)&h,
		.points = (uintptr_t)&p,
		.timeout_nsec = 50LL * 1000 * 1000, /* 50ms */
		.count_handles = 1,
		.flags = DRM_SYNCOBJ_WAIT_FLAGS_WAIT_FOR_SUBMIT,
	};
	return ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_TIMELINE_WAIT, &args);
}

struct ui_drm_timeline *ui_drm_timeline_setup(
	struct wp_linux_drm_syncobj_manager_v1 *man, struct wl_surface *s, int drm_fd) {
	if (!man || !s || drm_fd < 0) return NULL;
	struct ui_drm_timeline *t = calloc(1, sizeof(*t));
	if (!t) return NULL;
	t->drm_fd = drm_fd;
	if (syncobj_create(drm_fd, &t->handle) != 0) {
		free(t);
		return NULL;
	}
	int fd = syncobj_to_fd(drm_fd, t->handle);
	if (fd < 0) {
		struct drm_syncobj_destroy d = { .handle = t->handle };
		ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_DESTROY, &d);
		free(t);
		return NULL;
	}
	t->tl = wp_linux_drm_syncobj_manager_v1_import_timeline(man, fd);
	close(fd);
	if (!t->tl) {
		struct drm_syncobj_destroy d = { .handle = t->handle };
		ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_DESTROY, &d);
		free(t);
		return NULL;
	}
	t->surf = wp_linux_drm_syncobj_manager_v1_get_surface(man, s);
	if (!t->surf) {
		wp_linux_drm_syncobj_timeline_v1_destroy(t->tl);
		struct drm_syncobj_destroy d = { .handle = t->handle };
		ioctl(drm_fd, DRM_IOCTL_SYNCOBJ_DESTROY, &d);
		free(t);
		return NULL;
	}
	return t;
}

void ui_drm_timeline_destroy(struct ui_drm_timeline *t) {
	if (!t) return;
	if (t->surf) wp_linux_drm_syncobj_surface_v1_destroy(t->surf);
	if (t->tl) wp_linux_drm_syncobj_timeline_v1_destroy(t->tl);
	if (t->drm_fd >= 0 && t->handle) {
		struct drm_syncobj_destroy d = { .handle = t->handle };
		ioctl(t->drm_fd, DRM_IOCTL_SYNCOBJ_DESTROY, &d);
	}
	free(t);
}

int ui_drm_timeline_wait_slot(struct ui_drm_timeline *t, int slot) {
	if (!t || slot < 0 || slot >= UI_DMA_SLOTS) return 0;
	if (t->slot_rel[slot] == 0) return 0;
	return syncobj_wait(t->drm_fd, t->handle, t->slot_rel[slot]);
}

int ui_drm_timeline_set_points(struct ui_drm_timeline *t, int slot) {
	if (!t || !t->surf || !t->tl || slot < 0 || slot >= UI_DMA_SLOTS) return -1;
	t->point++;
	uint64_t acq = t->point;
	t->point++;
	uint64_t rel = t->point;
	if (syncobj_signal(t->drm_fd, t->handle, acq) != 0) return -1;
	wp_linux_drm_syncobj_surface_v1_set_acquire_point(t->surf, t->tl,
		(uint32_t)(acq >> 32), (uint32_t)acq);
	wp_linux_drm_syncobj_surface_v1_set_release_point(t->surf, t->tl,
		(uint32_t)(rel >> 32), (uint32_t)rel);
	t->slot_rel[slot] = rel;
	return 0;
}

void ui_wl_drm_syncobj_destroy(struct wp_linux_drm_syncobj_manager_v1 *m) {
	if (m) wp_linux_drm_syncobj_manager_v1_destroy(m);
}
