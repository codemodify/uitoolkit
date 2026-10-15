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
# It had two of its own, both found by applications the morning after
# it shipped and both pinned now by the root package's
# TestTheSkipReporterReportsSkipsAndFailures, against the fixture in
# internal/skipfixture: it reported the *first* line a test logged
# rather than its skip reason, and it hid a failed test run behind
# "0 skipped" and exit 0 — which is this script's own version of the
# mistake it is for.
#
# It then guessed the *last* line instead, and a cleanup that logs
# after t.Skip made that one wrong too (both players, the next
# morning). Guessing by position cannot work: `t.Skip` is `t.Log` plus
# `t.SkipNow`, so its message is an ordinary log line, and a test may
# talk on either side of it.
#
# What does identify it is where it came from. Every log line carries
# the file and line of the call that made it, and the skip's call is a
# call to Skip, which is in the test sources to be found. So the sites
# are collected first and a logged line that came from one of them is
# the reason. A test whose lines match no site, or several, has all of
# them printed instead of a guess, and a test that logged nothing at
# all — `t.SkipNow()` logs nothing — is reported as having said so.
#
# All of that is per *package*. Two packages may each have a TestFoo,
# and keying what was said by test name alone handed the second one
# the first one's reason — a silent SkipNow reported with somebody
# else's words (a video player, the release after). `go test` prints
# each package's output as a block ending in its `ok <pkg>` line, so
# the block is the scope, and the package is shown whenever more than
# one of them had something to report.
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

# The run's own status is the first thing reported, because a reporter
# that hides a failed invocation behind "0 skipped" is doing the exact
# thing it exists to catch. It did: `tools/skips.sh ./nosuchpkg/` used
# to print "0 skipped" and exit 0 while tools/test.sh exited 1, and a
# video player caught it the morning after this script shipped.
status=0
out=$(tools/test.sh -v "$@" 2>&1) || status=$?

# Where a Skip is called from, as file:line, which is what the log
# line it writes is stamped with. Collected from the test sources
# rather than guessed at from the output's shape.
# The `|| true` is for a tree with no Skip in it at all: grep exits 1,
# and under `set -o pipefail` that would end the script before it
# printed anything — which is how the first version of this line hid a
# failed run all over again, in a fixture that had no test sources.
sites=$(grep -rn --include='*_test.go' -E '\.(Skip|Skipf|SkipNow)\(' . 2>/dev/null |
  awk -F: '{ n = split($1, part, "/"); print part[n] ":" $2 }' | sort -u || true)

# The log lines are emitted while the test runs and the verdict is
# printed afterwards — for a subtest, in a block under its parent — so
# they are matched up by name rather than by being next to each other.
# The package's own line comes last of all, which is why the rows are
# held until the end rather than printed as they are found.
echo "$out" | awk -v sites="$sites" '
  function forget(   k) {
    for (k in said) delete said[k]
    for (k in count) delete count[k]
    for (k in from) delete from[k]
    for (k in fromCount) delete fromCount[k]
    name = ""
  }
  BEGIN {
    parts = split(sites, s, "\n")
    for (i = 1; i <= parts; i++) if (s[i] != "") site[s[i]] = 1
    pending = 1
  }
  /^ *=== (RUN|CONT|NAME) / { name = $3; next }
  /^ *[A-Za-z0-9_]+\.go:[0-9]+: / {
    line = $0; sub(/^ */, "", line)
    if (name == "") next
    said[name] = (name in said) ? said[name] "\n" line : line
    count[name]++
    where = line; sub(/: .*/, "", where)
    if (where in site) {
      from[name] = (name in from) ? from[name] "\n" line : line
      fromCount[name]++
    }
    next
  }
  /^ *--- SKIP: / {
    who = $0; sub(/^ *--- SKIP: /, "", who); sub(/ \(.*/, "", who)
    rows++
    rowName[rows] = who
    if (!(who in said))            rowWhy[rows] = "(said nothing)"
    else if (fromCount[who] == 1)  rowWhy[rows] = from[who]
    else if (count[who] == 1)      rowWhy[rows] = said[who]
    else { rowWhy[rows] = said[who]; rowMany[rows] = count[who]; unsure++ }
    next
  }
  # A package has finished: everything buffered since the last one was
  # its, and nothing it said may reach the next.
  /^(ok|FAIL|\?)[ \t]+[^ \t[]/ {
    for (i = pending; i <= rows; i++) rowPkg[i] = $2
    pending = rows + 1
    if (rows >= pending - 1 && pending > 1) pkgs[$2] = 1
    forget()
    next
  }
  END {
    for (i = pending; i <= rows; i++) rowPkg[i] = "(no package line)"
    for (i = 1; i <= rows; i++) seen[rowPkg[i]] = 1
    many = 0
    for (p in seen) many++
    last = ""
    for (i = 1; i <= rows; i++) {
      # The package as a heading above its rows, and never as an
      # indent on them: a row begins with the name of the test it is
      # about, which is what anything reading this output keys on.
      if (many > 1 && rowPkg[i] != last) { printf "%s\n", rowPkg[i]; last = rowPkg[i] }
      if (!(i in rowMany)) { printf "%-68s %s\n", rowName[i], rowWhy[i]; continue }
      print rowName[i]
      split(rowWhy[i], lines, "\n")
      for (j = 1; j <= rowMany[i]; j++) printf "    %s\n", lines[j]
    }
    printf "\n%d skipped\n", rows
    if (unsure) printf "%d of them logged lines this could not trace to a Skip call; every line is shown for those\n", unsure
  }
'

if [ "$status" != 0 ]; then
  echo
  echo "the test run itself failed (tools/test.sh exited $status) —"
  echo "the skips above, if any, are from a run that did not finish:"
  echo "$out" | grep -E "^(FAIL|# |.*\[build failed\]|--- FAIL)" | head -20 || true
fi
exit "$status"
