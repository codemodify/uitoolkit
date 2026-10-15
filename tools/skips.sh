#!/bin/bash
# skips.sh [go test args...] — every test that skipped, with its reason.
#
#   tools/skips.sh ./app/
#   tools/skips.sh -race ./app/ ./widgets/
#
# A skip is what green looks like when nothing ran, and three of them
# hid real coverage for two releases each before an application read
# the output more carefully than this repository did:
#
#   - TestAFramedWindowComesBackTheWayItWent skipped on every -race run
#     ("the clock beat the painter") and reported PASS.
#   - TestAMaximizedWindowDoesNotStandBehindItsOwnTurn skipped in *both*
#     its subtests while the parent printed PASS and the command exited
#     zero. `go test` prints a subtest's result indented, so a grep for
#     '^--- SKIP' does not match it, which is how the release note that
#     said "the remaining skips are all no-display ones" came to be
#     written. This script does not have that bug.
#
# What is left after those were fixed, and what is expected here:
#
#   no DISPLAY / no WAYLAND_DISPLAY     tools/test-display.sh covers it
#   no display (a real GPU device)      the same
#   this build has no ... (font_...)    tools/test-fonts.sh covers it
#
# Anything else is a test that is not running and is not saying why in
# one of those terms. Read it before believing the suite is green.
set -euo pipefail
cd "$(dirname "$0")/.."
out=$(tools/test.sh -v "$@" 2>&1) || true
# The reason is logged while the test runs and the verdict is printed
# afterwards — for a subtest, in a block under its parent — so they are
# matched up by name rather than by being next to each other.
echo "$out" | awk '
  /^ *=== (RUN|CONT|NAME) / { name = $3; next }
  /^ *[A-Za-z0-9_]+\.go:[0-9]+: / {
    line = $0; sub(/^ */, "", line)
    if (name != "" && !(name in why)) why[name] = line
    next
  }
  /^ *--- SKIP: / {
    who = $0; sub(/^ *--- SKIP: /, "", who); sub(/ \(.*/, "", who)
    printf "%-68s %s\n", who, (who in why ? why[who] : "(no reason given)")
    n++
    next
  }
  END { printf "\n%d skipped\n", n }
'
