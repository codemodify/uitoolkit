#!/usr/bin/env python3
"""Compositor-level screenshot of the nested KWin (what is actually on screen).

usage: shot.py out.png
Reads the nested session bus address from bus.addr next to this script.
"""
import os, sys, struct, zlib, threading
import dbus

here = None
addr = os.environ["BUS"]
bus = dbus.bus.BusConnection(addr)
obj = bus.get_object("org.kde.KWin", "/org/kde/KWin/ScreenShot2")
iface = dbus.Interface(obj, "org.kde.KWin.ScreenShot2")

r, w = os.pipe()
chunks = []
def reader():
    with os.fdopen(r, "rb") as f:
        while True:
            b = f.read(1 << 20)
            if not b:
                break
            chunks.append(b)
t = threading.Thread(target=reader)
t.start()
res = iface.CaptureWorkspace({"include-cursor": dbus.Boolean(False), "native-resolution": dbus.Boolean(True)}, dbus.types.UnixFd(w))
os.close(w)
t.join()
data = b"".join(chunks)
W, H, stride, fmt = int(res["width"]), int(res["height"]), int(res["stride"]), int(res["format"])
# QImage::Format_ARGB32 (5), ARGB32_Premultiplied (6), RGB32 (4): little-endian BGRA bytes.
# Format_RGBA8888(_Premultiplied) (17/18) / RGBX8888 (16): RGBA bytes.
rows = []
for y in range(H):
    row = data[y * stride:y * stride + W * 4]
    if fmt in (16, 17, 18):
        px = bytearray(row)
    else:
        px = bytearray(len(row))
        px[0::4] = row[2::4]
        px[1::4] = row[1::4]
        px[2::4] = row[0::4]
        px[3::4] = row[3::4]
    for i in range(3, len(px), 4):
        px[i] = 255
    rows.append(b"\x00" + bytes(px))
raw = b"".join(rows)
def chunk(tag, body):
    c = struct.pack(">I", len(body)) + tag + body
    return c + struct.pack(">I", zlib.crc32(tag + body) & 0xFFFFFFFF)
png = b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", W, H, 8, 6, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(raw, 6)) + chunk(b"IEND", b"")
out = os.environ["OUT"]
open(out, "wb").write(png)
print(out)
