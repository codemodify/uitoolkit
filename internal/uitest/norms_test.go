package uitest

import "testing"

func TestDesktopChromeNorms(t *testing.T) {
	for _, n := range ChromeNorms() {
		n := n
		t.Run(n.ID, func(t *testing.T) {
			if err := n.Check(); err != nil {
				t.Fatalf("%s (%s): %v", n.Want, n.Peer, err)
			}
		})
	}
}
