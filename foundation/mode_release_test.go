//go:build glutter_release

package foundation

import (
	"testing"

	"github.com/rm4n0s/errors"
)

func TestReleaseMode(t *testing.T) {
	if DebugMode || ProfileMode || !ReleaseMode {
		t.Fatalf("modes: debug %v profile %v release %v", DebugMode, ProfileMode, ReleaseMode)
	}
	err := recoverError(t, func() { SetDebugDefaultTargetPlatformOverride(Some(TargetPlatformMacOS)) })
	appErr, ok := errors.FromError(err)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if !appErr.HasRoute("SetDebugDefaultTargetPlatformOverride.TargetPlatformOverrideNotDebug") {
		t.Errorf("unexpected failure path: %s", appErr.Route())
	}
	if got := DefaultTargetPlatform(); got != TargetPlatformLinux {
		t.Errorf("default = %v", got)
	}
	// Diagnostics render nothing in release builds.
	tree := &TestTree{properties: []DiagnosticsNode{NewIntProperty("n", Some(1))}}
	if got := ToStringDeep(tree); got != "" {
		t.Errorf("ToStringDeep = %q, want empty", got)
	}
	// A disposed notifier is not checked.
	n := &testNotifier{}
	n.Dispose()
	n.AddListener(func() {})
}
