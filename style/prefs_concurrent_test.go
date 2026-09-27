package style

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// Saving preferences has to stay atomic when more than one writer is at
// it: two Settings windows, or one application saving from two
// goroutines.
//
// Every writer used to use the same "<path>.tmp", which defeats the
// point of a temporary file — two saves truncate the same inode and
// interleave their bytes, one renames the other's half-written file into
// place, and a failed save removes a file its neighbour is about to
// publish. The file that ends up where the preferences belong is then
// not valid JSON.
func TestConcurrentSavesNeverPublishHalfAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "look.json")
	var writeErrs, badReads atomic.Int64

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			for n := 0; n < 200; n++ {
				// Different lengths, so an interleaving shows up as
				// invalid JSON rather than as a plausible file.
				v := map[string]string{"theme": strings.Repeat("a", i*100+1)}
				if err := writeJSONFile(path, v); err != nil {
					writeErrs.Add(1)
				}
			}
		}(i)
	}
	// A reader racing the writers: whatever it sees must parse.
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for n := 0; n < 2000; n++ {
			b, err := os.ReadFile(path)
			if err != nil || len(b) == 0 {
				continue // not published yet
			}
			var out map[string]string
			if json.Unmarshal(b, &out) != nil {
				badReads.Add(1)
			}
		}
	}()
	close(start)
	wg.Wait()

	if n := writeErrs.Load(); n != 0 {
		t.Errorf("%d saves failed; concurrent writers must not collide", n)
	}
	if n := badReads.Load(); n != 0 {
		t.Errorf("%d reads saw invalid JSON where the preferences belong", n)
	}
	// And no temporary files are left behind.
	ents, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("a temporary file survived: %s", e.Name())
		}
	}
}
