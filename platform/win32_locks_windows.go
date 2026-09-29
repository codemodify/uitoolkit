//go:build windows

package platform

// LockKeys is the lock keys' state now ([LockKeysSurface]).
//
// GetKeyState's *low* bit is the toggle; the high bit says the key is
// held, which for a lock is the wrong question.
func (s *winSurface) LockKeys() (caps, num bool) {
	toggled := func(vk uintptr) bool {
		r, _, _ := procGetKeyState.Call(vk)
		return r&1 != 0
	}
	return toggled(vkCapital), toggled(vkNumLock)
}

var _ LockKeysSurface = (*winSurface)(nil)
