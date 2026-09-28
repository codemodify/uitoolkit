#!/bin/bash
# test-windows.sh — run the Windows tests on real Windows, from here.
#
#   tools/test-windows.sh [go test args...]
#
# The Windows backend cannot be tested on this machine: tools/test.sh
# never compiles it, and a cross-compiled test binary cannot run. So the
# tests are cross-compiled, handed to a Windows VM, run there, and their
# output brought back — the same shape as any other test run, just with a
# hypervisor in the middle.
#
# How the two halves talk: QEMU's user networking maps the host's loopback
# to 10.0.2.2 in the guest, so a server bound to 127.0.0.1 here is
# reachable there and nothing is exposed on the network. The guest fetches
# the test binary over it and POSTs the output back. There is deliberately
# no SSH and no share: both wanted a working guest before they could help,
# and this needs nothing installed in the guest but curl, which Windows 10
# and 11 both ship.
#
# What the guest needs, once:
#   * an interactive logged-in desktop — the tests make real windows, and
#     a window needs a session to appear in;
#   * Microsoft Defender told to leave the directory alone. It quarantines
#     freshly built unsigned Go binaries: the file vanishes, or survives
#     and will not start ("The system cannot execute the specified
#     program"), which looks exactly like a broken build.
#
# UITK_WIN_VM_MON  the QEMU monitor socket of the guest to drive
# UITK_WIN_VM_PORT the port this serves on (default 8099)
set -euo pipefail
cd "$(dirname "$0")/.."

MON=${UITK_WIN_VM_MON:-}
PORT=${UITK_WIN_VM_PORT:-8099}
if [ -z "$MON" ]; then
  echo "set UITK_WIN_VM_MON to the QEMU monitor socket of a logged-in Windows guest" >&2
  exit 2
fi

work=$(mktemp -d)
trap 'rm -rf "$work"; kill %1 2>/dev/null || true' EXIT

echo "== cross-compiling the Windows tests =="
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o "$work/platform.test.exe" ./platform/

cat > "$work/serve.py" <<'PY'
import http.server, os, sys
DIR = os.path.dirname(os.path.abspath(__file__))
class H(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *a, **kw): super().__init__(*a, directory=DIR, **kw)
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        open(os.path.join(DIR, 'result.txt'), 'wb').write(self.rfile.read(n))
        self.send_response(200); self.send_header('Content-Length', '3')
        self.end_headers(); self.wfile.write(b'ok\n')
    def log_message(self, *a): pass
http.server.ThreadingHTTPServer(('127.0.0.1', int(sys.argv[1])), H).serve_forever()
PY
python3 "$work/serve.py" "$PORT" &
sleep 1

# The guest side, as one file, so only one short line has to be typed at it.
{
  echo '@echo off'
  echo "set SRC=http://10.0.2.2:$PORT"
  echo 'set D=%TEMP%\uitkwin'
  echo 'if not exist "%D%" mkdir "%D%"'
  echo 'curl -s -o "%D%\platform.test.exe" %SRC%/platform.test.exe'
  echo "\"%D%\\platform.test.exe\" -test.v $* > \"%D%\\out.txt\" 2>&1"
  echo 'echo EXIT=%ERRORLEVEL% >> "%D%\out.txt"'
  echo 'curl -s --data-binary @"%D%\out.txt" %SRC%/result.txt'
} | sed 's/$/\r/' > "$work/run.cmd"

echo "== running them in the guest =="
python3 - "$MON" "$PORT" <<'PY'
import socket, sys, time
sock, port = sys.argv[1], sys.argv[2]
PLAIN = {' ': 'spc', '-': 'minus', '=': 'equal', '\\': 'backslash', '.': 'dot',
         '/': 'slash', ';': 'semicolon', ',': 'comma'}
SHIFT = {':': 'semicolon', '%': '5', '_': 'minus', '"': 'apostrophe', '&': '7'}
def keys(s):
    for ch in s:
        if ch.isdigit() or 'a' <= ch <= 'z': yield ch
        elif 'A' <= ch <= 'Z': yield 'shift-' + ch.lower()
        elif ch in PLAIN: yield PLAIN[ch]
        elif ch in SHIFT: yield 'shift-' + SHIFT[ch]
        else: raise SystemExit('no key for %r' % ch)
cmd = r'cmd /c curl -s -o %TEMP%\run.cmd http://10.0.2.2:' + port + r'/run.cmd && %TEMP%\run.cmd'
s = socket.socket(socket.AF_UNIX); s.connect(sock)
time.sleep(0.3); s.recv(65536)
def send(k):
    s.sendall(('sendkey %s\n' % k).encode()); time.sleep(0.06)
    s.settimeout(0.2)
    try: s.recv(65536)
    except socket.timeout: pass
send('meta_l-r'); time.sleep(2.5)      # the Run dialog always takes focus,
for k in keys(cmd): send(k)            # unlike whatever window is on top
send('ret')
s.close()
PY

echo "== waiting for the result =="
for _ in $(seq 1 120); do
  [ -s "$work/result.txt" ] && break
  sleep 5
done
if [ ! -s "$work/result.txt" ]; then
  echo "no result came back: is the guest logged in, and is Defender excluding %TEMP%\\uitkwin?" >&2
  exit 1
fi
tr -d '\r' < "$work/result.txt"
grep -q '^EXIT=0' "$work/result.txt" || exit 1
