#!/bin/bash
# theme-tour.sh N [PACK...] — show the gallery in every theme pack (or the
# packs named) on the real GPU in nested KWin instance N, and render the same
# pack headless with the CPU rasterizer next to it, so an engine that draws
# differently on the GPU (gradients, radial lights, paths, shadows) shows up.
#
#   ./start.sh 5 && ./theme-tour.sh 5            # every pack
#   ./theme-tour.sh 5 aqua luna breeze           # just these
#
# Writes $RIG/N/tour/PACK-gpu.png (compositor shot) and PACK-cpu.png (the
# window's content, headless, same 1000x760 size). Compare them in pairs;
# `tour.txt` lists what ran. Start the instance first (start.sh N).
set -u
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; shift
ROOT="$(cd "$RIG/../.." && pwd)"; D="$RIG/$N"; OUT="$D/tour"; BIN="$D/bin"
[ -f "$D/bus.addr" ] || { echo "instance $N is not running: ./start.sh $N"; exit 1; }
mkdir -p "$OUT" "$BIN"
(cd "$ROOT" && go build -o "$BIN/gallery" ./examples/gallery && go build -o "$BIN/sheet" ./cmd/uitk-themesheet) || exit 1
if [ $# -gt 0 ]; then
  PACKS=("$@")
else
  mapfile -t PACKS < <("$BIN/sheet" -list | cut -f1)
fi
: > "$OUT/tour.txt"
for P in "${PACKS[@]}"; do
  # The GPU window, in the nested compositor.
  # Animations off in both, so the GPU and CPU stills catch the same frame.
  UITK_ANIMATIONS=0 UITK_THEME="$P" RUN_WAIT=3 "$RIG/run.sh" "$N" "$BIN/gallery" >/dev/null
  "$RIG/shot.sh" "$N" "tour-$P" >/dev/null && mv "$D/shots/tour-$P.png" "$OUT/$P-gpu.png"
  # The same content on the CPU, headless (never on a display).
  TMP="$(mktemp -d)"
  (cd "$TMP" && env -u WAYLAND_DISPLAY -u DISPLAY UITK_ANIMATIONS=0 UITK_THEME="$P" XDG_CONFIG_HOME="$TMP/cfg" "$BIN/gallery" -headless >/dev/null 2>&1 && mv gallery.png "$OUT/$P-cpu.png")
  rm -rf "$TMP"
  grep -qi "panic\|fatal" "$D/app.log" && echo "$P  APP ERROR (see $D/app.log)" | tee -a "$OUT/tour.txt" || echo "$P  ok" >> "$OUT/tour.txt"
done
kill "$(cat "$D/app.pid" 2>/dev/null)" 2>/dev/null
echo "tour of ${#PACKS[@]} packs in $OUT (pairs: PACK-gpu.png / PACK-cpu.png)"
