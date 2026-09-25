#!/bin/bash
# oracle.sh N [APP...] — the partial-redraw oracle: every app of
# tools/perf/apps.txt (or those named) runs twice in rig instance N, once
# painting as it always does and once with UITK_PAINT_FULLFRAME=1 (every
# frame repainted and presented whole), through the same scripted input, and
# the compositor's screenshots after each step are compared pixel for pixel.
# A pixel that differs is one partial redraw got wrong: left stale, or
# painted where nothing changed.
#
#   tools/perf/oracle.sh 31                  # every app
#   tools/perf/oracle.sh 31 tour notes       # just these
#
# Animations are off (UITK_ANIMATIONS=0), so a busy bar or a caret holds
# still between the two runs. The players animate on their own clock and
# are compared only where the steps leave them alike. Uses the binaries
# measure.sh built ($RIG/N/perf/bin); run that first, or it builds them.
# PERF_BIN=DIR and PERF_TAG=NAME check other binaries, as for measure.sh.
# Writes $RIG/N/oracle/APP-STEP-{part,full,diff}.png and prints, per app and
# step, how many pixels differ and by how much at most (in 255ths).
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"; RIG="$ROOT/tools/e2e"
N="${1:?instance}"; shift
D="$RIG/$N"; OUT="$D/oracle${PERF_TAG:+-$PERF_TAG}"; BIN="${PERF_BIN:-$D/perf/bin}"
mkdir -p "$OUT" "$BIN"
[ -f "$D/bus.addr" ] || { echo "instance $N is not running (tools/perf/measure.sh starts it)"; exit 1; }
want=" $* "

# steps NAME MX MY SX SY: the input between shots, one step a line.
steps() {
  local mk=$1 mx=$2 my=$3 sx=$4 sy=$5
  echo "hover move $sx $sy sleep 300 move $((sx + 40)) $((sy + 20)) sleep 600"
  echo "scroll move $sx $sy sleep 200 wheel 0.6666667 sleep 200 wheel 0.6666667 sleep 800"
  if [ "$mk" != "-" ]; then
    local btn=left; [ "$mk" = right ] && btn=right
    echo "menu move $mx $my sleep 150 click $btn sleep 900"
    echo "menuhover move $((mx + 30)) $((my + 40)) sleep 400 move $((mx + 30)) $((my + 64)) sleep 800"
    echo "closed key 1 sleep 900"
  fi
}

total=0
while read -r name pkg mk mx my sx sy <&3; do
  [ -z "$name" ] && continue
  [ "$mk" = "-" ] && { sx=$mx; sy=$my; }
  [ "$want" != "  " ] && [[ "$want" != *" $name "* ]] && continue
  [ -x "$BIN/$name" ] || (cd "$ROOT" && nice go build -o "$BIN/$name" "./$pkg") || continue
  for mode in part full; do
    full=0; [ $mode = full ] && full=1
    rm -f "$D/app.pid" "$D/cfg/uitoolkit/look.json"
    UITK_ANIMATIONS=0 UITK_PAINT_FULLFRAME=$full RUN_WAIT=3 "$RIG/run.sh" "$N" "$BIN/$name" >/dev/null
    "$RIG/in.sh" "$N" move 5 5 sleep 300
    while read -r step input; do
      # shellcheck disable=SC2086
      "$RIG/in.sh" "$N" $input
      "$RIG/shot.sh" "$N" "oracle" >/dev/null && mv "$D/shots/oracle.png" "$OUT/$name-$step-$mode.png"
    done < <(steps "$mk" "$mx" "$my" "$sx" "$sy")
    pid=$(cat "$D/app.pid"); kill "$pid" 2>/dev/null
    for _ in $(seq 30); do kill -0 "$pid" 2>/dev/null || break; sleep 0.1; done
  done
  line="$name:"
  while read -r step _; do
    # The largest difference of any channel, per pixel: how many differ
    # at all, and by how much at most.
    magick "$OUT/$name-$step-part.png" "$OUT/$name-$step-full.png" -compose difference -composite \
      -separate -evaluate-sequence max "$OUT/$name-$step-diff.png"
    most=$(magick "$OUT/$name-$step-diff.png" -format "%[fx:round(maxima*255)]" info:)
    n=$(magick "$OUT/$name-$step-diff.png" -threshold 0 -format "%[fx:round(mean*w*h)]" info:)
    line="$line $step=$n/$most"
    total=$((total + n))
  done < <(steps "$mk" "$mx" "$my" "$sx" "$sy")
  echo "$line"
done 3< <(grep -v '^#' "$ROOT/tools/perf/apps.txt")
echo "pixels differing in all: $total (shots in $OUT)"
