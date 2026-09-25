// Command uitest-driver script-drives the widget showcase and Settings:
// the two windows here with enough chrome to be worth driving.
//
// It needs no display and opens nothing of the user's. See docs/testing.md.
//
//	go run ./cmd/uitest-driver
//	go run ./cmd/uitest-driver -short
//	go run ./cmd/uitest-driver -app=showcase
//	go run ./cmd/uitest-driver -compare
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/codemodify/uitoolkit/internal/apptest"
	"github.com/codemodify/uitoolkit/internal/uitest"
)

func main() {
	app := flag.String("app", "all", "showcase, settings, or all")
	short := flag.Bool("short", false, "skip compose and extra resize/scroll passes")
	compare := flag.Bool("compare", false, "print Avalonia/Qt/GTK chrome checklist and exit")
	flag.Parse()

	if *compare {
		failed := 0
		rows := uitest.CompareReport()
		for _, r := range rows {
			fmt.Println(r.String())
			if r.Err != nil {
				failed++
			}
		}
		if failed > 0 {
			fmt.Fprintf(os.Stderr, "%d chrome norm(s) failed\n", failed)
			os.Exit(1)
		}
		fmt.Printf("%d chrome norm(s) ok  (see docs/compare.md)\n", len(rows))
		return
	}

	rs := apptest.Run(apptest.Options{Apps: *app, Short: *short})
	failed := 0
	for _, r := range rs {
		fmt.Println(r.String())
		if r.Err != nil {
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d step(s) failed\n", failed)
		os.Exit(1)
	}
	fmt.Printf("%d step(s) ok\n", len(rs))
}
