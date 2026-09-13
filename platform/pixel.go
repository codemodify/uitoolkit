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

// maskWidth is the number of bits in a contiguous X visual mask.
func maskWidth(mask uint32) uint {
	if mask == 0 {
		return 8
	}
	mask >>= maskShift(mask)
	var n uint
	for mask&1 == 1 {
		mask >>= 1
		n++
	}
	return n
}

// scaleChannel narrows an 8-bit channel to a visual mask of width bits
// (8 -> 5 or 6 on a 16-bpp RGB565 / RGB555 visual).
func scaleChannel(v uint8, bits uint) uint32 {
	if bits >= 8 {
		return uint32(v)
	}
	if bits == 0 {
		return 0
	}
	return uint32(v) >> (8 - bits)
}

// packXPixelN writes one pixel of bytesPP bytes (4 or 2) into dst.
// Non-32-bpp TrueColor visuals (RGB565, RGB555) are still common on
// remote X, Xvfb, and old hardware; writing 4 bytes per pixel there
// overran the image by a factor of two.
func packXPixelN(dst []byte, bytesPP int, r, g, b, a uint8, msbFirst bool, redMask, greenMask, blueMask uint32) {
	switch bytesPP {
	case 4:
		packXPixel(dst, r, g, b, a, msbFirst, redMask, greenMask, blueMask)
	case 2:
		packXPixel16(dst, r, g, b, msbFirst, redMask, greenMask, blueMask)
	}
}

// packXPixel16 writes one 16-bit pixel using the visual's masks.
func packXPixel16(dst []byte, r, g, b uint8, msbFirst bool, redMask, greenMask, blueMask uint32) {
	if len(dst) < 2 {
		return
	}
	if redMask == 0 && greenMask == 0 && blueMask == 0 {
		// Assume RGB565 when the server reported no masks.
		redMask, greenMask, blueMask = 0xf800, 0x07e0, 0x001f
	}
	pix := scaleChannel(r, maskWidth(redMask))<<maskShift(redMask) |
		scaleChannel(g, maskWidth(greenMask))<<maskShift(greenMask) |
		scaleChannel(b, maskWidth(blueMask))<<maskShift(blueMask)
	if msbFirst {
		dst[0] = byte(pix >> 8)
		dst[1] = byte(pix)
		return
	}
	dst[0] = byte(pix)
	dst[1] = byte(pix >> 8)
}
