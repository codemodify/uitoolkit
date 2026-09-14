#!/bin/bash
# shot.sh N NAME — compositor screenshot of instance N into $RIG/N/shots/NAME.png (prints path)
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; NAME="${2:?name}"
BUS="$(cat "$RIG/$N/bus.addr")" OUT="$RIG/$N/shots/$NAME.png" exec python3 "$RIG/shot.py"
