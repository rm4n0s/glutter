package widgets

// ForceUnlockHistory unlocks the navigator whatever the state of the change of
// its history. A test that trips a check inside the change leaves the navigator
// locked, as Flutter does (it never unlocks, so that the object becomes unusable),
// and needs it unlocked for the Dispose at the end of the test.
func (s *NavigatorState) ForceUnlockHistory() { s.debugLocked = false }

// NewWindowBarForTest makes the part of the title bar before the window buttons,
// so that a test can put it under another text direction (the bar of a
// WindowDecoration has its own, left to right).
func NewWindowBarForTest(gap, title Widget, leading, actions []Widget, height float64) Widget {
	return newWindowBar(gap, title, leading, actions, height)
}
