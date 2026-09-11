package apptest

import (
	"testing"
)

func TestDriverGalleryAndMail(t *testing.T) {
	rs := Run(Options{Apps: "all", Short: testing.Short()})
	for _, r := range rs {
		t.Log(r.String())
		if r.Err != nil {
			t.Errorf("%s/%s: %v", r.App, r.Step, r.Err)
		}
	}
}

func TestDriverGalleryOnly(t *testing.T) {
	rs := Run(Options{Apps: "gallery", Short: true})
	if Failed(rs) {
		for _, r := range rs {
			if r.Err != nil {
				t.Errorf("%s", r)
			}
		}
	}
}
