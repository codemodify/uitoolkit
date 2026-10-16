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
// than by this repository. The script was written to end that, and
// applications then found three defects in it on two consecutive
// mornings: it reported the first line a test logged as the skip
// reason; it hid a failed test run behind "0 skipped" and exit 0, which
// is the very mistake it exists to catch; and, having switched to the
// last line instead, it reported a cleanup's log as the reason.
//
// Guessing by position cannot work, because a test may talk on either
// side of its skip. What identifies the line is where it came from:
// every log line carries the file and line of the call that wrote it,
// and the skip's call is a call to Skip, which is in the test sources
// to be found. A line that came from no Skip call, or a test with
// several that did, has everything it said printed instead of a guess,
// and t.SkipNow — which logs nothing — is reported as nothing said.
// All of it runs against the fixture in internal/skipfixture.
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
	for _, want := range []string{"TestOuter/logged", "TestOuter/plain", "TestOuter/silent", "TestOuter/shared"} {
		if !strings.Contains(out, want) {
			t.Errorf("%s is not in the report:\n%s", want, out)
		}
	}
	if strings.Contains(out, "TestThatRuns") {
		t.Errorf("a test that ran is reported as skipped:\n%s", out)
	}
	if !strings.Contains(out, "5 skipped") {
		t.Errorf("the count is not five:\n%s", out)
	}
	// One package, so it is not named: the heading is there to tell
	// two apart and is noise when there is only one.
	if strings.Contains(out, "github.com/codemodify/uitoolkit/internal/skipfixture\n") {
		t.Errorf("a single package is named in its own report:\n%s", out)
	}
	// The child that talked on both sides of its reason is reported
	// with the reason and nothing else. Guessing the first line lost
	// it to the setup log and guessing the last lost it to the
	// cleanup; both were reported from outside, a morning apart.
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "TestOuter/logged") {
			continue
		}
		if !strings.Contains(line, "the capability this needs is absent") {
			t.Errorf("the skip reason is not what is reported:\n  %s", strings.TrimSpace(line))
		}
		for _, noise := range []string{"setup talk", "teardown talk"} {
			if strings.Contains(line, noise) {
				t.Errorf("%q is reported as the skip reason:\n  %s", noise, strings.TrimSpace(line))
			}
		}
	}
	if strings.Contains(out, "setup talk") || strings.Contains(out, "teardown talk") {
		t.Errorf("the report carries lines that are not the reason:\n%s", out)
	}
	// A test that gave no reason is said to have given none, rather
	// than being handed somebody else's line.
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "TestOuter/silent") {
			continue
		}
		if !strings.Contains(line, "(said nothing)") {
			t.Errorf("a test that logged nothing is reported with a reason:\n  %s", strings.TrimSpace(line))
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

	// Two packages with a test of the same name keep their own words.
	// Keying what was said by test name alone handed the second one
	// the first one's reason, so a child that called t.SkipNow and
	// logged nothing was reported with another package's line.
	t.Run("two packages sharing a test name", func(t *testing.T) {
		out, code := run("-tags", "skipfixture",
			"./internal/skipfixture/", "./internal/skipfixture2/")
		if code != 0 {
			t.Fatalf("exited %d:\n%s", code, out)
		}
		if !strings.Contains(out, "6 skipped") {
			t.Errorf("the count is not six:\n%s", out)
		}
		// Both packages are named, since there is more than one.
		for _, want := range []string{
			"github.com/codemodify/uitoolkit/internal/skipfixture",
			"github.com/codemodify/uitoolkit/internal/skipfixture2",
		} {
			if !strings.Contains(out, want+"\n") {
				t.Errorf("%s is not named in the report:\n%s", want, out)
			}
		}
		// Both packages have a TestOuter/shared. The first skips with
		// a reason, the second says nothing at all — so the only
		// line the second could be reported with is the first's. A
		// row begins with the name of its test, at the left margin,
		// because that is what reads this output.
		var shared []string
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "TestOuter/shared ") {
				shared = append(shared, line)
			}
		}
		if len(shared) != 2 {
			t.Fatalf("expected two rows for the shared name, got %d:\n%s", len(shared), out)
		}
		if !strings.Contains(shared[0], "only this package is missing anything") {
			t.Errorf("the first package's reason is not reported:\n  %s", strings.TrimSpace(shared[0]))
		}
		if !strings.Contains(shared[1], "(said nothing)") {
			t.Errorf("the silent child in the other package borrowed its words:\n  %s", strings.TrimSpace(shared[1]))
		}
	})

	// A second attempt at the same test starts with nothing said.
	// Under -count=2 one test runs twice inside one package block,
	// and the second attempt was being reported with whatever the
	// first one said — invisible for an ordinary test, which says
	// the same thing both times.
	t.Run("the same test run twice", func(t *testing.T) {
		out, code := run("-tags", "skipfixture", "-count=2", "./internal/skipfixture/")
		if code != 0 {
			t.Fatalf("exited %d:\n%s", code, out)
		}
		var repeated []string
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "TestRepeated ") {
				repeated = append(repeated, line)
			}
		}
		if len(repeated) != 2 {
			t.Fatalf("expected two attempts, got %d:\n%s", len(repeated), out)
		}
		if !strings.Contains(repeated[0], "only the first attempt has anything to say") {
			t.Errorf("the first attempt is not reported with what it said:\n  %s", strings.TrimSpace(repeated[0]))
		}
		if !strings.Contains(repeated[1], "(said nothing)") {
			t.Errorf("the second attempt borrowed the first one's words:\n  %s", strings.TrimSpace(repeated[1]))
		}
	})

	// And it works in a tree that has no Skip anywhere, which this
	// repository can never be and so can never check from here.
	// Looking for the Skip call sites broke exactly that for a moment:
	// grep exits 1 when it matches nothing, and under `set -o
	// pipefail` the script ended before printing a thing — silently
	// producing nothing, for the third time in this script's short
	// life. A video player's probe caught it; this is the same check,
	// kept here.
	t.Run("a tree with no skips in it", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "tools"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"skips.sh", "test.sh", "testenv.sh"} {
			from := filepath.Join(repo, "tools", name)
			body, err := os.ReadFile(from)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "tools", name), body, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module quiet\n\ngo 1.22.2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "quiet"), 0o755); err != nil {
			t.Fatal(err)
		}
		body := "package quiet\n\nimport \"testing\"\n\nfunc TestNothingSkips(t *testing.T) {}\n"
		if err := os.WriteFile(filepath.Join(dir, "quiet", "quiet_test.go"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(filepath.Join(dir, "tools", "skips.sh"), "./quiet/")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("the reporter failed in a tree with no skips: %v\n%s", err, out)
		}
		if !strings.Contains(string(out), "0 skipped") {
			t.Errorf("a tree with no skips does not report none:\n%s", out)
		}
	})
}
