package uitoolkit

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// tools/skips.sh is how this repository finds out whether a green run
// checked anything, so what it claims has to be true.
//
// Three releases running, a test that skipped was taken for a test that
// passed — each time found by an application reading the output rather
// than by this repository. The script was written to end that, and two
// applications found two defects in it the morning after it shipped: it
// reported the first line a test logged as the skip reason, and it hid
// a failed test run behind "0 skipped" and exit 0 — which is the very
// mistake it exists to catch. Both are pinned here, against the fixture
// in internal/skipfixture.
func TestTheSkipReporterReportsSkipsAndFailures(t *testing.T) {
	// The script is a bash script that runs tools/test.sh, which runs
	// tools/testenv.sh, which needs a dbus-daemon to start a private
	// session bus on. So: Linux, with both of those to hand. Windows
	// cannot exec a .sh at all, and macOS has no dbus-daemon — the
	// Darwin rig runs go test over ssh and never goes through
	// testenv.sh, which is why neither rig could run this and both
	// said so the first time it was sent to them.
	if runtime.GOOS != "linux" {
		t.Skipf("tools/skips.sh runs through testenv.sh, which is Linux-only; this is %s", runtime.GOOS)
	}
	for _, need := range []string{"bash", "dbus-daemon", "go"} {
		if _, err := exec.LookPath(need); err != nil {
			t.Skipf("no %s on the path, which tools/skips.sh needs", need)
		}
	}
	repo, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repo, "tools", "skips.sh")
	if _, err := os.Stat(script); err != nil {
		t.Skipf("the script this is about is not here: %v", err)
	}

	run := func(args ...string) (string, int) {
		t.Helper()
		cmd := exec.Command(script, args...)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if ee, ok := err.(*exec.ExitError); ok {
			return string(out), ee.ExitCode()
		} else if err != nil {
			t.Fatalf("running %s: %v", script, err)
		}
		return string(out), 0
	}

	out, code := run("-tags", "skipfixture", "./internal/skipfixture/")
	if code != 0 {
		t.Fatalf("a fixture that only skips exited %d:\n%s", code, out)
	}
	// Both children. A grep for a line beginning "--- SKIP" finds
	// neither, because go test indents a subtest's verdict under its
	// parent — which is how a release note came to say that the only
	// skips left were the no-display ones.
	for _, want := range []string{"TestOuter/logged", "TestOuter/plain"} {
		if !strings.Contains(out, want) {
			t.Errorf("%s is not in the report:\n%s", want, out)
		}
	}
	if strings.Contains(out, "TestThatRuns") {
		t.Errorf("a test that ran is reported as skipped:\n%s", out)
	}
	if !strings.Contains(out, "2 skipped") {
		t.Errorf("the count is not two:\n%s", out)
	}
	// And the reason is the reason, not the first thing the test said.
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "TestOuter/logged") {
			continue
		}
		if strings.Contains(line, "setup talk") {
			t.Errorf("the setup log is reported as the skip reason:\n  %s", strings.TrimSpace(line))
		}
		if !strings.Contains(line, "the capability this needs is absent") {
			t.Errorf("the skip reason is not reported:\n  %s", strings.TrimSpace(line))
		}
	}

	// A run that did not finish is a failure, however few skips it
	// found on the way.
	out, code = run("./nosuchpackage/")
	if code == 0 {
		t.Errorf("a failed test run was reported as success:\n%s", out)
	}
	if !strings.Contains(out, "the test run itself failed") {
		t.Errorf("a failed test run does not say so:\n%s", out)
	}
}
