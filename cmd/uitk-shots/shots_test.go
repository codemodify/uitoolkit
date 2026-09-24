package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteScreenshotsDistinct(t *testing.T) {
	dir := t.TempDir()
	if err := writeScreenshots(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range screenshotNames {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if st.Size() < 200 {
			t.Fatalf("%s too small: %d", name, st.Size())
		}
	}
	for _, name := range compareThumbNames {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if st.Size() < 200 {
			t.Fatalf("%s too small: %d", name, st.Size())
		}
	}
}
