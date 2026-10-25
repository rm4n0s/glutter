package widgets

// ForceUnlockHistory unlocks the navigator whatever the state of the change of
// its history. A test that trips a check inside the change leaves the navigator
// locked, as Flutter does (it never unlocks, so that the object becomes unusable),
// and needs it unlocked for the Dispose at the end of the test.
func (s *NavigatorState) ForceUnlockHistory() { s.debugLocked = false }
