//go:build windows

package platform

import (
	"os"
	"strconv"
	"strings"
	"unsafe"
)

// Painting the desktop's frame in a window's own colours.
//
// [WindowFrame.SetPalette] names a KDE colour-scheme file, and that
// looked like the thing blocking this: Windows has no colour-scheme
// files, it has three COLORREFs through DwmSetWindowAttribute, so the
// signature seemed wrong for the platform and the method answered false
// rather than pretend.
//
// It is not wrong. The file is the *source* of the colours, not a
// KDE-only mechanism — a KConfig key file with a [WM] group holding
// activeBackground, activeForeground and inactiveBackground as "r,g,b".
// Reading it and handing those three to DWM is exactly what the method
// asks for, and platform already parses this format for the title-bar
// preferences (parseINI). Nothing about the seam had to change.
//
// Windows 11 build 22000 is where the attributes arrive. Before that
// DwmSetWindowAttribute answers E_INVALIDARG, which is why FramePalette
// is probed for rather than assumed: the capability has to say what this
// machine will do, not what the code knows how to ask for.

const (
	dwmwaBorderColor  = 34
	dwmwaCaptionColor = 35
	dwmwaTextColor    = 36
	// DWMWA_COLOR_DEFAULT: give the colour back to the desktop.
	dwmColorDefault = 0xFFFFFFFF
)

// winDwmColor sets one of the caption attributes, and reports whether
// this Windows knows it.
func (s *winSurface) winDwmColor(attr uintptr, colour uint32) bool {
	if s.hwnd == 0 {
		return false
	}
	r, _, _ := procDwmSetAttribute.Call(s.hwnd, attr,
		uintptr(unsafe.Pointer(&colour)), unsafe.Sizeof(colour))
	return r == 0
}

// winCaptionColorsWork probes once whether this machine takes the caption
// colour at all, by setting it to the default — which changes nothing and
// answers the question.
func (s *winSurface) winCaptionColorsWork() bool {
	if s.paletteProbed {
		return s.paletteOK
	}
	s.paletteProbed = true
	s.paletteOK = s.winDwmColor(dwmwaCaptionColor, dwmColorDefault)
	return s.paletteOK
}

// SetPalette paints the desktop's frame in the colours of a KDE
// colour-scheme file; "" gives them back to the desktop.
func (s *winSurface) SetPalette(path string) bool {
	if s == nil || s.hwnd == 0 || s.closed || !s.winCaptionColorsWork() {
		return false
	}
	if path == "" {
		ok := s.winDwmColor(dwmwaCaptionColor, dwmColorDefault)
		ok = s.winDwmColor(dwmwaTextColor, dwmColorDefault) && ok
		return s.winDwmColor(dwmwaBorderColor, dwmColorDefault) && ok
	}
	wm, err := winReadSchemeWM(path)
	if err != nil {
		return false
	}
	ok := false
	if c, found := wm["activeBackground"]; found {
		ok = s.winDwmColor(dwmwaCaptionColor, c) || ok
		// The border takes the caption's colour: KDE's scheme has a
		// "frame" key but it names the frame a *toolkit* draws, and on
		// Windows the border is part of the caption's own look.
		s.winDwmColor(dwmwaBorderColor, c)
	}
	if c, found := wm["activeForeground"]; found {
		ok = s.winDwmColor(dwmwaTextColor, c) || ok
	}
	return ok
}

// winReadSchemeWM is the [WM] group of a colour-scheme file as COLORREFs.
func winReadSchemeWM(path string) (map[string]uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]uint32{}
	for k, v := range parseINI(f)["WM"] {
		if c, ok := winColorRef(v); ok {
			out[k] = c
		}
	}
	return out, nil
}

// winColorRef turns KDE's "r,g,b" into a COLORREF, which is 0x00BBGGRR —
// the bytes the other way round from every other colour on this
// platform, and the single easiest thing to get wrong here.
func winColorRef(s string) (uint32, bool) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) < 3 {
		return 0, false
	}
	var c [3]uint32
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || n < 0 || n > 255 {
			return 0, false
		}
		c[i] = uint32(n)
	}
	return c[2]<<16 | c[1]<<8 | c[0], true
}
