#define _GNU_SOURCE
#include "wayland_dmabuf.h"

#include <dlfcn.h>
#include <errno.h>
#include <fcntl.h>
#include <linux/dma-heap.h>
#include <linux/udmabuf.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <unistd.h>

#ifndef MFD_CLOEXEC
#define MFD_CLOEXEC 0x0001U
#endif
#ifndef MFD_ALLOW_SEALING
#define MFD_ALLOW_SEALING 0x0002U
#endif
#ifndef F_ADD_SEALS
#define F_ADD_SEALS 1033
#endif
#ifndef F_SEAL_SHRINK
#define F_SEAL_SHRINK 0x0002
#endif

#ifndef GBM_BO_USE_RENDERING
#define GBM_BO_USE_RENDERING (1 << 2)
#endif
#ifndef GBM_BO_USE_LINEAR
#define GBM_BO_USE_LINEAR (1 << 4)
#endif
#ifndef GBM_BO_TRANSFER_READ_WRITE
#define GBM_BO_TRANSFER_READ_WRITE 3
#endif

extern void uitkWlDmabufFormat(uintptr_t id, uint32_t format);
extern void uitkWlDmabufModifier(uintptr_t id, uint32_t format, uint32_t hi, uint32_t lo);
extern void uitkWlDmabufFeedbackDone(uintptr_t id);
extern void uitkWlDmabufFormatTable(uintptr_t id, void *map, uint32_t size);
extern void uitkWlDmabufTrancheFormats(uintptr_t id, const uint16_t *idx, int n);

const struct wl_interface *ui_wl_dmabuf_iface(void) { return &zwp_linux_dmabuf_v1_interface; }

static void uitk_dmabuf_fmt(void *data, struct zwp_linux_dmabuf_v1 *d, uint32_t format) {
	(void)d;
	uitkWlDmabufFormat((uintptr_t)data, format);
}
static void uitk_dmabuf_mod(void *data, struct zwp_linux_dmabuf_v1 *d, uint32_t format, uint32_t hi, uint32_t lo) {
	(void)d;
	uitkWlDmabufModifier((uintptr_t)data, format, hi, lo);
}
static const struct zwp_linux_dmabuf_v1_listener uitk_dmabuf_listener = {
	.format = uitk_dmabuf_fmt,
	.modifier = uitk_dmabuf_mod,
};

void ui_wl_dmabuf_listen(struct zwp_linux_dmabuf_v1 *d, uintptr_t id) {
	if (d) zwp_linux_dmabuf_v1_add_listener(d, &uitk_dmabuf_listener, (void *)id);
}

static void uitk_fb_done(void *data, struct zwp_linux_dmabuf_feedback_v1 *f) {
	(void)f;
	uitkWlDmabufFeedbackDone((uintptr_t)data);
}
static void uitk_fb_table(void *data, struct zwp_linux_dmabuf_feedback_v1 *f, int32_t fd, uint32_t size) {
	(void)f;
	if (fd < 0 || size == 0) {
		if (fd >= 0) close(fd);
		return;
	}
	void *p = mmap(NULL, size, PROT_READ, MAP_PRIVATE, fd, 0);
	close(fd);
	if (p == MAP_FAILED) return;
	uitkWlDmabufFormatTable((uintptr_t)data, p, size);
	munmap(p, size);
}
static void uitk_fb_main(void *data, struct zwp_linux_dmabuf_feedback_v1 *f, struct wl_array *dev) {
	(void)data; (void)f; (void)dev;
}
static void uitk_fb_tranche_done(void *data, struct zwp_linux_dmabuf_feedback_v1 *f) {
	(void)data; (void)f;
}
static void uitk_fb_tranche_dev(void *data, struct zwp_linux_dmabuf_feedback_v1 *f, struct wl_array *dev) {
	(void)data; (void)f; (void)dev;
}
static void uitk_fb_tranche_fmt(void *data, struct zwp_linux_dmabuf_feedback_v1 *f, struct wl_array *indices) {
	(void)f;
	if (!indices || !indices->data || indices->size == 0) return;
	int n = (int)(indices->size / sizeof(uint16_t));
	uitkWlDmabufTrancheFormats((uintptr_t)data, (const uint16_t *)indices->data, n);
}
static void uitk_fb_flags(void *data, struct zwp_linux_dmabuf_feedback_v1 *f, uint32_t flags) {
	(void)data; (void)f; (void)flags;
}
static const struct zwp_linux_dmabuf_feedback_v1_listener uitk_fb_listener = {
	.done = uitk_fb_done,
	.format_table = uitk_fb_table,
	.main_device = uitk_fb_main,
	.tranche_done = uitk_fb_tranche_done,
	.tranche_target_device = uitk_fb_tranche_dev,
	.tranche_formats = uitk_fb_tranche_fmt,
	.tranche_flags = uitk_fb_flags,
};

struct zwp_linux_dmabuf_feedback_v1 *ui_wl_dmabuf_feedback(struct zwp_linux_dmabuf_v1 *d, uintptr_t id) {
	if (!d || zwp_linux_dmabuf_v1_get_version(d) < 4) return NULL;
	struct zwp_linux_dmabuf_feedback_v1 *fb = zwp_linux_dmabuf_v1_get_default_feedback(d);
	if (fb) zwp_linux_dmabuf_feedback_v1_add_listener(fb, &uitk_fb_listener, (void *)id);
	return fb;
}

void ui_wl_dmabuf_feedback_destroy(struct zwp_linux_dmabuf_feedback_v1 *f) {
	if (f) zwp_linux_dmabuf_feedback_v1_destroy(f);
}

void ui_wl_dmabuf_destroy(struct zwp_linux_dmabuf_v1 *d) {
	if (d) zwp_linux_dmabuf_v1_destroy(d);
}

struct wl_buffer *ui_wl_dmabuf_buffer(struct zwp_linux_dmabuf_v1 *dmabuf, int fd,
	int w, int h, uint32_t stride, uint32_t offset, uint32_t format,
	uint32_t mod_hi, uint32_t mod_lo) {
	if (!dmabuf || fd < 0 || w < 1 || h < 1) return NULL;
	if (zwp_linux_dmabuf_v1_get_version(dmabuf) < 2) return NULL;
	struct zwp_linux_buffer_params_v1 *p = zwp_linux_dmabuf_v1_create_params(dmabuf);
	if (!p) return NULL;
	zwp_linux_buffer_params_v1_add(p, fd, 0, offset, stride, mod_hi, mod_lo);
	struct wl_buffer *buf = zwp_linux_buffer_params_v1_create_immed(p, w, h, format, 0);
	zwp_linux_buffer_params_v1_destroy(p);
	return buf;
}

/* ---------- local dmabuf allocation (GBM / dma-heap / udmabuf) ---------- */

static void *g_gbm_lib;
static int g_drm_fd = -1;
static void *g_gbm_dev;
static int g_gbm_refs;

static void *(*p_gbm_create_device)(int);
static void (*p_gbm_device_destroy)(void *);
static void *(*p_gbm_bo_create)(void *, uint32_t, uint32_t, uint32_t, uint32_t);
static void (*p_gbm_bo_destroy)(void *);
static int (*p_gbm_bo_get_fd)(void *);
static uint32_t (*p_gbm_bo_get_stride)(void *);
static uint32_t (*p_gbm_bo_get_offset)(void *, int);
static uint64_t (*p_gbm_bo_get_modifier)(void *);
static void *(*p_gbm_bo_map)(void *, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t *, void **);
static void (*p_gbm_bo_unmap)(void *, void *);

static int load_gbm(void) {
	if (g_gbm_lib) return 1;
	g_gbm_lib = dlopen("libgbm.so.1", RTLD_LAZY | RTLD_LOCAL);
	if (!g_gbm_lib) g_gbm_lib = dlopen("libgbm.so", RTLD_LAZY | RTLD_LOCAL);
	if (!g_gbm_lib) return 0;
	p_gbm_create_device = dlsym(g_gbm_lib, "gbm_create_device");
	p_gbm_device_destroy = dlsym(g_gbm_lib, "gbm_device_destroy");
	p_gbm_bo_create = dlsym(g_gbm_lib, "gbm_bo_create");
	p_gbm_bo_destroy = dlsym(g_gbm_lib, "gbm_bo_destroy");
	p_gbm_bo_get_fd = dlsym(g_gbm_lib, "gbm_bo_get_fd");
	p_gbm_bo_get_stride = dlsym(g_gbm_lib, "gbm_bo_get_stride");
	p_gbm_bo_get_offset = dlsym(g_gbm_lib, "gbm_bo_get_offset");
	p_gbm_bo_get_modifier = dlsym(g_gbm_lib, "gbm_bo_get_modifier");
	p_gbm_bo_map = dlsym(g_gbm_lib, "gbm_bo_map");
	p_gbm_bo_unmap = dlsym(g_gbm_lib, "gbm_bo_unmap");
	if (!p_gbm_create_device || !p_gbm_bo_create || !p_gbm_bo_get_fd || !p_gbm_bo_map) {
		dlclose(g_gbm_lib);
		g_gbm_lib = NULL;
		return 0;
	}
	return 1;
}

static int open_render_node(void) {
	char path[64];
	int i;
	for (i = 128; i < 144; i++) {
		snprintf(path, sizeof(path), "/dev/dri/renderD%d", i);
		int fd = open(path, O_RDWR | O_CLOEXEC);
		if (fd >= 0) return fd;
	}
	for (i = 0; i < 8; i++) {
		snprintf(path, sizeof(path), "/dev/dri/card%d", i);
		int fd = open(path, O_RDWR | O_CLOEXEC);
		if (fd >= 0) return fd;
	}
	return -1;
}

static int gbm_retain(void) {
	if (g_gbm_dev) {
		g_gbm_refs++;
		return 1;
	}
	if (!load_gbm()) return 0;
	g_drm_fd = open_render_node();
	if (g_drm_fd < 0) return 0;
	g_gbm_dev = p_gbm_create_device(g_drm_fd);
	if (!g_gbm_dev) {
		close(g_drm_fd);
		g_drm_fd = -1;
		return 0;
	}
	g_gbm_refs = 1;
	return 1;
}

static void gbm_release(void) {
	if (g_gbm_refs > 0) g_gbm_refs--;
	if (g_gbm_refs > 0) return;
	if (g_gbm_dev && p_gbm_device_destroy) p_gbm_device_destroy(g_gbm_dev);
	g_gbm_dev = NULL;
	if (g_drm_fd >= 0) close(g_drm_fd);
	g_drm_fd = -1;
}

static size_t page_align(size_t n) {
	long ps = sysconf(_SC_PAGESIZE);
	if (ps < 4096) ps = 4096;
	return (n + (size_t)ps - 1) & ~((size_t)ps - 1);
}

static int alloc_gbm(struct ui_dmabuf_bo *out, int w, int h, uint32_t format) {
	if (!gbm_retain()) return -1;
	uint32_t flags = GBM_BO_USE_RENDERING | GBM_BO_USE_LINEAR;
	void *bo = p_gbm_bo_create(g_gbm_dev, (uint32_t)w, (uint32_t)h, format, flags);
	if (!bo) {
		gbm_release();
		return -1;
	}
	uint64_t mod = 0;
	if (p_gbm_bo_get_modifier) mod = p_gbm_bo_get_modifier(bo);
	if (mod != 0 && mod != 0x00ffffffffffffffULL) {
		p_gbm_bo_destroy(bo);
		gbm_release();
		return -1;
	}
	int fd = p_gbm_bo_get_fd(bo);
	if (fd < 0) {
		p_gbm_bo_destroy(bo);
		gbm_release();
		return -1;
	}
	uint32_t stride = 0;
	void *map_data = NULL;
	void *map = p_gbm_bo_map(bo, 0, 0, (uint32_t)w, (uint32_t)h, GBM_BO_TRANSFER_READ_WRITE, &stride, &map_data);
	if (!map || map == MAP_FAILED) {
		close(fd);
		p_gbm_bo_destroy(bo);
		gbm_release();
		return -1;
	}
	if (stride < (uint32_t)w * 4) stride = (uint32_t)w * 4;
	out->fd = fd;
	out->mapped = map;
	out->map_size = (size_t)stride * (size_t)h;
	out->stride = stride;
	out->offset = p_gbm_bo_get_offset ? p_gbm_bo_get_offset(bo, 0) : 0;
	out->modifier = mod;
	out->kind = UI_DMA_GBM;
	out->gbm_bo = bo;
	out->gbm_map_data = map_data;
	return 0;
}

static int alloc_heap(struct ui_dmabuf_bo *out, int w, int h) {
	static const char *paths[] = {
		"/dev/dma_heap/system",
		"/dev/dma_heap/system-uncached",
		NULL
	};
	uint32_t stride = (uint32_t)w * 4;
	size_t size = page_align((size_t)stride * (size_t)h);
	int i;
	for (i = 0; paths[i]; i++) {
		int heap = open(paths[i], O_RDWR | O_CLOEXEC);
		if (heap < 0) continue;
		struct dma_heap_allocation_data alloc;
		memset(&alloc, 0, sizeof(alloc));
		alloc.len = size;
		alloc.fd_flags = O_RDWR | O_CLOEXEC;
		if (ioctl(heap, DMA_HEAP_IOCTL_ALLOC, &alloc) != 0) {
			close(heap);
			continue;
		}
		close(heap);
		int fd = (int)alloc.fd;
		void *map = mmap(NULL, size, PROT_READ | PROT_WRITE, MAP_SHARED, fd, 0);
		if (map == MAP_FAILED) {
			close(fd);
			continue;
		}
		out->fd = fd;
		out->mapped = map;
		out->map_size = size;
		out->stride = stride;
		out->offset = 0;
		out->modifier = 0;
		out->kind = UI_DMA_HEAP;
		return 0;
	}
	return -1;
}

static int alloc_udmabuf(struct ui_dmabuf_bo *out, int w, int h) {
	uint32_t stride = (uint32_t)w * 4;
	size_t size = page_align((size_t)stride * (size_t)h);
	int memfd = memfd_create("uitk-dmabuf", MFD_CLOEXEC | MFD_ALLOW_SEALING);
	if (memfd < 0) return -1;
	if (ftruncate(memfd, (off_t)size) != 0) {
		close(memfd);
		return -1;
	}
	if (fcntl(memfd, F_ADD_SEALS, F_SEAL_SHRINK) != 0) {
		close(memfd);
		return -1;
	}
	int udev = open("/dev/udmabuf", O_RDWR | O_CLOEXEC);
	if (udev < 0) {
		close(memfd);
		return -1;
	}
	struct udmabuf_create create;
	memset(&create, 0, sizeof(create));
	create.memfd = (uint32_t)memfd;
	create.flags = UDMABUF_FLAGS_CLOEXEC;
	create.offset = 0;
	create.size = size;
	int fd = ioctl(udev, UDMABUF_CREATE, &create);
	close(udev);
	if (fd < 0) {
		close(memfd);
		return -1;
	}
	void *map = mmap(NULL, size, PROT_READ | PROT_WRITE, MAP_SHARED, memfd, 0);
	close(memfd);
	if (map == MAP_FAILED) {
		close(fd);
		return -1;
	}
	out->fd = fd;
	out->mapped = map;
	out->map_size = size;
	out->stride = stride;
	out->offset = 0;
	out->modifier = 0;
	out->kind = UI_DMA_UDMA;
	return 0;
}

int ui_dmabuf_alloc(struct ui_dmabuf_bo *out, int w, int h, uint32_t format) {
	if (!out || w < 1 || h < 1) return -1;
	memset(out, 0, sizeof(*out));
	out->fd = -1;
	if (alloc_gbm(out, w, h, format) == 0) return 0;
	if (alloc_heap(out, w, h) == 0) return 0;
	if (alloc_udmabuf(out, w, h) == 0) return 0;
	return -1;
}

void ui_dmabuf_free(struct ui_dmabuf_bo *bo) {
	if (!bo) return;
	if (bo->kind == UI_DMA_GBM) {
		if (bo->gbm_bo && p_gbm_bo_unmap) p_gbm_bo_unmap(bo->gbm_bo, bo->gbm_map_data);
		if (bo->fd >= 0) close(bo->fd);
		if (bo->gbm_bo && p_gbm_bo_destroy) p_gbm_bo_destroy(bo->gbm_bo);
		gbm_release();
	} else {
		if (bo->mapped && bo->mapped != MAP_FAILED && bo->map_size) munmap(bo->mapped, bo->map_size);
		if (bo->fd >= 0) close(bo->fd);
	}
	memset(bo, 0, sizeof(*bo));
	bo->fd = -1;
}

int ui_dmabuf_probe(char *name, int n) {
	struct ui_dmabuf_bo bo;
	if (ui_dmabuf_alloc(&bo, 64, 64, 0x34325241) != 0) {
		if (name && n > 0) name[0] = 0;
		return 0;
	}
	const char *s = "unknown";
	if (bo.kind == UI_DMA_GBM) s = "gbm";
	else if (bo.kind == UI_DMA_HEAP) s = "dma-heap";
	else if (bo.kind == UI_DMA_UDMA) s = "udmabuf";
	if (name && n > 0) {
		snprintf(name, (size_t)n, "%s", s);
	}
	ui_dmabuf_free(&bo);
	return 1;
}
