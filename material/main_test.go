package material_test

import (
	"os"
	"testing"

	"github.com/rm4n0s/glutter/foundation"
)

// TestMain runs the tests of the package as Linux on an Apple host. Many tests make their ThemeData
// (typography by platform) before gluttertest.New, whose own pin (SPEC §12.10) would come too late for
// it. A test of an Apple branch sets the platform itself and puts back what it found.
// applePinned is true when TestMain made the platform Linux because the host is an Apple one: a test
// that asks whether it chose a platform (newToolbarTester) treats that Linux as no choice unless
// platformChosen says the test asked for it.
var applePinned bool

// platformChosen is true while a test has chosen its platform with withPlatform.
var platformChosen bool

func TestMain(m *testing.M) {
	switch foundation.DefaultTargetPlatform() {
	case foundation.TargetPlatformMacOS, foundation.TargetPlatformIOS:
		if foundation.DebugMode && foundation.DebugDefaultTargetPlatformOverride().IsNone() {
			foundation.SetDebugDefaultTargetPlatformOverride(foundation.Some(foundation.TargetPlatformLinux))
			applePinned = true
		}
	}
	os.Exit(m.Run())
}
