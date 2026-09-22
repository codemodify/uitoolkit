//go:build !linux

package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// OpenURI opens uri with the system's default application for it: open(1)
// on macOS, the shell's association on Windows.
func OpenURI(uri string, opts OpenURIOptions) error {
	if !validOpenURI(uri) {
		return ErrNoOpener
	}
	if strings.ToLower(os.Getenv(EnvOpenURI)) == "off" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", uri)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", uri)
	default:
		cmd = exec.Command("xdg-open", uri)
	}
	if err := cmd.Start(); err != nil {
		return ErrNoOpener
	}
	go cmd.Wait()
	return nil
}
