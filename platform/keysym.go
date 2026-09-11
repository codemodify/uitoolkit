package platform

// KeyFromKeysym maps an X11 / xkbcommon keysym to a toolkit Key.
func KeyFromKeysym(ks uint64) Key {
	switch ks {
	case 0xff1b:
		return KeyEscape
	case 0xff09:
		return KeyTab
	case 0xff0d, 0xff8d:
		return KeyReturn
	case 0xff08:
		return KeyBackspace
	case 0xffff, 0xff9f:
		return KeyDelete
	case 0xff51, 0xff96:
		return KeyLeft
	case 0xff53, 0xff98:
		return KeyRight
	case 0xff52, 0xff97:
		return KeyUp
	case 0xff54, 0xff99:
		return KeyDown
	case 0xff50, 0xff95:
		return KeyHome
	case 0xff57, 0xff9c:
		return KeyEnd
	case 0xff55, 0xff9a:
		return KeyPageUp
	case 0xff56, 0xff9b:
		return KeyPageDown
	case 0x0020:
		return KeySpace
	case 0x0033:
		return Key3
	case 0x0023:
		return KeyHash
	case 0x002c:
		return KeyComma
	case 0xffbe:
		return KeyF1
	case 0xffbf:
		return KeyF2
	case 0xffc0:
		return KeyF3
	case 0xffc1:
		return KeyF4
	case 0xffc2:
		return KeyF5
	case 0xffc3:
		return KeyF6
	case 0xffc4:
		return KeyF7
	case 0xffc5:
		return KeyF8
	case 0xffc6:
		return KeyF9
	case 0xffc7:
		return KeyF10
	case 0xffc8:
		return KeyF11
	case 0xffc9:
		return KeyF12
	}
	if ks >= 'a' && ks <= 'z' {
		return KeyA + Key(ks-'a')
	}
	if ks >= 'A' && ks <= 'Z' {
		return KeyA + Key(ks-'A')
	}
	return KeyUnknown
}
