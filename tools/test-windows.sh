#!/bin/bash
# test-windows.sh — run the test suite on real Windows, from here.
#
#   tools/test-windows.sh                             every package
#   tools/test-windows.sh -test.run TestFoo           that test, everywhere
#   UITK_WIN_PKGS=./platform/ tools/test-windows.sh   one package
#
# The Windows backend cannot be tested on this machine: tools/test.sh
# never compiles it, and a cross-compiled test binary cannot run. So the
# tests are cross-compiled, handed to a Windows VM, run there, and their
# output brought back — the same shape as any other test run, just with a
# hypervisor in the middle.
#
# **It runs the whole suite**, not only ./platform/. Running that one
# package is what it did, and the asymmetry cost real bugs: the widget,
# style and app suites had never executed on Windows at all, so a layout
# that depended on the developer's installed fonts, or a path assumption,
# surfaced on macOS first and only because macOS ran everything.
#
# That costs more than one binary. `go test -c` compiles one package at a
# time, so a full run is two dozen binaries of about 18 MB each. They are
# fetched and deleted one at a time rather than shipped together: the
# guest never holds more than one, and the run starts producing output
# immediately instead of after a 450 MB download.
#
# The source tree goes over too, once, because some tests read files next
# to themselves — internal/uitest opens testdata/chrome-strip-*.png and
# skingen reads ../style/skins. `go test` runs each package in its own
# directory, so each binary is run from its package's directory in the
# unpacked tree.
#
# How the two halves talk: QEMU's user networking maps the host's loopback
# to 10.0.2.2 in the guest, so a server bound to 127.0.0.1 here is
# reachable there and nothing is exposed on the network. The guest fetches
# over it and POSTs the output back. There is deliberately no SSH and no
# share: both wanted a working guest before they could help, and this
# needs nothing in the guest but curl and tar, which Windows 10 and 11
# both ship.
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
# UITK_WIN_PKGS    the packages to test (default ./...)
set -euo pipefail
cd "$(dirname "$0")/.."

MON=${UITK_WIN_VM_MON:-}
PORT=${UITK_WIN_VM_PORT:-8099}
PKGS=${UITK_WIN_PKGS:-./...}
if [ -z "$MON" ]; then
  echo "set UITK_WIN_VM_MON to the QEMU monitor socket of a logged-in Windows guest" >&2
  exit 2
fi

# Every argument is quoted for cmd, which reads | & < > ^ in an unquoted
# word as its own syntax: a -test.run 'A|B' became a pipe into a command
# called B, and the guest sat there for ever while this waited out its
# timeout. Single-name runs worked, which is exactly what made it look
# like the window tests hanging.
args=""
for a in "$@"; do args+=" \"$a\""; done

# Nothing else may be on the port: the guest fetches run.cmd from it, and
# another server there answers 404. The run then sits in "waiting for the
# result" until it times out, and the message it prints blames the guest's
# login state or Defender — neither of which is wrong. That cost an
# afternoon once, to a leftover file server from an earlier session.
if command -v ss >/dev/null && ss -ltn "sport = :$PORT" 2>/dev/null | grep -q ":$PORT"; then
  echo "port $PORT is already in use; the guest would fetch run.cmd from whatever is there." >&2
  echo "free it, or set UITK_WIN_VM_PORT to another port." >&2
  exit 2
fi

work=$(mktemp -d)
trap 'rm -rf "$work"; kill %1 2>/dev/null || true' EXIT
mkdir -p "$work/bin"

echo "== cross-compiling the Windows tests =="
# One binary per package that has any, named after its import path with
# the slashes flattened, and pkgs.txt says which directory each has to run
# in.
: > "$work/pkgs.txt"
n=0
while read -r p; do
  [ -z "$p" ] && continue
  rel=${p#github.com/codemodify/uitoolkit}
  rel=${rel#/}
  [ -z "$rel" ] && rel="."
  name=$(printf '%s' "$rel" | tr '/' '_')
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
    go test -c -tags theme_engine_all -o "$work/bin/$name.test.exe" "$p"
  # Backslashes: the guest will cd to it.
  printf '%s %s\n' "$name" "$(printf '%s' "$rel" | tr '/' '\\')" >> "$work/pkgs.txt"
  n=$((n + 1))
done < <(GOOS=windows go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' \
  -tags theme_engine_all $PKGS)
echo "   $n packages"

echo "== packing the source tree =="
# For the tests that read files next to themselves. tools/atlas/out is
# generated screenshots and is the only large thing excluded for size.
tar czf "$work/src.tgz" --exclude=.git --exclude=tools/atlas/out --exclude='*.test.exe' .

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

# The guest side, as one file, so only one short line has to be typed at
# it. Nothing in here is typed, so it may hold characters the sendkey map
# below has no key for.
{
  echo '@echo off'
  echo "set SRC=http://10.0.2.2:$PORT"
  echo 'set D=%TEMP%\uitkwin'
  echo 'if exist "%D%" rd /s /q "%D%"'
  echo 'mkdir "%D%"'
  echo 'mkdir "%D%\src"'
  echo 'echo uitoolkit on Windows > "%D%\out.txt"'
  echo 'curl -s -o "%D%\src.tgz" %SRC%/src.tgz'
  echo 'if not exist "%D%\src.tgz" ('
  echo '  echo ===FAIL no source tree from %SRC% >> "%D%\out.txt"'
  echo '  echo ===DONE >> "%D%\out.txt"'
  echo '  curl -s --data-binary @"%D%\out.txt" %SRC%/result.txt'
  echo '  exit /b 1'
  echo ')'
  echo 'tar -xzf "%D%\src.tgz" -C "%D%\src"'
  echo 'curl -s -o "%D%\pkgs.txt" %SRC%/pkgs.txt'
  # One package at a time: fetched, run in its own directory, deleted.
  # A binary that will not download is reported rather than skipped — a
  # failed curl used to leave the previous one in place, and the guest
  # ran that and POSTed back a confident PASS from a stale binary.
  echo 'for /f "usebackq tokens=1,2" %%A in ("%D%\pkgs.txt") do ('
  echo '  del /q "%D%\t.exe" 2>nul'
  echo '  curl -s -o "%D%\t.exe" %SRC%/bin/%%A.test.exe'
  echo '  echo ===PKG %%A >> "%D%\out.txt"'
  echo '  if not exist "%D%\t.exe" ('
  echo '    echo ===FAIL %%A FETCH-FAILED >> "%D%\out.txt"'
  echo '  ) else ('
  echo '    pushd "%D%\src\%%B"'
  echo "    \"%D%\\t.exe\" -test.v$args >> \"%D%\\out.txt\" 2>&1"
  echo '    if errorlevel 1 echo ===FAIL %%A >> "%D%\out.txt"'
  echo '    popd'
  echo '  )'
  echo ')'
  echo 'del /q "%D%\t.exe" 2>nul'
  echo 'echo ===DONE >> "%D%\out.txt"'
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
# Longer than it was, because a full suite is two dozen binaries to fetch
# and run rather than one. The guest writes ===DONE last, so a result
# that arrives without it is a run that died partway and says so.
for _ in $(seq 1 360); do
  [ -s "$work/result.txt" ] && break
  sleep 5
done
if [ ! -s "$work/result.txt" ]; then
  echo "no result came back: is the guest logged in, and is Defender excluding %TEMP%\\uitkwin?" >&2
  exit 1
fi
tr -d '\r' < "$work/result.txt"

if ! grep -q '^===DONE' "$work/result.txt"; then
  echo >&2
  echo "the run did not finish: no ===DONE marker" >&2
  exit 1
fi
if grep -q '^===FAIL' "$work/result.txt"; then
  echo >&2
  echo "failed packages:" >&2
  tr -d '\r' < "$work/result.txt" | grep '^===FAIL' | sed 's/^===FAIL /  /' >&2
  exit 1
fi
