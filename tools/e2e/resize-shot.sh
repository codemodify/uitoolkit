#!/bin/bash
# resize-shot.sh N X Y DX DY NAME [SHOTS] — drag from X,Y (a window's
# resize corner or edge, compositor coords) by DX,DY in small steps, holding
# the button, and take SHOTS screenshots (default 3) while the pointer is
# still moving: N/shots/NAME-1.png ... What a window looks like in the middle
# of an interactive resize — half drawn, or in step with its frame
# (_NET_WM_SYNC_REQUEST on X11; UITK_X11_SYNC=0 turns it off to compare).
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; X="${2:?x}"; Y="${3:?y}"
DX="${4:?dx}"; DY="${5:?dy}"; NAME="${6:?name}"; SHOTS="${7:-3}"
STEPS=120
args=(move "$X" "$Y" sleep 100 down)
for i in $(seq 1 $STEPS); do
  args+=(move $((X + DX * i / STEPS)) $((Y + DY * i / STEPS)) sleep 25)
done
args+=(sleep 200 up)
"$RIG/in.sh" "$N" "${args[@]}" &
INJ=$!
sleep 0.9
for k in $(seq 1 "$SHOTS"); do
  "$RIG/shot.sh" "$N" "$NAME-$k" >/dev/null
  sleep 0.35
done
wait "$INJ"
ls "$RIG/$N/shots/$NAME"-*.png
