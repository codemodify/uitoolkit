#!/bin/bash
# stop.sh N — stop the app and the nested KWin of instance N.
RIG="$(cd "$(dirname "$0")" && pwd)"; N="${1:?instance}"; D="$RIG/$N"
[ -f "$D/app.pid" ] && kill "$(cat "$D/app.pid")" 2>/dev/null
[ -f "$D/kwin.pid" ] && pkill -TERM -s "$(ps -o sid= -p "$(cat "$D/kwin.pid")" 2>/dev/null | tr -d ' ')" 2>/dev/null
rm -f "$D/app.pid" "$D/kwin.pid"
echo "instance $N stopped"
