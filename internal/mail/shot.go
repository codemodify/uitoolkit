package mail

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// ScreenshotNames are the Mail PNGs written by WriteScreenshots.
var ScreenshotNames = []string{
	"mail-dark.png", "mail-light.png", "mail-classic.png", "mail-compose.png", "mail-prefs.png",
}

// WriteScreenshots paints the Mail dogfood frames into dir.
func WriteScreenshots(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	sock, stop, err := StartDemo(context.Background())
	if err != nil {
		return err
	}
	defer stop()
	cli, err := DialWait(sock, 2*time.Second)
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := writeMailShot(cli, filepath.Join(dir, "mail-dark.png"), false, LayoutVertical, -1); err != nil {
		return err
	}
	if err := writeMailShot(cli, filepath.Join(dir, "mail-light.png"), true, LayoutVertical, -1); err != nil {
		return err
	}
	if err := writeMailShot(cli, filepath.Join(dir, "mail-classic.png"), false, LayoutClassic, 0); err != nil {
		return err
	}
	if err := writeComposeShot(cli, filepath.Join(dir, "mail-compose.png")); err != nil {
		return err
	}
	return writePrefsShot(cli, filepath.Join(dir, "mail-prefs.png"))
}

func writeMailShot(cli *Client, path string, light bool, layout LayoutMode, menu int) error {
	look := style.DarkLook()
	if light {
		look = style.LightLook()
	}
	a := uitoolkit.New(uitoolkit.Options{Look: look, Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, Headless: true,
	})
	if err != nil {
		return err
	}
	w.SetContent(Open(a, w, cli, AppOptions{
		Light: light, Layout: layout, ShowFilter: true,
	}))
	a.PumpOnce()
	PrepareShot(w, menu)
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writeComposeShot(cli *Client, path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := OpenCompose(a, cli, ComposeOptions{})
	if err != nil {
		return err
	}
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}

func writePrefsShot(cli *Client, path string) error {
	a := uitoolkit.New(uitoolkit.Options{Look: style.DarkLook(), Headless: true})
	w, err := OpenPrefs(a, cli)
	if err != nil {
		return err
	}
	a.PumpOnce()
	if err := w.WritePNG(path); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	w.Close()
	return nil
}
