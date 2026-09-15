#!/usr/bin/env python3
"""kwin.py N 'JS' — run a KWin script in rig instance N and print its output.

In the script, OUT(x) prints x (read back from N/kwin.log). The KWin
scripting API reports and drives windows, e.g. an oracle for a window's
geometry and state, or a maximize / quick tile the rig's --no-global-shortcuts
KWin has no key for:

  ./kwin.py 7 'for (const w of workspace.windowList()) if (w.normalWindow)
      OUT(w.caption + " " + w.frameGeometry + " min=" + w.minimized)'
  ./kwin.py 7 'workspace.activeWindow.setMaximize(true, true)'
  ./kwin.py 7 'workspace.slotWindowQuickTileLeft()'
"""
import os, sys, tempfile, time

import dbus

rig = os.path.dirname(os.path.abspath(__file__))
n, code = sys.argv[1], sys.argv[2]
d = os.path.join(rig, n)
bus = dbus.bus.BusConnection(open(os.path.join(d, "bus.addr")).read().strip())
tag = "KWINPY%d" % int(time.time() * 1000)
fd, path = tempfile.mkstemp(suffix=".js", dir=d)
os.write(fd, code.replace("OUT(", "print('%s ' + " % tag).encode())
os.close(fd)
log = os.path.join(d, "kwin.log")
start = os.path.getsize(log)
try:
    sid = bus.call_blocking("org.kde.KWin", "/Scripting", "org.kde.kwin.Scripting", "loadScript", "ss", (path, tag))
    bus.call_blocking("org.kde.KWin", "/Scripting/Script%d" % sid, "org.kde.kwin.Script", "run", "", ())
    time.sleep(0.4)
    bus.call_blocking("org.kde.KWin", "/Scripting", "org.kde.kwin.Scripting", "unloadScript", "s", (tag,))
finally:
    os.unlink(path)
with open(log, errors="replace") as f:
    f.seek(start)
    for line in f:
        if tag in line:
            print(line.split(tag, 1)[1].strip())
