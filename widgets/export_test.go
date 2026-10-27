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

// OverlayChildModelLen is the number of locations in the child model of the overlay entry that
// bc is in: one for each OverlayPortal below it that has an overlay child.
func OverlayChildModelLen(bc BuildContext) int {
	n := 0
	for l := renderTheaterMarkerOf(bc, false).overlayEntryWidgetState.firstSibling; l != nil; l = l.next {
		n++
	}
	return n
}
