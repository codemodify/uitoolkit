// Command uitk-skingen draws the toolkit's own demo skins and writes them
// into style/skins/.
//
// The art of a skin that ships with the toolkit is generated, not painted:
// it is then demonstrably the toolkit's own work, it can be re-cut at any
// scale, and the manifest that binds it cannot drift from it, because both
// come out of the same Go description (the public skingen package).
//
//	go run ./cmd/uitk-skingen              # rewrite style/skins/
//	go run ./cmd/uitk-skingen -o /tmp/s    # somewhere else
//	go run ./cmd/uitk-skingen -list        # what it would write
//
// TestSkinArtIsReproducible regenerates into a temporary directory and
// compares byte for byte, so art changed here and not committed fails.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codemodify/uitoolkit/skingen"
)

func main() {
	out := flag.String("o", filepath.Join("style", "skins"), "output directory")
	list := flag.Bool("list", false, "print the skins and their sheets, and exit")
	flag.Parse()

	plans := skingen.Plans()
	if *list {
		for _, p := range plans {
			for _, sh := range p.Sheets {
				fmt.Printf("%s\t%s\t%dx%d\t%d sprites\tpixelated=%v\n",
					p.Name, sh.Name, sh.W, sh.H, len(sh.Cells), sh.Pixelated)
			}
		}
		return
	}
	for _, p := range plans {
		if err := skingen.Write(*out, p); err != nil {
			fmt.Fprintf(os.Stderr, "uitk-skingen: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", filepath.Join(*out, p.Name))
	}
}
