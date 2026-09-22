#!/usr/bin/env python3
"""crop.py out.png x0 y0 x1 y1 in1.png [in2.png ...] — stack the same crop of
several shots, a red rule between them.

Reads any PNG a screenshot is likely to be — KWin's, the toolkit's own
WritePNG, ImageMagick's: every filter type (Go's encoder picks one per row),
grey, grey and alpha, RGB, RGBA and palette images, 1- to 16-bit samples,
Adam7 interlacing and tRNS transparency. Python's standard library only.
A shot smaller than the crop is padded with transparent pixels."""
import struct
import sys
import zlib

SIG = b'\x89PNG\r\n\x1a\n'
CHANNELS = {0: 1, 2: 3, 3: 1, 4: 2, 6: 4}
# Adam7: each pass's first column, first row, and steps across and down.
ADAM7 = [(0, 0, 8, 8), (4, 0, 8, 8), (0, 4, 4, 8), (2, 0, 4, 4), (0, 2, 2, 4), (1, 0, 2, 2), (0, 1, 1, 2)]


def paeth(a, b, c):
    p = a + b - c
    pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
    if pa <= pb and pa <= pc:
        return a
    return b if pb <= pc else c


def unfilter(data, pos, width, height, bpp, stride):
    """The rows of one image (or interlace pass) starting at data[pos], with
    their filters undone; returns the rows and where the next pass starts."""
    rows, prev = [], bytearray(stride)
    for _ in range(height):
        ftype = data[pos]
        cur = bytearray(data[pos + 1:pos + 1 + stride])
        pos += 1 + stride
        if len(cur) != stride:
            raise ValueError('truncated image data')
        if ftype == 1:  # Sub
            for i in range(bpp, stride):
                cur[i] = (cur[i] + cur[i - bpp]) & 0xff
        elif ftype == 2:  # Up
            for i in range(stride):
                cur[i] = (cur[i] + prev[i]) & 0xff
        elif ftype == 3:  # Average
            for i in range(stride):
                left = cur[i - bpp] if i >= bpp else 0
                cur[i] = (cur[i] + ((left + prev[i]) >> 1)) & 0xff
        elif ftype == 4:  # Paeth
            for i in range(stride):
                left = cur[i - bpp] if i >= bpp else 0
                upleft = prev[i - bpp] if i >= bpp else 0
                cur[i] = (cur[i] + paeth(left, prev[i], upleft)) & 0xff
        elif ftype != 0:
            raise ValueError('unknown filter type %d' % ftype)
        rows.append(cur)
        prev = cur
    return rows, pos


def samples(row, width, channels, depth):
    """A row's samples as 8-bit values (16-bit keeps its high byte; 1-, 2-
    and 4-bit samples are the raw index or scaled grey, see to_rgba)."""
    n = width * channels
    if depth == 8:
        return list(row[:n])
    if depth == 16:
        return list(row[0:2 * n:2])
    out, per, mask = [], 8 // depth, (1 << depth) - 1
    for i in range(n):
        byte = row[i // per]
        shift = 8 - depth * (i % per + 1)
        out.append((byte >> shift) & mask)
    return out


def row_rgba(row, width, ctype, depth, plte, trns):
    """One unfiltered row as RGBA bytes, the common screenshot formats
    without a loop over their pixels."""
    if depth == 8 and ctype == 6:
        return row
    if depth == 8 and ctype == 2 and not trns:
        out = bytearray(b'\xff' * (width * 4))
        for c in range(3):
            out[c::4] = row[c::3]
        return out
    return to_rgba(samples(row, width, CHANNELS[ctype], depth), width, ctype, depth, plte, trns)


def to_rgba(vals, width, ctype, depth, plte, trns):
    """One row of samples as RGBA bytes."""
    out = bytearray(width * 4)
    if ctype == 3:
        for x in range(width):
            i = vals[x]
            r, g, b = plte[3 * i:3 * i + 3] if 3 * i + 3 <= len(plte) else (0, 0, 0)
            a = trns[i] if i < len(trns) else 255
            out[4 * x:4 * x + 4] = bytes((r, g, b, a))
        return out
    scale = 255 // ((1 << depth) - 1) if depth < 8 else 1
    key = None
    if trns and ctype in (0, 2):
        # tRNS names one colour, in the image's own depth, that is clear.
        k = struct.unpack('>%dH' % (len(trns) // 2), trns)
        key = tuple((v >> 8) if depth == 16 else v * scale for v in k)
    ch = CHANNELS[ctype]
    for x in range(width):
        px = [v * scale for v in vals[x * ch:(x + 1) * ch]]
        if ctype == 0:
            rgba = (px[0], px[0], px[0], 0 if key == (px[0],) else 255)
        elif ctype == 4:
            rgba = (px[0], px[0], px[0], px[1])
        elif ctype == 2:
            rgba = (px[0], px[1], px[2], 0 if key == tuple(px) else 255)
        else:
            rgba = tuple(px)
        out[4 * x:4 * x + 4] = bytes(rgba)
    return out


def load(path, upto=None):
    """Decodes a PNG to (width, height, rows of RGBA bytes); with upto, only
    the rows above that one are decoded (the rest come back transparent)."""
    with open(path, 'rb') as f:
        data = f.read()
    if data[:8] != SIG:
        raise ValueError('%s: not a PNG' % path)
    i, idat, plte, trns = 8, [], b'', b''
    while i < len(data):
        n = struct.unpack('>I', data[i:i + 4])[0]
        kind, body = data[i + 4:i + 8], data[i + 8:i + 8 + n]
        i += 12 + n
        if kind == b'IHDR':
            width, height, depth, ctype, _, _, interlace = struct.unpack('>IIBBBBB', body)
        elif kind == b'PLTE':
            plte = body
        elif kind == b'tRNS':
            trns = body
        elif kind == b'IDAT':
            idat.append(body)
        elif kind == b'IEND':
            break
    if ctype not in CHANNELS:
        raise ValueError('%s: unknown colour type %d' % (path, ctype))
    raw = zlib.decompress(b''.join(idat))
    ch = CHANNELS[ctype]
    bits = ch * depth
    bpp = max(1, bits // 8)

    def stride(w):
        return (w * bits + 7) // 8

    rows = [bytearray(width * 4) for _ in range(height)]
    if interlace == 0:
        # A row's filter reads only the row above it, so the rows under a
        # crop need not be undone at all.
        n = height if upto is None else max(0, min(height, upto))
        filtered, _ = unfilter(raw, 0, width, n, bpp, stride(width))
        for y, row in enumerate(filtered):
            rows[y] = row_rgba(row, width, ctype, depth, plte, trns)
        return width, height, rows
    pos = 0
    for x0, y0, dx, dy in ADAM7:
        pw = (width - x0 + dx - 1) // dx if width > x0 else 0
        ph = (height - y0 + dy - 1) // dy if height > y0 else 0
        if pw == 0 or ph == 0:
            continue
        filtered, pos = unfilter(raw, pos, pw, ph, bpp, stride(pw))
        for j, row in enumerate(filtered):
            rgba = row_rgba(row, pw, ctype, depth, plte, trns)
            y = y0 + j * dy
            for k in range(pw):
                x = x0 + k * dx
                rows[y][4 * x:4 * x + 4] = rgba[4 * k:4 * k + 4]
    return width, height, rows


def chunk(kind, body):
    return struct.pack('>I', len(body)) + kind + body + struct.pack('>I', zlib.crc32(kind + body) & 0xffffffff)


def save(path, width, rows):
    """Writes RGBA rows as an 8-bit RGBA PNG."""
    raw = b''.join(b'\x00' + bytes(r) for r in rows)
    with open(path, 'wb') as f:
        f.write(SIG + chunk(b'IHDR', struct.pack('>IIBBBBB', width, len(rows), 8, 6, 0, 0, 0)) +
                chunk(b'IDAT', zlib.compress(raw, 9)) + chunk(b'IEND', b''))


def crop(out, x0, y0, x1, y1, shots):
    w = x1 - x0
    if w <= 0 or y1 <= y0:
        raise ValueError('an empty crop box')
    rule = b'\xff\x00\x00\xff' * w
    rows = []
    for s in shots:
        width, height, img = load(s, y1)
        for y in range(y0, y1):
            row = bytearray(w * 4)  # transparent where the shot is smaller
            if 0 <= y < height:
                a, b = max(x0, 0), min(x1, width)
                if a < b:
                    row[4 * (a - x0):4 * (b - x0)] = img[y][4 * a:4 * b]
            rows.append(row)
        rows += [rule, rule]
    save(out, w, rows)


def main(argv):
    if len(argv) < 7:
        sys.exit(__doc__)
    x0, y0, x1, y1 = map(int, argv[2:6])
    crop(argv[1], x0, y0, x1, y1, argv[6:])


if __name__ == '__main__':
    main(sys.argv)
