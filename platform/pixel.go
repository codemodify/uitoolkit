package platform

// maskShift returns the bit offset of a contiguous X visual mask.
func maskShift(mask uint32) uint {
	if mask == 0 {
		return 0
	}
	var s uint
	for mask&1 == 0 {
		mask >>= 1
		s++
	}
	return s
}

// packXPixel writes one 32-bit ZPixmap pixel into dst (at least 4 bytes).
// Typical little-endian TrueColor (red 0xff0000) yields B,G,R,A in memory.
func packXPixel(dst []byte, r, g, b, a uint8, msbFirst bool, redMask, greenMask, blueMask uint32) {
	if len(dst) < 4 {
		return
	}
	var pix uint32
	if redMask == 0 && greenMask == 0 && blueMask == 0 {
		pix = uint32(b) | uint32(g)<<8 | uint32(r)<<16 | uint32(a)<<24
	} else {
		pix = uint32(r)<<maskShift(redMask) | uint32(g)<<maskShift(greenMask) | uint32(b)<<maskShift(blueMask)
		if a != 0 && redMask != 0xff000000 && greenMask != 0xff000000 && blueMask != 0xff000000 {
			// Depth-24 / 32 bpp: high byte is padding; keep alpha there when unused.
			if pix>>24 == 0 {
				pix |= uint32(a) << 24
			}
		}
	}
	if msbFirst {
		dst[0] = byte(pix >> 24)
		dst[1] = byte(pix >> 16)
		dst[2] = byte(pix >> 8)
		dst[3] = byte(pix)
		return
	}
	dst[0] = byte(pix)
	dst[1] = byte(pix >> 8)
	dst[2] = byte(pix >> 16)
	dst[3] = byte(pix >> 24)
}
