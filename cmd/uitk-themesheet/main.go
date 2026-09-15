// Command uitk-themesheet renders theme sheets: every control of a theme
// pack in every interesting state, offscreen, as PNG. It is the quickest way
// to see what an engine paints. -frames renders the window-frames sheet
// instead: windows whose frame uitoolkit draws in the pack, active, in the
// backdrop, maximized, with the close button under the pointer, and with the
// app's own title bar.
//
//	go run ./cmd/uitk-themesheet -theme win95 -o /tmp/sheets
//	go run ./cmd/uitk-themesheet -all -scale 2 -o /tmp/sheets
//	go run ./cmd/uitk-themesheet -frames -all -o /tmp/frames
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/internal/themesheet"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	var themes multi
	flag.Var(&themes, "theme", "theme pack id (repeatable, or comma separated)")
	all := flag.Bool("all", false, "render every built-in pack")
	out := flag.String("o", ".", "output directory")
	scale := flag.Float64("scale", 1, "display scale (1, 1.5, 2)")
	list := flag.Bool("list", false, "print every built-in pack (id, year, lineage, engine, label, summary) tab-separated and exit")
	frames := flag.Bool("frames", false, "render the window-frames sheet (<pack>-frames.png) instead of the controls sheet")
	flag.Parse()
	if *list {
		for _, p := range style.ListBuiltinThemes() {
			engine := p.Look().Engine().ID()
			fmt.Printf("%s\t%d\t%s\t%s\t%s\t%s\n", p.Name, p.Year, p.Lineage, engine, p.Display(), p.Summary)
		}
		return
	}
	if *all {
		themes = append(themes, style.AllBuiltinThemeNames()...)
	}
	if len(themes) == 0 {
		fmt.Fprintln(os.Stderr, "uitk-themesheet: pass -theme ID or -all")
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, id := range themes {
		pack, ok := style.LoadTheme(id)
		if !ok {
			fmt.Fprintf(os.Stderr, "uitk-themesheet: unknown theme %q\n", id)
			os.Exit(1)
		}
		var lk style.LookAndFeel = pack.Look()
		if *scale != 1 {
			lk = style.WithScale(lk, float32(*scale))
		}
		engine := style.LookTokens(lk).Engine
		if engine == "" {
			engine = "base"
		}
		title := fmt.Sprintf("%s  ·  %s  ·  engine %s", pack.Display(), pack.Name, engine)
		if pack.Year > 0 {
			title += fmt.Sprintf("  ·  %d", pack.Year)
		}
		var img *paintengine2d.Image
		name := pack.Name
		if *frames {
			frame := "adapted in-app frame"
			if style.NativeDecoration(pack.Look()) {
				frame = "native frame"
			}
			img = themesheet.RenderFrames(pack.Look(), float32(*scale), title+"  ·  "+frame)
			name += "-frames"
		} else {
			img = themesheet.Render(lk, title)
		}
		if *scale != 1 {
			name += fmt.Sprintf("@%gx", *scale)
		}
		path := filepath.Join(*out, name+".png")
		if err := img.WritePNGFile(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(path)
	}
}

type multi []string

func (m *multi) String() string { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error {
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			*m = append(*m, s)
		}
	}
	return nil
}
