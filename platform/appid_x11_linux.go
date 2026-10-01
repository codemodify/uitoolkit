//go:build linux && !nox11

package platform

/*
#include <stdlib.h>
*/
import "C"

import "unsafe"

// cAppClass is the pair X11 carries in WM_CLASS: the instance name, which
// is the application's id, and the class, which is it capitalized. A
// desktop matches a window to its .desktop file and its window rules on
// these, so two programs built with this toolkit must not share them.
//
// The caller frees both with freeAppClass.
func cAppClass() (*C.char, *C.char) {
	return C.CString(AppID()), C.CString(AppClass())
}

func freeAppClass(name, class *C.char) {
	C.free(unsafe.Pointer(name))
	C.free(unsafe.Pointer(class))
}
