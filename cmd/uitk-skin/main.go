// Command uitk-skin is a skin author's tool.
//
//	uitk-skin lint ~/.config/uitoolkit/skins/mine     # a directory with skin.json
//	uitk-skin lint mine.uskin                          # or the archive
//	uitk-skin lint nocturne deck                       # or skins by name
//	uitk-skin lint -strict mine                        # warnings fail too
//
// lint prints the loader's refusal, keyed by the manifest key it came from,
// or — for a skin that loads — what its author would want to know before
// shipping it: states a control falls back from, sprites nothing binds, art
// too small for 2× (style.LintSkin). It exits 1 when a skin does not load,
// and with -strict when there is anything to say at all.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/codemodify/uitoolkit/style"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "lint" {
		fmt.Fprintln(os.Stderr, "usage: uitk-skin lint [-strict] <skin dir | file.uskin | name>...")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	strict := fs.Bool("strict", false, "exit 1 on warnings as well as errors")
	_ = fs.Parse(os.Args[2:])
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: uitk-skin lint [-strict] <skin dir | file.uskin | name>...")
		os.Exit(2)
	}
	failed := false
	for _, arg := range fs.Args() {
		sk, err := load(arg)
		if err != nil {
			fmt.Println(err)
			failed = true
			continue
		}
		warnings := style.LintSkin(sk)
		for _, w := range warnings {
			fmt.Println(w)
		}
		if len(warnings) == 0 {
			fmt.Printf("%s: ok\n", sk.Name)
		}
		if *strict && len(warnings) > 0 {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// load reads a skin from a path when there is something there, and by name
// — a user skin, then one the toolkit ships — when there is not.
func load(arg string) (*style.Skin, error) {
	if _, err := os.Stat(arg); err == nil {
		return style.LoadSkinFile(arg)
	}
	if sk, ok := style.LoadSkin(arg); ok {
		return sk, nil
	}
	return nil, fmt.Errorf("%s: no skin by that name, and no directory or .uskin there", arg)
}
