#!/bin/bash
# measure.sh N SCALE [APP...] — memory, start-up, frame times and idle CPU of
# every demo app, one at a time, in nested-KWin rig instance N (tools/e2e) on
# the real GPU at output scale SCALE. The virtual screen is 1280x860 logical
# pixels at any scale, so every window opens in the same place and the
# scripted pointer lands on the same controls.
#
#   tools/perf/measure.sh 31 1.75           # every app at 1.75
#   tools/perf/measure.sh 31 1 mail notes   # just these, at 1x
#
# It (re)starts instance N at the scale asked for, builds the apps into
# $RIG/N/perf/bin, and for each app:
#
#   1. launches it with UITK_PERF_LOG (app/perflog.go) and no look.json, so
#      it opens in the default pack;
#   2. start-up: from the exec to the end of the first painted frame;
#   3. settled memory, 10 s after the launch: Private_Dirty and Rss from
#      /proc/PID/smaps_rollup (the same reading as RESUME's 2026-09-15 table),
#      Private_Dirty the median of PERF_RUNS launches (3; the others read
#      nothing else);
#   4. idle: CPU time and wakeups (context switches, every thread) over 10 s
#      with nothing touched;
#   5. frame times while the pointer sweeps the window (hover), the wheel
#      scrolls, a menu or popup opens and closes, and the theme switches
#      twice (look.json: breeze6, then metal-ocean);
#   6. memory again, settled after all that, and a heap profile
#      (SIGUSR1: $RIG/N/perf/APP.log.heap.1.pb.gz).
#
# PERF_BIN=DIR measures the binaries already in DIR (named as in apps.txt)
# instead of building the tree, e.g. an older build to compare with;
# PERF_TAG=NAME keeps its results apart. Keep binaries on a disk, not in a
# tmpfs such as /tmp: a tmpfs page is always dirty, so a binary there counts
# its whole text as Private_Dirty (12.6 MB for Settings), and not freshly
# written (see the sync below).
#
# Prints one row per app and writes $RIG/N/perf/results-SCALE.tsv. The load
# average beside each row is the machine's when the app started: compare
# rows taken on a quiet machine. Stop the instance with tools/e2e/stop.sh N.
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"; RIG="$ROOT/tools/e2e"
N="${1:?instance}"; SCALE="${2:?scale}"; shift 2
D="$RIG/$N"; OUT="$D/perf"; BIN="$OUT/bin"
mkdir -p "$BIN"

APPS="$(grep -v '^#' "$ROOT/tools/perf/apps.txt")"
want=" $* "

# The instance, at the scale asked for: a 1280x860 logical screen.
cur=""
[ -f "$D/bus.addr" ] && [ -f "$D/kwin.pid" ] && kill -0 "$(cat "$D/kwin.pid")" 2>/dev/null &&
  cur=$(WAYLAND_DISPLAY=uitk-e2e-$N DBUS_SESSION_BUS_ADDRESS="$(cat "$D/bus.addr")" kscreen-doctor -o 2>/dev/null |
    sed 's/\x1b\[[0-9;]*m//g' | awk '/Geometry:/{g=$3} /Scale:/{s=$2} END{print g "@" s}')
if [ "$cur" != "1280x860@$SCALE" ]; then
  "$RIG/stop.sh" "$N" >/dev/null; sleep 1
  W=$(awk -v s="$SCALE" 'BEGIN{printf "%d", 1280*s+0.5}'); H=$(awk -v s="$SCALE" 'BEGIN{printf "%d", 860*s+0.5}')
  "$RIG/start.sh" "$N" "$W" "$H" >/dev/null
  WAYLAND_DISPLAY=uitk-e2e-$N DBUS_SESSION_BUS_ADDRESS="$(cat "$D/bus.addr")" \
    kscreen-doctor "output.Virtual-0.scale.$SCALE" >/dev/null 2>&1
  sleep 2
fi
[ -x "$RIG/inj" ] || (cd "$RIG" && ./build.sh >/dev/null) || exit 1

# ticks and wakeups of every thread of PID: "utime+stime ctxsw"
usage() {
  local t=0 c=0 f
  for f in /proc/"$1"/task/*; do
    t=$((t + $(awk '{print $14 + $15}' "$f/stat" 2>/dev/null || echo 0)))
    c=$((c + $(awk '/ctxt_switches/{s+=$2} END{print s+0}' "$f/status" 2>/dev/null || echo 0)))
  done
  echo "$t $c"
}
mem() { awk '/^Rss:/{r=$2} /^Private_Dirty:/{p=$2} END{printf "%.1f %.1f", p/1024, r/1024}' "/proc/$1/smaps_rollup"; }
mark() { echo "mark $1 $(date +%s%N)" >> "$LOG"; }
look() { mkdir -p "$D/cfg/uitoolkit"; printf '{"version":2,"theme":"%s"}\n' "$1" > "$D/cfg/uitoolkit/look.json"; }

# Every binary is built, then written back to the disk, before the first
# launch: the page cache holds a file just written as dirty pages, and a
# binary mapped from them counts them all as Private_Dirty (17 MB more for
# Files) until the filesystem flushes them, some 30 s later on btrfs.
if [ -z "${PERF_BIN:-}" ]; then
  while read -r name pkg _ <&3; do
    [ -z "$name" ] && continue
    [ "$want" != "  " ] && [[ "$want" != *" $name "* ]] && continue
    (cd "$ROOT" && nice go build -o "$BIN/$name" "./$pkg") || echo "$name: build failed"
  done 3<<< "$APPS"
fi
sync "${PERF_BIN:-$BIN}"/*

TSV="$OUT/results-$SCALE${PERF_TAG:+-$PERF_TAG}.tsv"
printf 'app\tscale\tload\tstart_ms\town_MB\trss_MB\tidle_cpu_ms\tidle_wakeups\thover_ms\tscroll_ms\tmenu_ms\ttheme_ms\town_after_MB\tgoroutines\ttrim_s\ttex_MB\n' > "$TSV"
echo "$(date -I) · scale $SCALE · $(uname -r) · $(grep -m1 'model name' /proc/cpuinfo | cut -d: -f2 | xargs)"
printf '%-9s %5s %8s %6s %6s %13s %11s %11s %11s %11s %6s %4s %5s %5s\n' app load start own rss idle hover scroll menu theme after gor trim tex
while read -r name pkg mk mx my sx sy <&3; do
  [ -z "$name" ] && continue
  [ "$mk" = "-" ] && { sx=$mx; sy=$my; }
  [ "$want" != "  " ] && [[ "$want" != *" $name "* ]] && continue
  exe="$BIN/$name"
  [ -n "${PERF_BIN:-}" ] && exe="$PERF_BIN/$name"
  [ -x "$exe" ] || continue
  rm -f "$D/cfg/uitoolkit/look.json" "$OUT/$name.log"*
  # One launch's settled memory can sit 2–3 MB either side of another's
  # (where the heap lands, when the collector ran): the extra launches
  # read only that, and the row gives the median of them all.
  owns=()
  for _ in $(seq 2 "${PERF_RUNS:-3}"); do
    rm -f "$D/app.pid"
    RUN_WAIT=10 "$RIG/run.sh" "$N" "$exe" >/dev/null
    p=$(cat "$D/app.pid"); kill -0 "$p" 2>/dev/null && owns+=("$(mem "$p" | cut -d' ' -f1)")
    kill "$p" 2>/dev/null
    for _ in $(seq 50); do kill -0 "$p" 2>/dev/null || break; sleep 0.1; done
  done
  LOG="$OUT/$name.log"
  load=$(cut -d' ' -f1 /proc/loadavg)
  # run.sh waits 0.4 s to see a previous app out when it finds its pid
  # file, and that wait would count as start-up: the last app is gone.
  rm -f "$D/app.pid"
  t0=$(date +%s%N)
  UITK_PERF_LOG="$LOG" RUN_WAIT=0 "$RIG/run.sh" "$N" "$exe" >/dev/null
  pid=$(cat "$D/app.pid")
  sleep 10
  kill -0 "$pid" 2>/dev/null || { echo "$name: exited (see $D/app.log)"; continue; }
  read -r own rss <<< "$(mem "$pid")"
  own=$(printf '%s\n' "$own" "${owns[@]}" | sort -n | awk '{v[NR]=$1} END{print v[int((NR+1)/2)]}')
  read -r c0 w0 <<< "$(usage "$pid")"; sleep 10; read -r c1 w1 <<< "$(usage "$pid")"
  idle_ms=$(( (c1 - c0) * 1000 / $(getconf CLK_TCK) )); idle_w=$((w1 - w0))
  # The window, for the hover sweep: three rows across it.
  geo=$("$RIG/kwin.py" "$N" 'const w = workspace.activeWindow; if (w) OUT(w.frameGeometry.x+" "+w.frameGeometry.y+" "+w.frameGeometry.width+" "+w.frameGeometry.height)')
  read -r gx gy gw gh <<< "${geo:-0 0 800 600}"
  moves=()
  for fy in 0.3 0.5 0.7; do
    for i in $(seq 0 23); do
      moves+=(move "$(awk -v x="$gx" -v w="$gw" -v i="$i" 'BEGIN{printf "%d", x + w*(0.05 + 0.9*i/23)}')" \
        "$(awk -v y="$gy" -v h="$gh" -v f="$fy" 'BEGIN{printf "%d", y + h*f}')" sleep 16)
    done
  done
  mark hover; "$RIG/in.sh" "$N" "${moves[@]}"; sleep 0.5
  mark scroll
  "$RIG/in.sh" "$N" move "$sx" "$sy" sleep 200 $(for i in $(seq 8); do echo wheel 0.6666667 sleep 60; done) \
    $(for i in $(seq 8); do echo wheel -0.6666667 sleep 60; done); sleep 0.5
  mark menu
  if [ "$mk" != "-" ]; then
    btn=left; [ "$mk" = right ] && btn=right
    for i in 1 2 3; do "$RIG/in.sh" "$N" move "$mx" "$my" sleep 100 click $btn sleep 600 key 1 sleep 400; done
  fi
  sleep 0.5
  mark theme; look breeze6; sleep 2.5; look metal-ocean; sleep 2.5
  mark end
  sleep 5
  read -r own2 _ <<< "$(mem "$pid")"
  kill -USR1 "$pid"; sleep 1.5
  kill "$pid" 2>/dev/null
  for _ in $(seq 50); do kill -0 "$pid" 2>/dev/null || break; sleep 0.1; done
  row=$(python3 "$ROOT/tools/perf/stats.py" "$LOG" "$t0" 2>/dev/null || echo "- - - - - - - -")
  read -r start hov scr men thm gor trim tex <<< "$row"
  printf '%-9s %5s %8s %6s %6s %13s %11s %11s %11s %11s %6s %4s %5s %5s\n' "$name" "$load" "${start}ms" "$own" "$rss" \
    "${idle_ms}ms/${idle_w}w" "$hov" "$scr" "$men" "$thm" "$own2" "$gor" "$trim" "$tex"
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$name" "$SCALE" "$load" "$start" "$own" "$rss" \
    "$idle_ms" "$idle_w" "$hov" "$scr" "$men" "$thm" "$own2" "$gor" "$trim" "$tex" >> "$TSV"
  rm -f "$D/cfg/uitoolkit/look.json"
  sleep 2
done 3<<< "$APPS"
echo "results: $TSV"
