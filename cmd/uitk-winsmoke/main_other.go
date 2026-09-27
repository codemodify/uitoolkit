//go:build !windows

// Command uitk-winsmoke reports on the Win32 backend, and so only runs
// on Windows. It is built from Linux with GOOS=windows and carried into
// a VM; see docs/windows.md.
package main

import "fmt"

func main() { fmt.Println("uitk-winsmoke runs on Windows; build it with GOOS=windows") }
