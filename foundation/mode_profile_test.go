//go:build glutter_profile && !glutter_release

package foundation

import "testing"

func TestProfileMode(t *testing.T) {
	if DebugMode || !ProfileMode || ReleaseMode {
		t.Fatalf("modes: debug %v profile %v release %v", DebugMode, ProfileMode, ReleaseMode)
	}
}
