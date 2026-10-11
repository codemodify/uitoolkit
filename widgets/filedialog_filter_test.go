package widgets

import "testing"

// **The fallback chooser understands every pattern a filter carries.**
//
// FileOptions.Filter has always been documented as globs separated by
// spaces or ';', and the native chooser is handed exactly those. The
// fallback parsed the field itself and read the whole of it as one
// filename suffix, so "*.mp4 *.mkv" matched nothing: a video player
// with a multi-format filter saw a folder with no videos in it wherever
// no native chooser was available. Reported against media-player-video.
func TestTheFallbackChooserTakesEveryPatternInAFilter(t *testing.T) {
	ents := []FileInfo{
		{Name: "Movies", Dir: true},
		{Name: "first.mp4"},
		{Name: "second.mkv"},
		{Name: "notes.txt"},
		{Name: "THIRD.MP4"},
	}
	names := func(in []FileInfo) []string {
		var out []string
		for _, e := range in {
			out = append(out, e.Name)
		}
		return out
	}
	eq := func(got []string, want ...string) bool {
		if len(got) != len(want) {
			return false
		}
		for i := range got {
			if got[i] != want[i] {
				return false
			}
		}
		return true
	}
	for _, tc := range []struct {
		filter string
		want   []string
	}{
		{"*.mp4", []string{"Movies", "first.mp4", "THIRD.MP4"}},
		{"*.mp4 *.mkv", []string{"Movies", "first.mp4", "second.mkv", "THIRD.MP4"}},
		{"*.mp4;*.mkv", []string{"Movies", "first.mp4", "second.mkv", "THIRD.MP4"}},
		{"*.mp4,*.mkv", []string{"Movies", "first.mp4", "second.mkv", "THIRD.MP4"}},
		{"", []string{"Movies", "first.mp4", "second.mkv", "notes.txt", "THIRD.MP4"}},
		{"*", []string{"Movies", "first.mp4", "second.mkv", "notes.txt", "THIRD.MP4"}},
		{"*.*", []string{"Movies", "first.mp4", "second.mkv", "notes.txt", "THIRD.MP4"}},
		// A real glob, not just a suffix.
		{"f?rst.mp4", []string{"Movies", "first.mp4"}},
		// Nothing that parses as a filter shows everything rather than
		// an empty folder, which is what it did before.
		{"videos", []string{"Movies", "first.mp4", "second.mkv", "notes.txt", "THIRD.MP4"}},
	} {
		if got := names(filterEntries(ents, tc.filter)); !eq(got, tc.want...) {
			t.Errorf("filter %q kept %v, want %v", tc.filter, got, tc.want)
		}
	}
}

// A directory is never filtered out: the chooser has to stay navigable
// whatever the filter says.
func TestAFilterNeverHidesADirectory(t *testing.T) {
	ents := []FileInfo{{Name: "Movies", Dir: true}, {Name: "a.txt"}}
	got := filterEntries(ents, "*.mp4")
	if len(got) != 1 || got[0].Name != "Movies" {
		t.Errorf("a filter matching nothing kept %v, want just the directory", got)
	}
}
