#ifndef UITK_WAYLAND_DMABUF_H
#define UITK_WAYLAND_DMABUF_H

#include <stddef.h>
#include <stdint.h>
#include <wayland-client.h>
#include "linux-dmabuf-unstable-v1-client-protocol.h"

#ifdef __cplusplus
extern "C" {
#endif

enum {
	UI_DMA_NONE = 0,
	UI_DMA_GBM = 1,
	UI_DMA_HEAP = 2,
	UI_DMA_UDMA = 3
};

struct ui_dmabuf_bo {
	int fd;
	void *mapped;
	size_t map_size;
	/* stride is the buffer object's real row pitch: what the compositor
	   is told in zwp_linux_buffer_params_v1.add. The CPU mapping may be
	   staged with a different pitch, so map_stride is tracked apart. */
	uint32_t stride;
	uint32_t map_stride;
	uint32_t offset;
	uint64_t modifier;
	int kind;
	int w;
	int h;
	void *gbm_bo;
	void *gbm_map_data;
};

const struct wl_interface *ui_wl_dmabuf_iface(void);
void ui_wl_dmabuf_listen(struct zwp_linux_dmabuf_v1 *d, uintptr_t id);
struct zwp_linux_dmabuf_feedback_v1 *ui_wl_dmabuf_feedback(struct zwp_linux_dmabuf_v1 *d, uintptr_t id);
void ui_wl_dmabuf_feedback_destroy(struct zwp_linux_dmabuf_feedback_v1 *f);
void ui_wl_dmabuf_destroy(struct zwp_linux_dmabuf_v1 *d);

struct wl_buffer *ui_wl_dmabuf_buffer(struct zwp_linux_dmabuf_v1 *dmabuf, int fd,
	int w, int h, uint32_t stride, uint32_t offset, uint32_t format,
	uint32_t mod_hi, uint32_t mod_lo);

int ui_dmabuf_alloc(struct ui_dmabuf_bo *out, int w, int h, uint32_t format);
void ui_dmabuf_free(struct ui_dmabuf_bo *bo);
int ui_dmabuf_probe(char *name, int n);
int ui_dmabuf_drm_fd(void);
int ui_dmabuf_cpu_begin(int fd);
int ui_dmabuf_cpu_end(int fd);
int ui_dmabuf_map(struct ui_dmabuf_bo *bo, int w, int h);
void ui_dmabuf_unmap(struct ui_dmabuf_bo *bo);
int ui_dmabuf_export_sync_file(int fd);

#ifdef __cplusplus
}
#endif

#endif
