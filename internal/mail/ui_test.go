package mail

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func TestMailAppPaints(t *testing.T) {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(MailApp(a, w))
	a.PumpOnce()
	PrepareShot(w, -1)
	a.PumpOnce()

	var tables, trees, areas, bars, menus int
	widget.Walk(w.Content(), func(c widget.Component) {
		switch c.(type) {
		case *widgets.TableView:
			tables++
		case *widgets.TreeView:
			trees++
		case *widgets.TextArea:
			areas++
		case *widgets.StatusBar:
			bars++
		case *widgets.MenuBar:
			menus++
		}
	})
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TableView); ok && tv.CellText != nil && tv.RowCount > 0 {
			subj := tv.CellText(0, 2)
			t.Logf("row0 subject %q selected=%d rows=%d", subj, tv.Selected, tv.RowCount)
			if subj == "" {
				t.Error("empty first subject")
			}
		}
	})
	if tables < 1 || trees < 1 || areas < 1 || bars < 1 || menus < 1 {
		t.Fatalf("chrome table=%d tree=%d area=%d status=%d menu=%d", tables, trees, areas, bars, menus)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "mail.png")
	if err := w.WritePNG(path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() < 2000 {
		t.Fatalf("png too small: %d", st.Size())
	}
	w.Close()
}

func TestComposeAppPaints(t *testing.T) {
	sock, stop, err := StartDemo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	cli, err := DialWait(sock, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	a := uitoolkit.New(uitoolkit.Options{Look: style.LightLook(), Headless: true})
	w, err := OpenCompose(a, cli, ComposeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.png")
	if err := w.WritePNG(path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() < 1000 {
		t.Fatalf("compose png too small: %d", st.Size())
	}
	w.Close()
}

func TestWriteScreenshotsDistinct(t *testing.T) {
	dir := t.TempDir()
	if err := WriteScreenshots(dir); err != nil {
		t.Fatal(err)
	}
	seen := map[int64]string{}
	for _, name := range ScreenshotNames {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if st.Size() < 200 {
			t.Fatalf("%s too small: %d", name, st.Size())
		}
		if other, ok := seen[st.Size()]; ok {
			t.Fatalf("%s same size as %s (%d) — likely duplicate", name, other, st.Size())
		}
		seen[st.Size()] = name
	}
}
