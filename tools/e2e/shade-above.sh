#!/bin/bash
# shade-above.sh N [BACKEND [PACK]] — the two title-bar behaviours that need a
# real compositor to see: the wheel over the caption rolling the window up to
# its title bar and back, and the keep-above caption button holding a window
# in front of another.
#
#   ./start.sh 7 && ./shade-above.sh 7            # wayland, breeze
#   ./shade-above.sh 7 x11 win95                  # the X11 path
#
# KWin is the oracle for both. For rolling up it is the window's geometry —
# a rolled-up window is its caption and nothing else, and the height it comes
# back to must be the one it went up with — and the size hints KWin read: a
# rolled-up window is pinned to one height and free in width, and the pin is
# gone once it is back down (a pin left standing would hold the window shut).
# For keep-above it is the stacking order: the window behind is clicked, so it
# becomes the active window, and the kept-above one must still be on top.
#
# Keep-above is X11-only. On Wayland there is no keep-above a client may ask
# for on its own surface, so the run checks that the toolkit says so rather
# than offering a button that does nothing. Shots land in $RIG/N/shade/ —
# look at them: only a picture says the caption is still drawn and the
# toggled-on button reads as on.
set -u
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; BACKEND="${2:-wayland}"; PACK="${3:-breeze}"
ROOT="$(cd "$RIG/../.." && pwd)"; D="$RIG/$N"; OUT="$D/shade"; BIN="$D/bin"
[ -f "$D/bus.addr" ] || { echo "instance $N is not running: ./start.sh $N"; exit 1; }
mkdir -p "$OUT" "$BIN" "$D/cfg/uitoolkit"
(cd "$ROOT" && go build -o "$BIN/rigapp" ./tools/e2e/rigapp) || exit 1

fail=0
say() { printf '%-54s %s\n' "$1" "$2"; }
shot() { "$RIG/shot.sh" "$N" "$1" >/dev/null && mv "$D/shots/$1.png" "$OUT/"; }
js() { "$RIG/kwin.py" "$N" "$1"; }
one() { js "for (const w of workspace.windowList()) if (w.normalWindow && w.caption == \"$1\") OUT($2)"; }

# The wheel reaches the app on Xwayland only when a move came first in the
# same call, so every turn here is move, wheel, move.
turn() { "$RIG/in.sh" "$N" move "$2" "$3" sleep 400 wheel "$1" sleep 200 move "$2" $(($3 + 1)) sleep 900 >/dev/null; }

cat > "$D/cfg/uitoolkit/look.json" <<EOF
{"theme":"$PACK","corners":"theme","iconSize":"small","reduceMotion":true,"decorations":"toolkit"}
EOF

# ---- 1. the wheel over the caption rolls the window up and back -------------
UITK_BACKEND="$BACKEND" RUN_WAIT=3.5 "$RIG/run.sh" "$N" "$BIN/rigapp" \
  -title "RollUp" -w 520 -h 320 -body "CONTENT BELOW THE TITLE BAR" >/dev/null || exit 1
read -r x y w h <<<"$(one RollUp 'w.frameGeometry.x + " " + w.frameGeometry.y + " " + w.frameGeometry.width + " " + w.frameGeometry.height')"
[ -n "${h:-}" ] || { echo "no window on instance $N" >&2; exit 1; }
open_h=${h%.*}
capx=$(( ${x%.*} + ${w%.*} / 3 )); capy=$(( ${y%.*} + 14 ))
shot "sa-$BACKEND-$PACK-1-open"

turn -0.6666667 "$capx" "$capy"
shot "sa-$BACKEND-$PACK-2-rolled-up"
read -r rolled <<<"$(one RollUp 'w.frameGeometry.height')"
read -r minh maxh minw maxw <<<"$(one RollUp 'w.minSize.height + " " + w.maxSize.height + " " + w.minSize.width + " " + w.maxSize.width')"
if [ "${rolled%.*}" -lt "$open_h" ] && [ "${rolled%.*}" -le 64 ]; then
  say "$BACKEND/$PACK: the wheel rolls the window up to its caption" "ok (${open_h} -> ${rolled%.*})"
else
  say "$BACKEND/$PACK: the wheel rolls the window up to its caption" "FAIL (${open_h} -> ${rolled%.*})"; fail=1
fi
# The pin is the height alone: the width keeps whatever the window stated
# (the Wayland backend floors a resizable toplevel at 200 logical px, X11
# states no minimum at all), so a rolled-up window still resizes sideways.
if [ "${minh%.*}" = "${maxh%.*}" ] && [ "${minw%.*}" != "${maxw%.*}" ]; then
  say "$BACKEND/$PACK: rolled up, the height is pinned and the width free" "ok (h=${minh%.*}, w ${minw%.*}..${maxw%.*})"
else
  say "$BACKEND/$PACK: rolled up, the height is pinned and the width free" "FAIL (min ${minw%.*}x${minh%.*} max ${maxw%.*}x${maxh%.*})"; fail=1
fi

turn 0.6666667 "$capx" "$capy"
shot "sa-$BACKEND-$PACK-3-rolled-down"
read -r back <<<"$(one RollUp 'w.frameGeometry.height')"
read -r minh2 maxh2 <<<"$(one RollUp 'w.minSize.height + " " + w.maxSize.height')"
[ "${back%.*}" = "$open_h" ] &&
  say "$BACKEND/$PACK: it comes back to the height it went up with" ok ||
  { say "$BACKEND/$PACK: it comes back to the height it went up with" "FAIL (${back%.*}, was $open_h)"; fail=1; }
[ "${minh2%.*}" != "${maxh2%.*}" ] &&
  say "$BACKEND/$PACK: the height pin goes when the window comes back" ok ||
  { say "$BACKEND/$PACK: the height pin goes when the window comes back" "FAIL (still pinned at ${minh2%.*})"; fail=1; }

# ---- 2. the keep-above button holds a window in front ----------------------
kill "$(cat "$D/app.pid" 2>/dev/null)" 2>/dev/null; sleep 0.5
launch() {
  UITK_BACKEND="$BACKEND" WAYLAND_DISPLAY="uitk-e2e-$N" DISPLAY="$(cat "$D/display")" \
    XDG_CONFIG_HOME="$D/cfg" XDG_DATA_HOME="$D/data" XDG_CACHE_HOME="$D/cache" \
    DBUS_SESSION_BUS_ADDRESS="$(cat "$D/bus.addr")" \
    "$BIN/rigapp" "$@" >> "$D/shade-above.log" 2>&1 &
  echo $! >> "$D/shade.pids"
}
rm -f "$D/shade-above.log" "$D/shade.pids"
launch -title "BEHIND" -w 620 -h 420 -body "THE WINDOW BEHIND"; sleep 3
launch -title "ONTOP" -w 380 -h 240 -body "SHOULD STAY IN FRONT"; sleep 3

can=$(grep -o 'canKeepAbove=[a-z]*' "$D/shade-above.log" | tail -1 | cut -d= -f2)
case "$BACKEND" in
  wayland) want=false;;
  *)       want=true;;
esac
if [ "$can" = "$want" ]; then
  say "$BACKEND: the toolkit says what this backend can do" "ok (canKeepAbove=$can)"
else
  say "$BACKEND: the toolkit says what this backend can do" "FAIL (canKeepAbove=$can, want $want)"; fail=1
fi
if [ "$can" != true ]; then
  # Nothing to drive: no compositor-side keep-above, and so no button.
  kill $(cat "$D/shade.pids" 2>/dev/null) 2>/dev/null
  echo "shots in $OUT"
  exit $fail
fi

read -r ox oy <<<"$(one ONTOP 'w.frameGeometry.x + " " + w.frameGeometry.y')"
read -r bx by bh <<<"$(one BEHIND 'w.frameGeometry.x + " " + w.frameGeometry.y + " " + w.frameGeometry.height')"
# The keep-above button sits in the window-menu slot, at the caption's left.
"$RIG/in.sh" "$N" move $(( ${ox%.*} + 14 )) $(( ${oy%.*} + 15 )) sleep 400 click sleep 900 >/dev/null
above=$(one ONTOP 'w.keepAbove')
[ "$above" = true ] &&
  say "x11: the caption button sets _NET_WM_STATE_ABOVE" ok ||
  { say "x11: the caption button sets _NET_WM_STATE_ABOVE" "FAIL (keepAbove=$above)"; fail=1; }

# Click the window behind, well clear of the one in front, so it is activated
# and raised: the kept-above window must stay on top of it anyway.
"$RIG/in.sh" "$N" move $(( ${bx%.*} + 50 )) $(( ${by%.*} + ${bh%.*} - 30 )) sleep 400 click sleep 900 >/dev/null
shot "sa-$BACKEND-$PACK-4-kept-above"
top=$(js 'const s = workspace.stackingOrder.filter(w => w.normalWindow); OUT(s[s.length - 1].caption)')
active=$(js 'OUT(workspace.activeWindow.caption)')
if [ "$top" = ONTOP ] && [ "$active" = BEHIND ]; then
  say "x11: it stays in front of the window that was just raised" ok
else
  say "x11: it stays in front of the window that was just raised" "FAIL (top=$top active=$active)"; fail=1
fi

kill $(cat "$D/shade.pids" 2>/dev/null) 2>/dev/null
echo "shots in $OUT (1-open, 2-rolled-up, 3-rolled-down, 4-kept-above)"
exit $fail
