#!/usr/bin/env python3
"""tear.py X0 Y0 X1 Y1 PNG... — score each frame for a torn window.

A window presented from a paint target that was reallocated and never
painted shows whatever that memory held before. It comes out one of two
ways, and this scores both: a recycled image is a dense lattice of one-
and two-pixel dashes (nearly every horizontal neighbour jumps), and
zeroed pages are near-black where the interface is light. Real interface
pixels are mostly flat fills and text, so a clean frame scores well under
0.10 and a torn one over 0.25.

The box is a rectangle inside the window at its smallest, in the
screenshot's own pixels (device pixels: multiply logical geometry from
kwin.py by the output scale). Prints "score name" per frame and exits
non-zero if any frame is torn.

`tear.py --box PNG LOGICAL_SCREEN_W X Y` prints a box for a window whose
top-left is at logical (X, Y), converted to the screenshot's pixels.
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import crop

TORN = 0.25


def luma(row, x):
    o = x * 4
    return (row[o] * 299 + row[o + 1] * 587 + row[o + 2] * 114) // 1000


def score(path, box):
    w, h, px = crop.load(path)
    x0, y0, x1, y1 = box
    x0, y0 = max(x0, 0), max(y0, 0)
    x1, y1 = min(x1, w), min(y1, h)
    if x0 + 1 >= x1 or y0 >= y1:
        raise SystemExit("tear.py: the box is outside %s (%dx%d)" % (path, w, h))
    jump = dark = tot = 0
    for y in range(y0, y1, 2):
        row = px[y]
        prev = luma(row, x0)
        for x in range(x0 + 1, x1):
            cur = luma(row, x)
            tot += 1
            jump += abs(cur - prev) > 96
            dark += cur < 48
            prev = cur
    return max(jump, dark) / max(tot, 1)


def content_box(png, logical_w, x, y):
    """The box for a window whose top-left is at logical (x, y).

    A screenshot is device pixels and kwin.py speaks logical ones, so the
    scale is the screenshot's width over the logical screen's. The box is
    the top-left of the content, which a right-edge drag never moves.
    """
    if logical_w < 1:
        raise SystemExit("tear.py: no logical screen width (is the instance up?)")
    w, _, _ = crop.load(png, upto=1)
    f = w / float(logical_w)
    x0, y0 = int((x + 12) * f), int((y + 40) * f)
    return x0, y0, x0 + int(560 * f), y0 + int(380 * f)


def main(argv):
    if len(argv) == 6 and argv[1] == "--box":
        print("%d %d %d %d" % content_box(argv[2], float(argv[3]), float(argv[4]), float(argv[5])))
        return 0
    if len(argv) < 6:
        raise SystemExit(__doc__)
    box = tuple(int(v) for v in argv[1:5])
    torn = 0
    for path in argv[5:]:
        s = score(path, box)
        torn += s > TORN
        print("%.3f %s%s" % (s, os.path.basename(path), "  TORN" if s > TORN else ""))
    print("%d of %d frames torn" % (torn, len(argv) - 5))
    return 1 if torn else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
