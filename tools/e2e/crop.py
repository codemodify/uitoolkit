#!/usr/bin/env python3
"""crop.py out.png x0 y0 x1 y1 in1.png [in2.png ...] — stack the same crop of several shots (red separators)."""
import sys, zlib, struct
def load(p):
    d=open(p,'rb').read(); i=8; idat=b''; W=H=0
    while i<len(d):
        n=struct.unpack('>I',d[i:i+4])[0]; t=d[i+4:i+8]; b=d[i+8:i+8+n]; i+=12+n
        if t==b'IHDR': W,H=struct.unpack('>II',b[:8])
        if t==b'IDAT': idat+=b
    return W,H,zlib.decompress(idat)
out=sys.argv[1]; x0,y0,x1,y1=map(int,sys.argv[2:6]); rows=[]
for s in sys.argv[6:]:
    W,H,raw=load(s); st=W*4+1
    for y in range(y0,min(y1,H)):
        rows.append(b'\x00'+raw[y*st+1+x0*4:y*st+1+x1*4])
    rows.append(b'\x00'+b'\xff\x00\x00\xff'*(x1-x0)); rows.append(b'\x00'+b'\xff\x00\x00\xff'*(x1-x0))
w=x1-x0; h=len(rows)
def ch(t,b): return struct.pack('>I',len(b))+t+b+struct.pack('>I',zlib.crc32(t+b)&0xffffffff)
open(out,'wb').write(b'\x89PNG\r\n\x1a\n'+ch(b'IHDR',struct.pack('>IIBBBBB',w,h,8,6,0,0,0))+ch(b'IDAT',zlib.compress(b''.join(rows)))+ch(b'IEND',b''))
