package widget

import (
	"os"
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// A file: URI belonging to another machine does not become a local path.
//
// This is the dangerous shape of a decoding bug: the old reader appended
// u.Path without looking at the authority, so "file://remote-host/etc/hosts"
// arrived as the path "/etc/hosts". An application that opened what it was
// handed therefore opened a *different* file — a wrong answer, not a
// failure, and one it had no way to notice. The remote entry is kept, whole,
// somewhere an application can see it and say it could not take it.
func TestARemoteFileURIIsNotALocalPath(t *testing.T) {
	const body = "file://remote-host/music/song.mp3\r\n"
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte(body))

	if len(e.Paths) != 0 {
		t.Errorf("a remote URI became the local path(s) %q", e.Paths)
	}
	if got := e.Remote; len(got) != 1 || got[0] != "file://remote-host/music/song.mp3" {
		t.Errorf("Remote = %q, want the one URI whole", got)
	}
}

// The three spellings of "this machine" all still decode, because a guard
// that rejected them would break every file drop there is.
//
// RFC 8089 allows an empty authority, "localhost", and the machine's own
// name. The last matters: a network-aware file manager writes it, and a
// drop from one is as local as any other.
func TestLocalFileURIsStillDecode(t *testing.T) {
	host, err := os.Hostname()
	if err != nil || host == "" {
		t.Skip("no hostname to test the third spelling with")
	}
	short := host
	if i := strings.IndexByte(host, '.'); i > 0 {
		short = host[:i]
	}
	body := strings.Join([]string{
		"# a comment, which this format allows",
		"file:///home/user/a b.mp3",
		"file://localhost/home/user/b.mp3",
		"file://LocalHost/home/user/c.mp3",
		"file://" + host + "/home/user/d.mp3",
		"file://" + strings.ToUpper(short) + "/home/user/e.mp3",
	}, "\r\n")
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte(body))

	want := []string{
		"/home/user/a b.mp3", // percent decoding survives
		"/home/user/b.mp3",
		"/home/user/c.mp3",
		"/home/user/d.mp3",
		"/home/user/e.mp3",
	}
	if len(e.Paths) != len(want) {
		t.Fatalf("Paths = %q, want %d local paths", e.Paths, len(want))
	}
	for i, w := range want {
		if e.Paths[i] != w {
			t.Errorf("Paths[%d] = %q, want %q", i, e.Paths[i], w)
		}
	}
	if len(e.Remote) != 0 {
		t.Errorf("a local URI was called remote: %q", e.Remote)
	}
}

// A percent-encoded URI is decoded, including one whose encoding hides the
// authority's separator. %2F in a path is not a slash that ends a host.
func TestURIListPercentDecoding(t *testing.T) {
	body := "file:///home/user/Caf%C3%A9%20%231.mp3\n"
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte(body))
	if len(e.Paths) != 1 || e.Paths[0] != "/home/user/Café #1.mp3" {
		t.Errorf("Paths = %q, want the decoded name", e.Paths)
	}
}

// A drop of several things sorts them: the local files are openable and the
// rest are not, and one list does not swallow the other. This is what a
// multiple selection out of a file manager that has a network share mounted
// actually looks like.
func TestAMixedURIListSeparatesLocalFromRemote(t *testing.T) {
	body := strings.Join([]string{
		"file:///music/one.mp3",
		"file://nas/music/two.mp3",
		"smb://nas/music/three.mp3",
		"https://example.invalid/four.mp3",
		"file:///music/five.mp3",
	}, "\r\n")
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte(body))

	wantLocal := []string{"/music/one.mp3", "/music/five.mp3"}
	wantRemote := []string{
		"file://nas/music/two.mp3",
		"smb://nas/music/three.mp3",
		"https://example.invalid/four.mp3",
	}
	if strings.Join(e.Paths, "|") != strings.Join(wantLocal, "|") {
		t.Errorf("Paths = %q, want %q", e.Paths, wantLocal)
	}
	if strings.Join(e.Remote, "|") != strings.Join(wantRemote, "|") {
		t.Errorf("Remote = %q, want %q", e.Remote, wantRemote)
	}
}

// An authority with a port is not this machine. "file://localhost:8080/x"
// is not a local file however local the name in front of the colon looks,
// and the old reader would have called it /x.
func TestAFileURIWithAPortIsNotLocal(t *testing.T) {
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte("file://localhost:8080/etc/hosts\n"))
	if len(e.Paths) != 0 {
		t.Errorf("Paths = %q, want none", e.Paths)
	}
	if len(e.Remote) != 1 {
		t.Errorf("Remote = %q, want the URI", e.Remote)
	}
}

// A Windows path that is not a URI is left alone rather than reported as a
// remote location. Its drive letter parses as a one-letter scheme, and
// calling "C:\music\song.mp3" a network address would have an application
// telling the user it could not take a local file.
func TestAWindowsPathIsNotMistakenForARemoteURI(t *testing.T) {
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte("C:\\music\\song.mp3\n"))
	if len(e.Paths) != 0 || len(e.Remote) != 0 {
		t.Errorf("Paths = %q, Remote = %q, want neither", e.Paths, e.Remote)
	}
}

// A file: URI with no path at all contributes no path. An empty string in
// Paths is a path to the current directory as far as most file APIs are
// concerned, which is worse than nothing.
func TestARelativeFileURIDoesNotBecomeAnEmptyPath(t *testing.T) {
	e := NewDropEvent(paintengine2d.Point{}, "text/uri-list", []byte("file:song.mp3\nfile://\n"))
	for _, p := range e.Paths {
		if p == "" {
			t.Fatalf("Paths holds an empty path: %q", e.Paths)
		}
	}
}
