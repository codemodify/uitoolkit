//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-client
#cgo linux LDFLAGS: -ldl
#cgo linux CFLAGS: -I${SRCDIR}
#include "wayland_dmabuf.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func (c *wlConn) requestDmabufFeedback() {
	if c == nil || c.dmabuf == nil || c.dmabufVer < 4 {
		return
	}
	fb := C.ui_wl_dmabuf_feedback((*C.struct_zwp_linux_dmabuf_v1)(c.dmabuf), C.uintptr_t(c.id))
	if fb != nil {
		c.dmabufFB = unsafe.Pointer(fb)
	}
}

func (c *wlConn) chooseWaylandPresent() {
	if c == nil {
		return
	}
	c.useDmabuf = false
	c.dmaAlloc = ""
	if !waylandWantDmabuf() {
		return
	}
	if c.dmabuf == nil || c.dmabufVer < 2 {
		return
	}
	fmtu, mod, ok := pickDmabufFormat(c.dmaPairs)
	if !ok {
		return
	}
	var name [32]C.char
	if C.ui_dmabuf_probe(&name[0], 32) == 0 {
		return
	}
	c.dmaFmt = fmtu
	c.dmaMod = mod
	c.dmaSwizzle = dmabufNeedsSwizzle(fmtu)
	c.dmaAlloc = C.GoString(&name[0])
	c.useDmabuf = true
}

func (c *wlConn) destroyDmabufLocked() {
	if c.dmabufFB != nil {
		C.ui_wl_dmabuf_feedback_destroy((*C.struct_zwp_linux_dmabuf_feedback_v1)(c.dmabufFB))
		c.dmabufFB = nil
	}
	if c.dmabuf != nil {
		C.ui_wl_dmabuf_destroy((*C.struct_zwp_linux_dmabuf_v1)(c.dmabuf))
		c.dmabuf = nil
	}
	c.useDmabuf = false
}

func (s *wlSurface) ensureDmabufSlot(i, w, h int) error {
	if s.conn == nil || s.conn.dmabuf == nil {
		return fmt.Errorf("platform: no linux-dmabuf")
	}
	bo := (*C.struct_ui_dmabuf_bo)(C.calloc(1, C.size_t(unsafe.Sizeof(C.struct_ui_dmabuf_bo{}))))
	if bo == nil {
		return fmt.Errorf("platform: dmabuf calloc")
	}
	if C.ui_dmabuf_alloc(bo, C.int(w), C.int(h), C.uint32_t(s.conn.dmaFmt)) != 0 {
		C.free(unsafe.Pointer(bo))
		return fmt.Errorf("platform: dmabuf alloc failed")
	}
	mod := s.conn.dmaMod
	if bo.modifier != 0 && modifierCPUFriendly(uint64(bo.modifier)) {
		mod = uint64(bo.modifier)
	}
	hi := C.uint32_t(mod >> 32)
	lo := C.uint32_t(mod)
	buf := C.ui_wl_dmabuf_buffer((*C.struct_zwp_linux_dmabuf_v1)(s.conn.dmabuf),
		bo.fd, C.int(w), C.int(h), bo.stride, bo.offset, C.uint32_t(s.conn.dmaFmt), hi, lo)
	if buf == nil {
		C.ui_dmabuf_free(bo)
		C.free(unsafe.Pointer(bo))
		return fmt.Errorf("platform: linux-dmabuf create_immed failed")
	}
	s.slots[i] = wlSlot{
		buf:    buf,
		mem:    bo.mapped,
		size:   int(bo.map_size),
		fd:     int(bo.fd),
		w:      w,
		h:      h,
		stride: int(bo.stride),
		dma:    unsafe.Pointer(bo),
	}
	return nil
}

func (s *wlSurface) freeDmabufSlot(i int) {
	sl := &s.slots[i]
	if sl.dma != nil {
		C.ui_dmabuf_free((*C.struct_ui_dmabuf_bo)(sl.dma))
		C.free(sl.dma)
		sl.dma = nil
		sl.mem = nil
	}
}

func dmabufAllocatorOK() bool {
	var name [32]C.char
	return C.ui_dmabuf_probe(&name[0], 32) != 0
}

func dmabufAllocatorName() string {
	var name [32]C.char
	if C.ui_dmabuf_probe(&name[0], 32) == 0 {
		return ""
	}
	return C.GoString(&name[0])
}

func waylandUsingDmabuf() bool {
	wlMu.Lock()
	defer wlMu.Unlock()
	return wlc != nil && wlc.useDmabuf
}

//export uitkWlDmabufFormat
func uitkWlDmabufFormat(id C.uintptr_t, format C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	c.dmaPairs = append(c.dmaPairs, dmaFmtMod{format: uint32(format), modifier: drmModInvalid})
}

//export uitkWlDmabufModifier
func uitkWlDmabufModifier(id C.uintptr_t, format, hi, lo C.uint32_t) {
	c := wlConnBy(id)
	if c == nil {
		return
	}
	mod := uint64(hi)<<32 | uint64(lo)
	c.dmaPairs = append(c.dmaPairs, dmaFmtMod{format: uint32(format), modifier: mod})
}

//export uitkWlDmabufFeedbackDone
func uitkWlDmabufFeedbackDone(id C.uintptr_t) {
	_ = id
}

//export uitkWlDmabufFormatTable
func uitkWlDmabufFormatTable(id C.uintptr_t, mapped unsafe.Pointer, size C.uint32_t) {
	c := wlConnBy(id)
	if c == nil || mapped == nil || size < 16 {
		return
	}
	n := int(size) / 16
	b := unsafe.Slice((*byte)(mapped), int(size))
	c.dmaTable = make([]dmaFmtMod, 0, n)
	for i := 0; i < n; i++ {
		off := i * 16
		fmtu := *(*uint32)(unsafe.Pointer(&b[off]))
		mod := *(*uint64)(unsafe.Pointer(&b[off+8]))
		c.dmaTable = append(c.dmaTable, dmaFmtMod{format: fmtu, modifier: mod})
	}
}

//export uitkWlDmabufTrancheFormats
func uitkWlDmabufTrancheFormats(id C.uintptr_t, idx *C.uint16_t, n C.int) {
	c := wlConnBy(id)
	if c == nil || idx == nil || n <= 0 {
		return
	}
	ids := unsafe.Slice((*uint16)(unsafe.Pointer(idx)), int(n))
	for _, i := range ids {
		if int(i) >= len(c.dmaTable) {
			continue
		}
		p := c.dmaTable[i]
		if dmabufFormatUsable(p.format) {
			c.dmaPairs = append(c.dmaPairs, p)
		}
	}
}
