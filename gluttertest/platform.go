package gluttertest

import "github.com/rm4n0s/glutter/foundation"

// pinApplePlatform makes a test run as Linux when the host is an Apple platform and the test has chosen
// none: the framework's branches by platform (typography, scroll physics, shortcuts, the text selection
// toolbar) are those of foundation.DefaultTargetPlatform, which on macOS and iOS is Apple's, and the
// suite and its goldens were written for the desktop that the module began on. A test on a Mac then
// gives the same result as on Linux (SPEC §12.10); a test of an Apple branch sets the platform itself
// (foundation.SetDebugDefaultTargetPlatformOverride), before or after New, and keeps it.
//
// The pin is removed when the test ends, unless the test changed the override in the meantime.
func pinApplePlatform(tb TB) {
	tb.Helper()
	pinPlatformFor(tb, foundation.DefaultTargetPlatform())
}

// pinPlatformFor is pinApplePlatform for a host platform that the caller names (a test of it names each).
func pinPlatformFor(tb TB, host foundation.TargetPlatform) {
	tb.Helper()
	if !foundation.DebugMode || foundation.DebugDefaultTargetPlatformOverride().IsSome() {
		return
	}
	switch host {
	case foundation.TargetPlatformMacOS, foundation.TargetPlatformIOS:
	default:
		return
	}
	foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
	tb.Cleanup(func() {
		if p := foundation.DebugDefaultTargetPlatformOverride(); p.IsSome() && p.Value() == foundation.TargetPlatformLinux {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.None[foundation.TargetPlatform]())
		}
	})
}
