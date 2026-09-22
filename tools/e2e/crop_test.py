#!/usr/bin/env python3
"""crop_test.py — crop.py against PNGs of every filter type, colour type,
bit depth and interlacing, written here by an encoder that picks the
filters itself. Run: python3 tools/e2e/crop_test.py"""
import os
import struct
import sys
import tempfile
import unittest
import zlib

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import crop  # noqa: E402

W, H = 13, 11  # odd, so every Adam7 pass and every sub-byte row is ragged


def pixel(x, y):
    """The test card: every channel different and changing both ways, so a
    filter undone against the wrong neighbour shows."""
    return ((x * 37 + y * 11) & 0xff, (x * 5 + y * 53) & 0xff, (x * y * 7 + 29) & 0xff, (x * 19 + y * 23 + 40) & 0xff)


def filt(ftype, cur, prev, bpp):
    out = bytearray(len(cur))
    for i, v in enumerate(cur):
        a = cur[i - bpp] if i >= bpp else 0
        b = prev[i]
        c = prev[i - bpp] if i >= bpp else 0
        pred = (0, a, b, (a + b) >> 1, crop.paeth(a, b, c))[ftype]
        out[i] = (v - pred) & 0xff
    return bytes([ftype]) + bytes(out)


def encode(path, rows, ctype, depth, ftypes, interlace=False, plte=b'', trns=b''):
    """Writes rows (lists of samples at depth) with the filter for row y
    ftypes[y % len(ftypes)], interlaced with Adam7 when asked."""
    ch = crop.CHANNELS[ctype]
    bits = ch * depth
    bpp = max(1, bits // 8)

    def pack(vals):
        if depth == 8:
            return bytes(vals)
        if depth == 16:
            return b''.join(struct.pack('>H', v) for v in vals)
        out, per = bytearray((len(vals) * depth + 7) // 8), 8 // depth
        for i, v in enumerate(vals):
            out[i // per] |= v << (8 - depth * (i % per + 1))
        return bytes(out)

    def image(sub):
        raw, prev, n = b'', None, 0
        for r in sub:
            cur = pack(r)
            prev = prev or bytes(len(cur))
            raw += filt(ftypes[n % len(ftypes)], cur, prev, bpp)
            prev, n = cur, n + 1
        return raw

    if not interlace:
        raw = image(rows)
    else:
        raw = b''
        for x0, y0, dx, dy in crop.ADAM7:
            sub = [row[x0 * ch::dx * ch] if ch == 1 else
                   [v for x in range(x0, W, dx) for v in row[x * ch:(x + 1) * ch]]
                   for row in rows[y0::dy]]
            if sub and sub[0]:
                raw += image(sub)
    body = crop.chunk(b'IHDR', struct.pack('>IIBBBBB', W, H, depth, ctype, 0, 0, 1 if interlace else 0))
    if plte:
        body += crop.chunk(b'PLTE', plte)
    if trns:
        body += crop.chunk(b'tRNS', trns)
    # Split IDAT in two, as encoders that stream do.
    z = zlib.compress(raw)
    body += crop.chunk(b'IDAT', z[:len(z) // 2]) + crop.chunk(b'IDAT', z[len(z) // 2:])
    with open(path, 'wb') as f:
        f.write(crop.SIG + body + crop.chunk(b'IEND', b''))


class CropReadsEveryPNG(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.dir = tmp.name

    def path(self, name):
        return os.path.join(self.dir, name)

    def check(self, name, want):
        """crop.py's crop of the whole card, then of a box off its edges."""
        src, out = self.path(name + '.png'), self.path(name + '-out.png')
        w, h, rows = crop.load(src)
        self.assertEqual((w, h), (W, H), name)
        for y in range(H):
            for x in range(W):
                got = tuple(rows[y][4 * x:4 * x + 4])
                self.assertEqual(got, want(x, y), '%s at %d,%d' % (name, x, y))
        # A box that runs past the right and bottom edges, of two shots.
        crop.crop(out, 3, 2, W + 4, H + 1, [src, src])
        cw, ch, crows = crop.load(out)
        self.assertEqual(cw, W + 1)
        self.assertEqual(ch, 2 * (H - 1 + 2))
        self.assertEqual(tuple(crows[0][0:4]), want(3, 2), name)
        self.assertEqual(tuple(crows[0][4 * (W - 4):4 * (W - 3)]), want(W - 1, 2), name)
        self.assertEqual(tuple(crows[0][4 * W:4 * W + 4]), (0, 0, 0, 0), name + ': padding')
        self.assertEqual(tuple(crows[H - 2][0:4]), (0, 0, 0, 0), name + ': the row past the bottom')
        self.assertEqual(tuple(crows[H - 1][0:4]), (255, 0, 0, 255), name + ': the rule')
        self.assertEqual(tuple(crows[H + 1][0:4]), want(3, 2), name + ': the second shot')
        # A crop of the top rows decodes only those, and gets them right.
        crop.crop(out, 0, 1, W, 4, [src])
        _, th, trows = crop.load(out)
        self.assertEqual(th, 3 + 2)
        for y in range(3):
            for x in range(W):
                self.assertEqual(tuple(trows[y][4 * x:4 * x + 4]), want(x, y + 1), '%s top at %d,%d' % (name, x, y))

    def test_rgba_every_filter(self):
        rows = [[v for x in range(W) for v in pixel(x, y)] for y in range(H)]
        for f in range(5):
            encode(self.path('rgba%d.png' % f), rows, 6, 8, [f])
            self.check('rgba%d' % f, pixel)
        # A different filter on each row, the way Go's encoder (and so the
        # toolkit's WritePNG) chooses them.
        encode(self.path('rgba-mixed.png'), rows, 6, 8, [4, 1, 0, 2, 3, 4, 2])
        self.check('rgba-mixed', pixel)
        encode(self.path('rgba-adam7.png'), rows, 6, 8, [4, 2, 1, 3], interlace=True)
        self.check('rgba-adam7', pixel)

    def test_rgb_and_grey(self):
        rgb = [[v for x in range(W) for v in pixel(x, y)[:3]] for y in range(H)]
        encode(self.path('rgb.png'), rgb, 2, 8, [1, 2, 3, 4, 0])
        self.check('rgb', lambda x, y: pixel(x, y)[:3] + (255,))
        # tRNS: one colour of an RGB image is clear.
        key = pixel(5, 5)[:3]
        encode(self.path('rgb-key.png'), rgb, 2, 8, [4], trns=struct.pack('>3H', *key))
        self.check('rgb-key', lambda x, y: pixel(x, y)[:3] + (0 if pixel(x, y)[:3] == key else 255,))
        grey = [[pixel(x, y)[0] for x in range(W)] for y in range(H)]
        encode(self.path('grey.png'), grey, 0, 8, [3, 4])
        self.check('grey', lambda x, y: (pixel(x, y)[0],) * 3 + (255,))
        ga = [[v for x in range(W) for v in (pixel(x, y)[0], pixel(x, y)[3])] for y in range(H)]
        encode(self.path('grey-alpha.png'), ga, 4, 8, [2, 4, 1], interlace=True)
        self.check('grey-alpha', lambda x, y: (pixel(x, y)[0],) * 3 + (pixel(x, y)[3],))

    def test_sixteen_bit(self):
        rows = [[v * 256 + (x * 31 + y) % 256 for x in range(W) for v in pixel(x, y)] for y in range(H)]
        encode(self.path('rgba16.png'), rows, 6, 16, [0, 1, 2, 3, 4])
        self.check('rgba16', pixel)

    def test_palette_and_low_depths(self):
        colours = [pixel(i, 3 * i) for i in range(16)]
        plte = b''.join(bytes(c[:3]) for c in colours)
        trns = bytes(c[3] for c in colours[:9])  # the last entries opaque
        for depth in (1, 2, 4, 8):
            n = 1 << min(depth, 4)
            rows = [[(x + 2 * y) % n for x in range(W)] for y in range(H)]

            def want(x, y, n=n):
                i = (x + 2 * y) % n
                return colours[i][:3] + (colours[i][3] if i < 9 else 255,)
            name = 'plte%d' % depth
            encode(self.path(name + '.png'), rows, 3, depth, [4, 0, 1, 2, 3], plte=plte, trns=trns)
            self.check(name, want)
            encode(self.path(name + 'i.png'), rows, 3, depth, [1, 4], interlace=True, plte=plte, trns=trns)
            self.check(name + 'i', want)
        for depth in (1, 2, 4):
            top = (1 << depth) - 1
            rows = [[(x * y) % (top + 1) for x in range(W)] for y in range(H)]
            encode(self.path('grey%d.png' % depth), rows, 0, depth, [2, 3])
            self.check('grey%d' % depth, lambda x, y, d=depth, t=top: ((x * y) % (t + 1) * (255 // t),) * 3 + (255,))


if __name__ == '__main__':
    unittest.main()
