#!/usr/bin/env python3
"""stats.py LOG T0 — reduce an app's UITK_PERF_LOG (app/perflog.go) to one row
for measure.sh: start-up in ms (T0, the exec in unix ns, to the end of the
first painted frame), then for each marked stretch (hover, scroll, menu,
theme) "frames/median/p95" of the frame times in ms, the goroutine count
of the first SIGUSR1 dump, when the first idle trim ran (s after T0), and
the most GPU texture memory one window's device held (MB)."""
import statistics, sys

log, t0 = sys.argv[1], int(sys.argv[2])
frames, marks, trims, gor, tex = [], [], [], "-", 0
for line in open(log):
    f = line.split()
    if f[0] == "frame":
        frames.append((int(f[1]), int(f[2]) / 1000.0, f[3]))
        if len(f) > 4:
            tex = max(tex, int(f[4]))
    elif f[0] == "mark":
        marks.append((f[1], int(f[2])))
    elif f[0] == "trim":
        trims.append(int(f[1]))
    elif f[0] == "dump" and gor == "-":
        gor = f[3]

start = "%.0f" % ((frames[0][0] - t0) / 1e6) if frames else "-"


def stretch(i):
    lo, hi = marks[i][1], marks[i + 1][1]
    ms = sorted(d for end, d, _ in frames if lo <= end < hi)
    if not ms:
        return "0"
    p95 = ms[min(len(ms) - 1, int(round(0.95 * (len(ms) - 1))))]
    return "%d/%.1f/%.1f" % (len(ms), statistics.median(ms), p95)


trim = "%.1f" % ((trims[0] - t0) / 1e9) if trims else "-"
print(start, *[stretch(i) for i in range(len(marks) - 1)], gor, trim, "%.1f" % (tex / 1048576))
